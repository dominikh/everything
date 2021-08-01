// Package sparql implements a parser for SPARQL 1.0
//
// The parsed representation is not true to the source form,
// as various normalizations take place during parsing:
//
// - compact forms of writing triples with repeated elements will be expanded during parsing
// - numeric and boolean literals are transformed into typed literals, e.g. -5.0 becomes "-5.0"^^xsd:decimal
package main

import (
	"fmt"
	"regexp"

	"honnef.co/go/everything/sparql/ast"

	"github.com/shurcooL/go-goon"
)

// FIXME SPARQL is case insensitive

// TODO remove duplication between predictors and parsers.

/*
TODOs:
   - A.2
*/

type token struct {
	value string
	kind  string
}

// OPT(dh): use an efficient lexer
func Lex(in string) []token {
	var tokens []token
	emit := func(s, kind string) {
		if kind == "whitespace" || kind == "comment" {
			return
		}
		tokens = append(tokens, token{s, kind})
	}

	for len(in) > 0 {
		var bestMatch struct {
			s    string
			kind string
		}
		for _, p := range patterns {
			if m := p.r.FindString(in); m != "" {
				if len(m) > len(bestMatch.s) {
					bestMatch.s = m
					bestMatch.kind = p.label
				}
			}
		}
		if bestMatch.kind != "" {
			emit(bestMatch.s, bestMatch.kind)
			in = in[len(bestMatch.s):]
		} else {
			panic(fmt.Sprintf("no token found at %q", in[0:10]))
		}
	}

	return tokens
}

type iter struct {
	blankID int
	tokens  []token
}

func (iter *iter) newBlankNode() ast.BlankNode {
	iter.blankID++
	return ast.BlankNode{ID: iter.blankID}
}

func (iter *iter) next() token {
	tok := iter.tokens[0]
	iter.tokens = iter.tokens[1:]
	return tok
}

func (iter *iter) peek() token {
	if len(iter.tokens) == 0 {
		return token{kind: "EOF"}
	} else {
		return iter.tokens[0]
	}
}

func expect(tok token, kind string) token {
	if tok.kind != kind {
		panic(fmt.Sprintf("%q != %q", tok.kind, kind))
	}
	return tok
}

func predictVar(tok token) bool                      { return tok.kind == "VAR1" || tok.kind == "VAR2" }
func predictOrderClause(tok token) bool              { return tok.value == "ORDER" }
func predictLimitClause(tok token) bool              { return tok.value == "LIMIT" }
func predictOffsetClause(tok token) bool             { return tok.value == "OFFSET" }
func predictBrackettedExpression(tok token) bool     { return tok.value == "(" }
func predictRegexExpression(tok token) bool          { return tok.value == "REGEX" }
func predictIRIrefOrFunction(tok token) bool         { return predictIRIref(tok) }
func predictPrefixedName(tok token) bool             { return tok.kind == "PNAME_LN" || tok.kind == "PNAME_NS" }
func predictBaseDecl(tok token) bool                 { return tok.value == "BASE" }
func predictPrefixDecl(tok token) bool               { return tok.value == "PREFIX" }
func predictDatasetClause(tok token) bool            { return tok.value == "FROM" }
func predictTriplesBlock(tok token) bool             { return predictTriplesSameSubject(tok) }
func predictTriplesSameSubject(tok token) bool       { return predictVarOrTerm(tok) }
func predictVarOrTerm(tok token) bool                { return predictVar(tok) || predictGraphTerm(tok) }
func predictRDFLiteral(tok token) bool               { return predictString(tok) }
func predictBooleanLiteral(tok token) bool           { return tok.value == "true" || tok.value == "false" }
func predictBlankNode(tok token) bool                { return tok.kind == "BLANK_NODE_LABEL" || tok.kind == "ANON" }
func predictVarOrIRIref(tok token) bool              { return predictVar(tok) || predictIRIref(tok) }
func predictCollection(tok token) bool               { return tok.value == "(" }
func predictBlankNodePropertyList(tok token) bool    { return tok.value == "[" }
func predictGraphNode(tok token) bool                { return predictVarOrTerm(tok) || predictTriplesNode(tok) }
func predictVerb(tok token) bool                     { return predictVarOrIRIref(tok) || tok.value == "a" }
func predictPropertyListNotEmpty(tok token) bool     { return predictVerb(tok) }
func predictFunctionCall(tok token) bool             { return predictIRIref(tok) }
func predictConstructTriples(tok token) bool         { return predictTriplesSameSubject(tok) }
func predictOptionalGraphPattern(tok token) bool     { return tok.value == "OPTIONAL" }
func predictGroupOrUnionGraphPattern(tok token) bool { return predictGroupGraphPattern(tok) }
func predictGraphGraphPattern(tok token) bool        { return tok.value == "GRAPH" }
func predictGroupGraphPattern(tok token) bool        { return tok.value == "{" }
func predictFilter(tok token) bool                   { return tok.value == "FILTER" }
func predictDefaultGraphClause(tok token) bool       { return predictSourceSelector(tok) }
func predictNamedGraphClause(tok token) bool         { return tok.value == "NAMED" }
func predictSourceSelector(tok token) bool           { return predictIRIref(tok) }

func predictTriplesNode(tok token) bool {
	return predictCollection(tok) || predictBlankNodePropertyList(tok)
}

func predictIRIref(tok token) bool {
	if tok.kind == "IRI_REF" {
		return true
	}
	return predictPrefixedName(tok)
}

func predictGraphTerm(tok token) bool {
	return predictIRIref(tok) || predictRDFLiteral(tok) || predictNumericLiteral(tok) || predictBooleanLiteral(tok) || predictBlankNode(tok) || tok.kind == "NIL"
}

func predictNumericLiteral(tok token) bool {
	return predictNumericLiteralUnsigned(tok) || predictNumericLiteralPositive(tok) || predictNumericLiteralNegative(tok)
}

func predictNumericLiteralUnsigned(tok token) bool {
	switch tok.kind {
	case "INTEGER", "DECIMAL", "DOUBLE":
		return true
	default:
		return false
	}
}

func predictNumericLiteralPositive(tok token) bool {
	switch tok.kind {
	case "INTEGER_POSITIVE", "DECIMAL_POSITIVE", "DOUBLE_POSITIVE":
		return true
	default:
		return false
	}
}

func predictNumericLiteralNegative(tok token) bool {
	switch tok.kind {
	case "INTEGER_NEGATIVE", "DECIMAL_NEGATIVE", "DOUBLE_NEGATIVE":
		return true
	default:
		return false
	}
}

func predictString(tok token) bool {
	switch tok.kind {
	case "STRING_LITERAL1", "STRING_LITERAL2", "STRING_LITERAL_LONG1", "STRING_LITERAL_LONG2":
		return true
	default:
		return false
	}
}

func predictLimitOffsetClauses(tok token) bool {
	return predictLimitClause(tok) || predictOffsetClause(tok)
}

func predictPrimaryExpression(tok token) bool {
	return predictBrackettedExpression(tok) || predictBuiltInCall(tok) || predictIRIrefOrFunction(tok) || predictRDFLiteral(tok) || predictNumericLiteral(tok) || predictBooleanLiteral(tok) || predictVar(tok)
}

func predictBuiltInCall(tok token) bool {
	switch tok.value {
	case "STR", "LANG", "LANGMATCHES", "DATATYPE", "BOUND", "sameTerm", "isIRI", "isURI", "isBLANK", "isLITERAL":
		return true
	}

	return predictRegexExpression(tok)
}

func predictArgList(tok token) bool {
	return tok.kind == "NIL" || tok.value == "("
}

func predictOrderCondition(tok token) bool {
	return tok.value == "ASC" || tok.value == "DESC" || predictConstraint(tok) || predictVar(tok)
}

func predictConstraint(tok token) bool {
	return predictBrackettedExpression(tok) || predictBuiltInCall(tok) || predictFunctionCall(tok)
}

func predictWhereClause(tok token) bool {
	return tok.value == "WHERE" || predictGroupGraphPattern(tok)
}

func predictGraphPatternNotTriples(tok token) bool {
	return predictOptionalGraphPattern(tok) || predictGroupOrUnionGraphPattern(tok) || predictGraphGraphPattern(tok)
}

func GraphNode(iter *iter) ast.Node {
	tok := iter.peek()
	if predictVarOrTerm(tok) {
		return VarOrTerm(iter)
	} else if predictTriplesNode(tok) {
		return TriplesNode(iter)
	} else {
		panic(fmt.Sprintf("unexpected token %v", tok))
	}
}

func VarOrTerm(iter *iter) ast.Node {
	if predictVar(iter.peek()) {
		return Var(iter)
	} else {
		return GraphTerm(iter)
	}
}

func Var(iter *iter) ast.Var {
	tok := iter.next()
	if tok.kind != "VAR1" && tok.kind != "VAR2" {
		panic(fmt.Sprintf("unexpected token %v", tok))
	}
	return ast.Var(tok.value[1:])
}

func GraphTerm(iter *iter) ast.Node {
	tok := iter.peek()
	if predictIRIref(tok) {
		return IRIref(iter)
	} else if predictRDFLiteral(tok) {
		return RDFLiteral(iter)
	} else if predictNumericLiteral(tok) {
		return NumericLiteral(iter)
	} else if predictBooleanLiteral(tok) {
		return BooleanLiteral(iter)
	} else if predictBlankNode(tok) {
		return BlankNode(iter)
	} else if tok.kind == "NIL" {
		iter.next()
		return ast.IRI("http://www.w3.org/1999/02/22-rdf-syntax-ns#nil")
	} else {
		panic(fmt.Sprintf("unexpected token %v", tok))
	}
}

func BaseDecl(iter *iter) ast.IRI {
	requireValue(iter, "BASE")

	return ast.IRI(requireKind(iter, "IRI_REF").value)
}

func PrefixDecl(iter *iter) ast.Prefix {
	iter.next()
	return ast.Prefix{
		Name: expect(iter.next(), "PNAME_NS").value,
		IRI:  ast.IRI(expect(iter.next(), "IRI_REF").value),
	}
}

func Prologue(iter *iter) (base ast.IRI, prefixes []ast.Prefix) {
	tok := iter.peek()
	if predictBaseDecl(tok) {
		base = BaseDecl(iter)
	}
	for tok := iter.peek(); predictPrefixDecl(tok); tok = iter.peek() {
		prefixes = append(prefixes, PrefixDecl(iter))
	}
	return
}

func DefaultGraphClause(iter *iter) ast.From {
	return ast.From{
		IRI: SourceSelector(iter),
	}
}

func NamedGraphClause(iter *iter) ast.From {
	requireValue(iter, "NAMED")
	return ast.From{
		Named: true,
		IRI:   SourceSelector(iter),
	}
}

func SourceSelector(iter *iter) ast.IRIref {
	return IRIref(iter)
}

func DatasetClause(iter *iter) ast.From {
	requireValue(iter, "FROM")

	tok := iter.peek()
	if predictDefaultGraphClause(tok) {
		return DefaultGraphClause(iter)
	} else if predictNamedGraphClause(tok) {
		return NamedGraphClause(iter)
	} else {
		panic(fmt.Sprintf("unexpected token %v", tok))
	}
}

func PrefixedName(iter *iter) ast.PrefixedName {
	tok := iter.peek()
	switch tok.kind {
	case "PNAME_LN":
		return ast.PrefixedName(iter.next().value)
	case "PNAME_NS":
		return ast.PrefixedName(iter.next().value)
	default:
		panic(fmt.Sprintf("unexpected token %v", tok))
	}
}

func IRIref(iter *iter) ast.IRIref {
	tok := iter.peek()
	if tok.kind == "IRI_REF" {
		return ast.IRI(iter.next().value)
	} else if predictPrefixedName(tok) {
		return PrefixedName(iter)
	} else {
		panic(fmt.Sprintf("unexpected token %v", tok))
	}
}

func String(iter *iter) string {
	tok := iter.peek()
	switch tok.kind {
	case "STRING_LITERAL1", "STRING_LITERAL2", "STRING_LITERAL_LONG1", "STRING_LITERAL_LONG2":
		tok := iter.next()
		// XXX strip quotes
		return tok.value
	default:
		panic(fmt.Sprintf("unexpected token %v", tok))
	}
}

func RDFLiteral(iter *iter) ast.RDFLiteral {
	s := String(iter)
	var lang string
	var typ ast.IRIref

	tok := iter.peek()
	if tok.kind == "LANGTAG" {
		lang = iter.next().value[1:]
	} else if tok.value == "^^" {
		iter.next()
		typ = IRIref(iter)
	}

	// XXX strip surrounding quotes
	return ast.RDFLiteral{Value: s, Lang: lang, Type: typ}
}

func NumericLiteralUnsigned(iter *iter) ast.RDFLiteral {
	tok := iter.peek()
	switch tok.kind {
	case "INTEGER":
		return ast.RDFLiteral{
			Value: iter.next().value,
			Type:  ast.IRI("http://www.w3.org/2001/XMLSchema#integer"),
		}
	case "DECIMAL":
		return ast.RDFLiteral{
			Value: iter.next().value,
			Type:  ast.IRI("http://www.w3.org/2001/XMLSchema#decimal"),
		}
	case "DOUBLE":
		return ast.RDFLiteral{
			Value: iter.next().value,
			Type:  ast.IRI("http://www.w3.org/2001/XMLSchema#double"),
		}
	default:
		panic(fmt.Sprintf("unexpected token %v", tok))
	}
}

func NumericLiteralPositive(iter *iter) ast.RDFLiteral {
	tok := iter.peek()
	switch tok.kind {
	case "INTEGER_POSITIVE":
		return ast.RDFLiteral{
			Value: iter.next().value[1:],
			Type:  ast.IRI("http://www.w3.org/2001/XMLSchema#integer"),
		}
	case "DECIMAL_POSITIVE":
		return ast.RDFLiteral{
			Value: iter.next().value[1:],
			Type:  ast.IRI("http://www.w3.org/2001/XMLSchema#decimal"),
		}
	case "DOUBLE_POSITIVE":
		return ast.RDFLiteral{
			Value: iter.next().value[1:],
			Type:  ast.IRI("http://www.w3.org/2001/XMLSchema#double"),
		}
	default:
		panic(fmt.Sprintf("unexpected token %v", tok))
	}
}

func NumericLiteralNegative(iter *iter) ast.RDFLiteral {
	tok := iter.peek()
	switch tok.kind {
	case "INTEGER_NEGATIVE":
		return ast.RDFLiteral{
			Value: iter.next().value,
			Type:  ast.IRI("http://www.w3.org/2001/XMLSchema#integer"),
		}
	case "DECIMAL_NEGATIVE":
		return ast.RDFLiteral{
			Value: iter.next().value,
			Type:  ast.IRI("http://www.w3.org/2001/XMLSchema#decimal"),
		}
	case "DOUBLE_NEGATIVE":
		return ast.RDFLiteral{
			Value: iter.next().value,
			Type:  ast.IRI("http://www.w3.org/2001/XMLSchema#double"),
		}
	default:
		panic(fmt.Sprintf("unexpected token %v", tok))
	}
}

func NumericLiteral(iter *iter) ast.RDFLiteral {
	tok := iter.peek()
	if predictNumericLiteralUnsigned(tok) {
		return NumericLiteralUnsigned(iter)
	} else if predictNumericLiteralPositive(tok) {
		return NumericLiteralPositive(iter)
	} else if predictNumericLiteralNegative(tok) {
		return NumericLiteralNegative(iter)
	} else {
		panic(fmt.Sprintf("unexpected token %v", tok))
	}
}

func BooleanLiteral(iter *iter) ast.RDFLiteral {
	tok := iter.peek()
	switch tok.value {
	case "true":
		iter.next()
		return ast.RDFLiteral{
			Value: "true",
			Type:  ast.IRI("http://www.w3.org/2001/XMLSchema#boolean"),
		}
	case "false":
		iter.next()
		return ast.RDFLiteral{
			Value: "false",
			Type:  ast.IRI("http://www.w3.org/2001/XMLSchema#boolean"),
		}
	default:
		panic(fmt.Sprintf("unexpected token %v", tok))
	}
}

func BlankNode(iter *iter) ast.Node {
	tok := iter.peek()
	switch tok.kind {
	case "BLANK_NODE_LABEL":
		iter.next()
		return iter.newBlankNode()
	case "ANON":
		tok := iter.next()
		return ast.NamedBlankNode{
			Name: tok.value,
		}
	default:
		panic(fmt.Sprintf("unexpected token %v", tok))
	}
}

func VarOrIRIref(iter *iter) ast.Node {
	tok := iter.peek()
	if predictVar(tok) {
		return Var(iter)
	} else if predictIRIref(tok) {
		return IRIref(iter)
	} else {
		panic(fmt.Sprintf("unexpected token %v", tok))
	}
}

func Verb(iter *iter) ast.Node {
	tok := iter.peek()
	if predictVarOrIRIref(tok) {
		return VarOrIRIref(iter)
	} else if tok.value == "a" {
		iter.next()
		return ast.IRI("http://www.w3.org/1999/02/22-rdf-syntax-ns#type")
	} else {
		panic(fmt.Sprintf("unexpected token %v", tok))
	}
}

func Collection(iter *iter) {
	panic("collections are unsupported")

	requireValue(iter, "(")
	GraphNode(iter)
	for tok := iter.peek(); predictGraphNode(tok); tok = iter.peek() {
		GraphNode(iter)
	}
	requireValue(iter, ")")
}

func BlankNodePropertyList(iter *iter) []verbObject {
	tok := iter.next()
	if tok.value != "[" {
		panic(fmt.Sprintf("unexpected token %v", tok))
	}

	vos := PropertyListNotEmpty(iter)

	tok = iter.next()
	if tok.value != "]" {
		panic(fmt.Sprintf("unexpected token %v", tok))
	}

	return vos
}

func TriplesNode(iter *iter) interface{} {
	tok := iter.peek()
	if predictCollection(tok) {
		Collection(iter)
		panic("collections are unsupported")
	} else if predictBlankNodePropertyList(tok) {
		// The subject is a blank node, which has the following verb-objects.
		return BlankNodePropertyList(iter)
	} else {
		panic(fmt.Sprintf("unexpected token %v", tok))
	}
}

func Object(iter *iter) ast.Node {
	return GraphNode(iter)
}

func ObjectList(iter *iter) []ast.Node {
	var objects []ast.Node
	objects = append(objects, Object(iter))
	for tok := iter.peek(); tok.value == ","; tok = iter.peek() {
		iter.next()
		objects = append(objects, Object(iter))
	}
	return objects
}

type verbObject struct {
	verb   ast.Node
	object ast.Node
}

func PropertyListNotEmpty(iter *iter) []verbObject {
	var vos []verbObject

	verb := Verb(iter)
	objects := ObjectList(iter)
	for _, object := range objects {
		vos = append(vos, verbObject{verb, object})
	}
	for tok := iter.peek(); tok.value == ";"; tok = iter.peek() {
		iter.next()
		if predictVerb(iter.peek()) {
			verb := Verb(iter)
			objects := ObjectList(iter)
			for _, object := range objects {
				vos = append(vos, verbObject{verb, object})
			}
		}
	}

	return vos
}

func PropertyList(iter *iter) []verbObject {
	if predictPropertyListNotEmpty(iter.peek()) {
		return PropertyListNotEmpty(iter)
	} else {
		return nil
	}
}

func TriplesSameSubject(iter *iter) []ast.Triple {
	if predictVarOrTerm(iter.peek()) {
		var triples []ast.Triple

		subject := VarOrTerm(iter)
		vos := PropertyListNotEmpty(iter)
		for _, vo := range vos {
			triples = append(triples, ast.Triple{
				Subject:   subject,
				Predicate: vo.verb,
				Object:    vo.object,
			})
		}
		return triples
	} else {
		var triples []ast.Triple
		ret := TriplesNode(iter)
		var subject ast.Node
		switch ret := ret.(type) {
		case []verbObject:
			subject = iter.newBlankNode()
			for _, vo := range ret {
				triples = append(triples, ast.Triple{Subject: subject, Predicate: vo.verb, Object: vo.object})
			}
		default:
			panic(fmt.Sprintf("unhandled type %T", ret))
		}
		vos := PropertyList(iter)
		for _, vo := range vos {
			triples = append(triples, ast.Triple{Subject: subject, Predicate: vo.verb, Object: vo.object})
		}
		return triples
	}
}

func TriplesBlock(iter *iter) []ast.Triple {
	out := TriplesSameSubject(iter)
	if iter.peek().value == "." {
		iter.next()
		if predictTriplesBlock(iter.peek()) {
			out = append(out, TriplesBlock(iter)...)
		}
	}
	return out
}

func OptionalGraphPattern(iter *iter) ast.GroupGraphPattern {
	requireValue(iter, "OPTIONAL")
	ggp := GroupGraphPattern(iter)
	ggp.Optional = true
	return ggp
}

func GroupOrUnionGraphPattern(iter *iter) ast.Node {
	var patterns []ast.GroupGraphPattern
	patterns = append(patterns, GroupGraphPattern(iter))
	for tok := iter.peek(); tok.value == "UNION"; tok = iter.peek() {
		iter.next()
		patterns = append(patterns, GroupGraphPattern(iter))
	}
	if len(patterns) == 1 {
		return patterns[0]
	} else {
		return ast.Union{Patterns: patterns}
	}
}

func GraphGraphPattern(iter *iter) ast.GraphGraphPattern {
	requireValue(iter, "GRAPH")
	return ast.GraphGraphPattern{
		Name:    VarOrIRIref(iter),
		Pattern: GroupGraphPattern(iter),
	}
}

func GraphPatternNotTriples(iter *iter) ast.Node {
	tok := iter.peek()
	if predictOptionalGraphPattern(tok) {
		return OptionalGraphPattern(iter)
	} else if predictGroupOrUnionGraphPattern(tok) {
		return GroupOrUnionGraphPattern(iter)
	} else if predictGraphGraphPattern(tok) {
		return GraphGraphPattern(iter)
	} else {
		panic(fmt.Sprintf("unexpected token %v", tok))
	}
}

func Filter(iter *iter) ast.Expr {
	requireValue(iter, "FILTER")
	return Constraint(iter)
}

func GroupGraphPattern(iter *iter) ast.GroupGraphPattern {
	requireValue(iter, "{")

	root := ast.GroupGraphPattern{}
	basic := ast.BasicGraphPattern{}

	tok := iter.peek()
	if predictTriplesBlock(tok) {
		basic.Patterns = append(basic.Patterns, TriplesBlock(iter)...)
	}

	for {
		tok = iter.peek()
		if predictGraphPatternNotTriples(tok) {
			if len(basic.Patterns) > 0 {
				root.Patterns = append(root.Patterns, basic)
				basic = ast.BasicGraphPattern{}
			}

			root.Patterns = append(root.Patterns, GraphPatternNotTriples(iter))
		} else if predictFilter(tok) {
			root.Filters = append(root.Filters, Filter(iter))
		} else {
			break
		}
		optionalValue(iter, ".")
		if predictTriplesBlock(iter.peek()) {
			basic.Patterns = append(basic.Patterns, TriplesBlock(iter)...)
		}
	}

	if len(basic.Patterns) > 0 {
		root.Patterns = append(root.Patterns, basic)
	}

	requireValue(iter, "}")

	return root
}

func WhereClause(iter *iter) ast.GroupGraphPattern {
	tok := iter.peek()
	if tok.value == "WHERE" {
		iter.next()
	}

	return GroupGraphPattern(iter)
}

func requireValue(iter *iter, value string) {
	tok := iter.peek()
	if tok.value == value {
		iter.next()
	} else {
		panic(fmt.Sprintf("unexpected token %v", tok))
	}
}

func optionalValue(iter *iter, value string) bool {
	tok := iter.peek()
	if tok.value == value {
		iter.next()
		return true
	} else {
		return false
	}
}

func requireKind(iter *iter, kind string) token {
	tok := iter.peek()
	if tok.kind == kind {
		return iter.next()
	} else {
		panic(fmt.Sprintf("unexpected token %v", tok))
	}
}

func RegexExpression(iter *iter) ast.Regex {
	node := ast.Regex{}

	requireValue(iter, "REGEX")
	requireValue(iter, "(")
	node.String = Expression(iter)
	requireValue(iter, ",")
	node.Pattern = Expression(iter)
	if iter.peek().value == "," {
		iter.next()
		node.Flags = Expression(iter)
	}
	requireValue(iter, ")")

	return node
}

func BuiltInCall(iter *iter) ast.Expr {
	tok := iter.peek()
	switch tok.value {
	case "STR", "LANG", "DATATYPE", "isIRI", "isURI", "isBLANK", "isLiteral":
		fn := iter.next().value
		requireValue(iter, "(")
		arg1 := Expression(iter)
		requireValue(iter, ")")
		return ast.Call{
			Func: ast.Builtin{Name: fn},
			Args: []ast.Expr{arg1},
		}
	case "LANGMATCHES", "sameTerm":
		fn := iter.next().value
		requireValue(iter, "(")
		arg1 := Expression(iter)
		requireValue(iter, ",")
		arg2 := Expression(iter)
		requireValue(iter, ")")
		return ast.Call{
			Func: ast.Builtin{Name: fn},
			Args: []ast.Expr{arg1, arg2},
		}
	case "BOUND":
		iter.next()
		requireValue(iter, "(")
		arg1 := Var(iter)
		requireValue(iter, ")")
		return ast.Call{
			Func: ast.Builtin{Name: "BOUND"},
			Args: []ast.Expr{arg1},
		}
	default:
		if predictRegexExpression(tok) {
			return RegexExpression(iter)
		} else {
			panic(fmt.Sprintf("unexpected token %v", tok))
		}
	}
}

func ArgList(iter *iter) []ast.Expr {
	tok := iter.peek()
	if tok.kind == "NIL" {
		iter.next()
		return []ast.Expr{}
	} else if tok.value == "(" {
		iter.next()
		var args []ast.Expr
		args = append(args, Expression(iter))
		for tok := iter.peek(); tok.value == ","; tok = iter.peek() {
			iter.next()
			args = append(args, Expression(iter))
		}
		requireValue(iter, ")")
		return args
	} else {
		panic(fmt.Sprintf("unexpected token %v", tok))
	}
}

func IRIrefOrFunction(iter *iter) ast.Expr {
	ref := IRIref(iter)
	if predictArgList(iter.peek()) {
		args := ArgList(iter)
		return ast.Call{
			Func: ref,
			Args: args,
		}
	} else {
		return ref
	}
}

func PrimaryExpression(iter *iter) ast.Expr {
	tok := iter.peek()
	if predictBrackettedExpression(tok) {
		return BrackettedExpression(iter)
	} else if predictBuiltInCall(tok) {
		return BuiltInCall(iter)
	} else if predictIRIrefOrFunction(tok) {
		return IRIrefOrFunction(iter)
	} else if predictRDFLiteral(tok) {
		return RDFLiteral(iter)
	} else if predictNumericLiteral(tok) {
		return NumericLiteral(iter)
	} else if predictBooleanLiteral(tok) {
		return BooleanLiteral(iter)
	} else if predictVar(tok) {
		return Var(iter)
	} else {
		panic(fmt.Sprintf("unexpected token %v", tok))
	}
}

func UnaryExpression(iter *iter) ast.Expr {
	tok := iter.peek()
	if tok.value == "!" {
		iter.next()
		return ast.UnOp{Tok: "!", X: PrimaryExpression(iter)}
	} else if tok.value == "+" {
		iter.next()
		return ast.UnOp{Tok: "+", X: PrimaryExpression(iter)}
	} else if tok.value == "-" {
		iter.next()
		return ast.UnOp{Tok: "-", X: PrimaryExpression(iter)}
	} else if predictPrimaryExpression(tok) {
		return PrimaryExpression(iter)
	} else {
		panic(fmt.Sprintf("unexpected token %v", tok))
	}
}

func MultiplicativeExpression(iter *iter) ast.Expr {
	expr := UnaryExpression(iter)

	for {
		tok := iter.peek()
		if tok.value == "*" {
			iter.next()
			expr = ast.BinOp{X: expr, Tok: "*", Y: UnaryExpression(iter)}
		} else if tok.value == "/" {
			iter.next()
			expr = ast.BinOp{X: expr, Tok: "/", Y: UnaryExpression(iter)}
		} else {
			break
		}
	}

	return expr
}

func AdditiveExpression(iter *iter) ast.Expr {
	expr := MultiplicativeExpression(iter)

	for {
		tok := iter.peek()
		switch tok.value {
		case "+":
			iter.next()
			expr = ast.BinOp{X: expr, Tok: "+", Y: MultiplicativeExpression(iter)}
		case "-":
			iter.next()
			expr = ast.BinOp{X: expr, Tok: "-", Y: MultiplicativeExpression(iter)}
		}
		if predictNumericLiteralPositive(tok) {
			expr = ast.BinOp{X: expr, Tok: "+", Y: NumericLiteralPositive(iter)}
		} else if predictNumericLiteralNegative(tok) {
			expr = ast.BinOp{X: expr, Tok: "-", Y: NumericLiteralNegative(iter)}
		} else {
			break
		}
	}

	return expr
}

func NumericExpression(iter *iter) ast.Expr {
	return AdditiveExpression(iter)
}

func RelationalExpression(iter *iter) ast.Expr {
	expr := NumericExpression(iter)

	tok := iter.peek()
	switch tok.value {
	case "=":
		iter.next()
		return ast.BinOp{X: expr, Tok: "=", Y: NumericExpression(iter)}
	case "!=":
		iter.next()
		return ast.BinOp{X: expr, Tok: "!=", Y: NumericExpression(iter)}
	case "<":
		iter.next()
		return ast.BinOp{X: expr, Tok: "<", Y: NumericExpression(iter)}
	case ">":
		iter.next()
		return ast.BinOp{X: expr, Tok: ">", Y: NumericExpression(iter)}
	case "<=":
		iter.next()
		return ast.BinOp{X: expr, Tok: "<=", Y: NumericExpression(iter)}
	case ">=":
		iter.next()
		return ast.BinOp{X: expr, Tok: ">=", Y: NumericExpression(iter)}
	default:
		return expr
	}
}

func ValueLogical(iter *iter) ast.Expr {
	return RelationalExpression(iter)
}

func ConditionalAndExpression(iter *iter) ast.Expr {
	expr := ValueLogical(iter)
	for tok := iter.peek(); tok.value == "&&"; tok = iter.peek() {
		iter.next()
		expr = ast.And{X: expr, Y: ValueLogical(iter)}
	}
	return expr
}

func ConditionalOrExpression(iter *iter) ast.Expr {
	expr := ConditionalAndExpression(iter)
	for tok := iter.peek(); tok.value == "||"; tok = iter.peek() {
		iter.next()
		expr = ast.Or{X: expr, Y: ConditionalAndExpression(iter)}
	}
	return expr
}

func Expression(iter *iter) ast.Node {
	return ConditionalOrExpression(iter)
}

func BrackettedExpression(iter *iter) ast.Expr {
	requireValue(iter, "(")
	expr := Expression(iter)
	requireValue(iter, ")")
	return expr
}

func FunctionCall(iter *iter) ast.Expr {
	IRIref(iter)
	ArgList(iter)
	panic("not implemented")
}

func Constraint(iter *iter) ast.Expr {
	tok := iter.peek()
	if predictBrackettedExpression(tok) {
		return BrackettedExpression(iter)
	} else if predictBuiltInCall(tok) {
		return BuiltInCall(iter)
	} else if predictFunctionCall(tok) {
		return FunctionCall(iter)
	} else {
		panic(fmt.Sprintf("unexpected token %v", tok))
	}
}

func OrderCondition(iter *iter) ast.OrderCondition {
	tok := iter.peek()
	if tok.value == "ASC" || tok.value == "DESC" {
		dir := iter.next().value
		expr := BrackettedExpression(iter)
		return ast.OrderCondition{
			Direction:  dir,
			Expression: expr,
		}
	} else if predictConstraint(tok) {
		return ast.OrderCondition{Expression: Constraint(iter)}
	} else if predictVar(tok) {
		return ast.OrderCondition{Expression: Var(iter)}
	} else {
		panic(fmt.Sprintf("unexpected token %v", tok))
	}
}

func OrderClause(iter *iter) []ast.OrderCondition {
	requireValue(iter, "ORDER")
	requireValue(iter, "BY")

	var out []ast.OrderCondition

	out = append(out, OrderCondition(iter))
	for tok := iter.peek(); predictOrderCondition(tok); tok = iter.peek() {
		out = append(out, OrderCondition(iter))
	}
	return out
}

func LimitClause(iter *iter) ast.RDFLiteral {
	requireValue(iter, "LIMIT")
	return ast.RDFLiteral{
		Value: requireKind(iter, "INTEGER").value,
		Type:  "http://www.w3.org/2001/XMLSchema#integer",
	}
}

func OffsetClause(iter *iter) ast.RDFLiteral {
	requireValue(iter, "OFFSET")
	return ast.RDFLiteral{
		Value: requireKind(iter, "INTEGER").value,
		Type:  "http://www.w3.org/2001/XMLSchema#integer",
	}
}

func LimitOffsetClauses(iter *iter) (limit, offset ast.RDFLiteral) {
	tok := iter.peek()
	if predictLimitClause(tok) {
		limit = LimitClause(iter)
		if predictOffsetClause(iter.peek()) {
			offset = OffsetClause(iter)
		}
	} else if predictOffsetClause(tok) {
		offset = OffsetClause(iter)
		if predictLimitClause(iter.peek()) {
			limit = LimitClause(iter)
		}
	} else {
		panic(fmt.Sprintf("unexpected token %v", tok))
	}
	return
}

func SolutionModifier(iter *iter) ast.SolutionModifier {
	var out ast.SolutionModifier
	if predictOrderClause(iter.peek()) {
		out.Order = OrderClause(iter)
	}
	if predictLimitOffsetClauses(iter.peek()) {
		out.Limit, out.Offset = LimitOffsetClauses(iter)
	}
	return out
}

func SelectQuery(iter *iter) ast.SelectQuery {
	query := ast.SelectQuery{}

	iter.next()
	tok := iter.peek()
	if tok.value == "DISTINCT" {
		iter.next()
		query.Distinct = true
	} else if tok.value == "REDUCED" {
		iter.next()
		query.Reduced = true
	}

	tok = iter.peek()
	if predictVar(tok) {
		for tok := iter.peek(); predictVar(tok); tok = iter.peek() {
			query.Variables = append(query.Variables, Var(iter))
		}
	} else if tok.value == "*" {
		iter.next()
		query.VarStar = true
	} else {
		panic(fmt.Sprintf("unexpected token %v", tok))
	}

	for tok := iter.peek(); predictDatasetClause(tok); tok = iter.peek() {
		query.Froms = append(query.Froms, DatasetClause(iter))
	}

	query.Where = WhereClause(iter)
	query.SolutionModifier = SolutionModifier(iter)

	return query
}

func ConstructTriples(iter *iter) []ast.Triple {
	var out []ast.Triple
	out = append(out, TriplesSameSubject(iter)...)
	if iter.peek().value == "." {
		iter.next()
		if predictConstructTriples(iter.peek()) {
			out = append(out, ConstructTriples(iter)...)
		}
	}
	return out
}

func ConstructTemplate(iter *iter) ast.Node {
	requireValue(iter, "{")
	var out ast.Node
	if predictConstructTriples(iter.peek()) {
		out = ConstructTriples(iter)
	}
	requireValue(iter, "}")
	return out
}

func ConstructQuery(iter *iter) ast.ConstructQuery {
	requireValue(iter, "CONSTRUCT")

	query := ast.ConstructQuery{}
	query.Template = ConstructTemplate(iter)
	for tok := iter.peek(); predictDatasetClause(tok); tok = iter.peek() {
		query.Froms = append(query.Froms, DatasetClause(iter))
	}
	query.Where = WhereClause(iter)
	query.SolutionModifier = SolutionModifier(iter)
	return query
}

func DescribeQuery(iter *iter) ast.DescribeQuery {
	var query ast.DescribeQuery

	requireValue(iter, "DESCRIBE")
	tok := iter.peek()
	if predictVarOrIRIref(tok) {
		for tok := iter.peek(); predictVarOrIRIref(tok); tok = iter.peek() {
			query.Variables = append(query.Variables, VarOrIRIref(iter))
		}
	} else if tok.value == "*" {
		iter.next()
		query.VarStar = true
	} else {
		panic(fmt.Sprintf("unexpected token %v", tok))
	}
	for tok := iter.peek(); predictDatasetClause(tok); tok = iter.peek() {
		query.Froms = append(query.Froms, DatasetClause(iter))
	}
	if predictWhereClause(iter.peek()) {
		query.Where = WhereClause(iter)
	}
	query.SolutionModifier = SolutionModifier(iter)
	return query
}

func AskQuery(iter *iter) ast.AskQuery {
	var query ast.AskQuery

	requireValue(iter, "ASK")
	for tok := iter.peek(); predictDatasetClause(tok); tok = iter.peek() {
		query.Froms = append(query.Froms, DatasetClause(iter))
	}
	query.Where = WhereClause(iter)
	return query
}

func Parse(in string) ast.Query {
	iter := &iter{tokens: Lex(in)}

	query := ast.Query{}
	query.Base, query.Prefixes = Prologue(iter)
	switch tok := iter.peek(); tok.value {
	case "SELECT":
		query.Query = SelectQuery(iter)
	case "CONSTRUCT":
		query.Query = ConstructQuery(iter)
	case "DESCRIBE":
		query.Query = DescribeQuery(iter)
	case "ASK":
		query.Query = AskQuery(iter)
	default:
		panic(fmt.Sprintf("unexpected token %v", tok))
	}

	return query
}

type pattern struct {
	r     *regexp.Regexp
	label string
}

var patterns = []pattern{
	{regexp.MustCompile("^" + IRI_REF), "IRI_REF"},
	{regexp.MustCompile("^" + PNAME_LN), "PNAME_LN"},
	{regexp.MustCompile("^" + PNAME_NS), "PNAME_NS"},
	{regexp.MustCompile("^" + BLANK_NODE_LABEL), "BLANK_NODE_LABEL"},
	{regexp.MustCompile("^" + VAR1), "VAR1"},
	{regexp.MustCompile("^" + VAR2), "VAR2"},
	{regexp.MustCompile("^" + LANGTAG), "LANGTAG"},
	{regexp.MustCompile("^" + INTEGER), "INTEGER"},
	{regexp.MustCompile("^" + DECIMAL), "DECIMAL"},
	{regexp.MustCompile("^" + DOUBLE), "DOUBLE"},
	{regexp.MustCompile("^" + INTEGER_POSITIVE), "INTEGER_POSITIVE"},
	{regexp.MustCompile("^" + DECIMAL_POSITIVE), "DECIMAL_POSITIVE"},
	{regexp.MustCompile("^" + DOUBLE_POSITIVE), "DOUBLE_POSITIVE"},
	{regexp.MustCompile("^" + INTEGER_NEGATIVE), "INTEGER_NEGATIVE"},
	{regexp.MustCompile("^" + DECIMAL_NEGATIVE), "DECIMAL_NEGATIVE"},
	{regexp.MustCompile("^" + DOUBLE_NEGATIVE), "DOUBLE_NEGATIVE"},
	{regexp.MustCompile("^" + STRING_LITERAL1), "STRING_LITERAL1"},
	{regexp.MustCompile("^" + STRING_LITERAL2), "STRING_LITERAL2"},
	{regexp.MustCompile("^" + STRING_LITERAL_LONG1), "STRING_LITERAL_LONG1"},
	{regexp.MustCompile("^" + STRING_LITERAL_LONG2), "STRING_LITERAL_LONG2"},
	{regexp.MustCompile("^" + NIL), "NIL"},
	{regexp.MustCompile("^" + ANON), "ANON"},
	{regexp.MustCompile(`^\s+`), "whitespace"},
	{regexp.MustCompile("^#.*"), "comment"},
	{regexp.MustCompile(`^(\W|\w+)`), "word"},
}

// const IRI_REF = "(<([^<>\"{}\\|^`\\]-[\\x00-\\x20])*>)"
const IRI_REF = "(<([^ <>])*>)"
const PNAME_NS = "(" + PN_PREFIX + "?:)"
const PNAME_LN = "(" + PNAME_NS + PN_LOCAL + ")"
const BLANK_NODE_LABEL = "(_:" + PN_LOCAL + ")"
const VAR1 = "(\\?" + VARNAME + ")"
const VAR2 = "($" + VARNAME + ")"
const LANGTAG = "(@[a-zA-Z]+(-[a-zA-Z0-9]+)*)"
const INTEGER = "([0-9]+)"
const DECIMAL = "([0-9]+\\.[0-9]*|\\.[0-9]+)"
const DOUBLE = "([0-9]+\\.[0-9]*" + EXPONENT + "|\\.([0-9])+" + EXPONENT + "|([0-9])+" + EXPONENT + ")"
const INTEGER_POSITIVE = "(\\+" + INTEGER + ")"
const DECIMAL_POSITIVE = "(\\+" + DECIMAL + ")"
const DOUBLE_POSITIVE = "(\\+" + DOUBLE + ")"
const INTEGER_NEGATIVE = "(-" + INTEGER + ")"
const DECIMAL_NEGATIVE = "(-" + DECIMAL + ")"
const DOUBLE_NEGATIVE = "(-" + DOUBLE + ")"
const EXPONENT = "([eE][+-]?[0-9]+)"
const STRING_LITERAL1 = "('(([^\\x27\\x5C\\x0A\\x0D])|" + ECHAR + ")*?')"
const STRING_LITERAL2 = `("(([^'\x5C\n\r])|` + ECHAR + `)*?")`
const STRING_LITERAL_LONG1 = "('''(('|'')?([^'\\\\]|" + ECHAR + "))*?''')"
const STRING_LITERAL_LONG2 = "(\"\"\"((\"|\"\")?([^\"\\\\]|" + ECHAR + "))*?\"\"\")"
const ECHAR = `(\\[tbnrf"'])`
const NIL = "(\\(" + WS + "*\\))"
const WS = "(\\x20|\\x09|\\x0D|\\x0A)"
const ANON = "(\\[" + WS + "*\\])"
const PN_CHARS_BASE = `([A-Z]|[a-z]|[\x{00C0}-\x{00D6}]|[\x{00D8}-\x{00F6}]|[\x{00F8}-\x{02FF}]|[\x{0370}-\x{037D}]|[\x{037F}-\x{1FFF}]|[\x{200C}-\x{200D}]|[\x{2070}-\x{218F}]|[\x{2C00}-\x{2FEF}]|[\x{3001}-\x{D7FF}]|[\x{F900}-\x{FDCF}]|[\x{FDF0}-\x{FFFD}]|[\x{10000}-\x{EFFFF}])`
const PN_CHARS_U = "(" + PN_CHARS_BASE + "|_)"
const VARNAME = "((" + PN_CHARS_U + "|[0-9])(" + PN_CHARS_U + "|[0-9]|\\x{00B7}|[\\x{0300}-\\x{036F}]|[\\x{203F}-\\x{2040}])*)"
const PN_CHARS = "(" + PN_CHARS_U + "|-|[0-9]|\\x{00B7}|[\\x{0300}-\\x{036F}]|[\\x{203F}-\\x{2040}])"
const PN_PREFIX = "(" + PN_CHARS_BASE + "((" + PN_CHARS + "|\\.)*" + PN_CHARS + ")?)"
const PN_LOCAL = "((" + PN_CHARS_U + "|[0-9])((" + PN_CHARS + "|\\.)*" + PN_CHARS + ")?)"

func main() {
	in := `PREFIX foaf: <http://xmlns.com/foaf/0.1/>
PREFIX dc:   <http://purl.org/dc/elements/1.1/>

SELECT ?name ?mbox ?date
WHERE
  {  ?g dc:publisher ?name ;
        dc:date ?date .
    GRAPH ?g
      { ?person foaf:name ?name ; foaf:mbox ?mbox }
  }`
	iter := Parse(in)

	goon.Dump(iter)
}
