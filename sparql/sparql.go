// Package sparql implements a parser for SPARQL 1.0
//
// The parsed representation is not true to the source form,
// as various normalizations take place during parsing:
//
// - compact forms of writing triples with repeated elements will be expanded during parsing
// - numeric and boolean literals are transformed into typed literals, e.g. -5.0 becomes "-5.0"^^xsd:decimal
// - comments are not preserved
package main

import (
	"fmt"

	"honnef.co/go/everything/sparql/ast"
	"honnef.co/go/everything/sparql/scanner"

	"github.com/shurcooL/go-goon"
)

// FIXME SPARQL is case insensitive

// TODO remove duplication between predictors and parsers.

type token = scanner.Token

type iter struct {
	blankID int
	tokens  []scanner.Token
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
		return token{Kind: scanner.EOF}
	} else {
		return iter.tokens[0]
	}
}

func predictVar(tok token) bool                  { return tok.Kind == scanner.VAR1 || tok.Kind == scanner.VAR2 }
func predictOrderClause(tok token) bool          { return tok.Kind == scanner.ORDER }
func predictLimitClause(tok token) bool          { return tok.Kind == scanner.LIMIT }
func predictOffsetClause(tok token) bool         { return tok.Kind == scanner.OFFSET }
func predictBrackettedExpression(tok token) bool { return tok.Kind == scanner.LPAREN }
func predictRegexExpression(tok token) bool      { return tok.Value == "REGEX" && tok.Kind == scanner.IDENT }
func predictIRIrefOrFunction(tok token) bool     { return predictIRIref(tok) }
func predictPrefixedName(tok token) bool {
	return tok.Kind == scanner.PNAME_LN || tok.Kind == scanner.PNAME_NS
}
func predictBaseDecl(tok token) bool           { return tok.Kind == scanner.BASE }
func predictPrefixDecl(tok token) bool         { return tok.Kind == scanner.PREFIX }
func predictDatasetClause(tok token) bool      { return tok.Kind == scanner.FROM }
func predictTriplesBlock(tok token) bool       { return predictTriplesSameSubject(tok) }
func predictTriplesSameSubject(tok token) bool { return predictVarOrTerm(tok) }
func predictVarOrTerm(tok token) bool          { return predictVar(tok) || predictGraphTerm(tok) }
func predictRDFLiteral(tok token) bool         { return predictString(tok) }
func predictBooleanLiteral(tok token) bool {
	return (tok.Value == "true" || tok.Value == "false") && tok.Kind == scanner.IDENT
} // XXX case insensitive
func predictBlankNode(tok token) bool {
	return tok.Kind == scanner.BLANK_NODE_LABEL || tok.Kind == scanner.ANON
}
func predictVarOrIRIref(tok token) bool              { return predictVar(tok) || predictIRIref(tok) }
func predictCollection(tok token) bool               { return tok.Kind == scanner.LPAREN }
func predictBlankNodePropertyList(tok token) bool    { return tok.Kind == scanner.LBRACK }
func predictGraphNode(tok token) bool                { return predictVarOrTerm(tok) || predictTriplesNode(tok) }
func predictVerb(tok token) bool                     { return predictVarOrIRIref(tok) || tok.Kind == scanner.A }
func predictPropertyListNotEmpty(tok token) bool     { return predictVerb(tok) }
func predictFunctionCall(tok token) bool             { return predictIRIref(tok) }
func predictConstructTriples(tok token) bool         { return predictTriplesSameSubject(tok) }
func predictOptionalGraphPattern(tok token) bool     { return tok.Kind == scanner.OPTIONAL }
func predictGroupOrUnionGraphPattern(tok token) bool { return predictGroupGraphPattern(tok) }
func predictGraphGraphPattern(tok token) bool        { return tok.Kind == scanner.GRAPH }
func predictGroupGraphPattern(tok token) bool        { return tok.Kind == scanner.LBRACE }
func predictFilter(tok token) bool                   { return tok.Kind == scanner.FILTER }
func predictDefaultGraphClause(tok token) bool       { return predictSourceSelector(tok) }
func predictNamedGraphClause(tok token) bool         { return tok.Kind == scanner.NAMED }
func predictSourceSelector(tok token) bool           { return predictIRIref(tok) }

func predictTriplesNode(tok token) bool {
	return predictCollection(tok) || predictBlankNodePropertyList(tok)
}

func predictIRIref(tok token) bool {
	if tok.Kind == scanner.IRI_REF {
		return true
	}
	return predictPrefixedName(tok)
}

func predictGraphTerm(tok token) bool {
	return predictIRIref(tok) || predictRDFLiteral(tok) || predictNumericLiteral(tok) || predictBooleanLiteral(tok) || predictBlankNode(tok) || tok.Kind == scanner.NIL
}

func predictNumericLiteral(tok token) bool {
	return predictNumericLiteralUnsigned(tok) || predictNumericLiteralPositive(tok) || predictNumericLiteralNegative(tok)
}

func predictNumericLiteralUnsigned(tok token) bool {
	switch tok.Kind {
	case scanner.INTEGER, scanner.DECIMAL, scanner.DOUBLE:
		return true
	default:
		return false
	}
}

func predictNumericLiteralPositive(tok token) bool {
	switch tok.Kind {
	case scanner.INTEGER_POSITIVE, scanner.DECIMAL_POSITIVE, scanner.DOUBLE_POSITIVE:
		return true
	default:
		return false
	}
}

func predictNumericLiteralNegative(tok token) bool {
	switch tok.Kind {
	case scanner.INTEGER_NEGATIVE, scanner.DECIMAL_NEGATIVE, scanner.DOUBLE_NEGATIVE:
		return true
	default:
		return false
	}
}

func predictString(tok token) bool {
	switch tok.Kind {
	case scanner.STRING_LITERAL1, scanner.STRING_LITERAL2, scanner.STRING_LITERAL_LONG1, scanner.STRING_LITERAL_LONG2:
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
	if tok.Kind == scanner.IDENT {
		switch tok.Value {
		case "STR", "LANG", "LANGMATCHES", "DATATYPE", "BOUND", "sameTerm", "isIRI", "isURI", "isBLANK", "isLITERAL": // XXX case insensitive
			return true
		}
	}

	return predictRegexExpression(tok)
}

func predictArgList(tok token) bool {
	return tok.Kind == scanner.NIL || tok.Kind == scanner.LPAREN
}

func predictOrderCondition(tok token) bool {
	return tok.Kind == scanner.ASC || tok.Kind == scanner.DESC || predictConstraint(tok) || predictVar(tok)
}

func predictConstraint(tok token) bool {
	return predictBrackettedExpression(tok) || predictBuiltInCall(tok) || predictFunctionCall(tok)
}

func predictWhereClause(tok token) bool {
	return tok.Kind == scanner.WHERE || predictGroupGraphPattern(tok)
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
	if tok.Kind != scanner.VAR1 && tok.Kind != scanner.VAR2 {
		panic(fmt.Sprintf("unexpected token %v", tok))
	}
	return ast.Var(tok.Value[1:])
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
	} else if tok.Kind == scanner.NIL {
		iter.next()
		return ast.IRI("http://www.w3.org/1999/02/22-rdf-syntax-ns#nil")
	} else {
		panic(fmt.Sprintf("unexpected token %v", tok))
	}
}

func BaseDecl(iter *iter) ast.IRI {
	requireToken(iter, scanner.BASE)

	return ast.IRI(requireToken(iter, scanner.IRI_REF).Value)
}

func PrefixDecl(iter *iter) ast.Prefix {
	iter.next()
	return ast.Prefix{
		Name: requireToken(iter, scanner.PNAME_NS).Value,
		IRI:  ast.IRI(requireToken(iter, scanner.IRI_REF).Value),
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
	requireToken(iter, scanner.NAMED)
	return ast.From{
		Named: true,
		IRI:   SourceSelector(iter),
	}
}

func SourceSelector(iter *iter) ast.IRIref {
	return IRIref(iter)
}

func DatasetClause(iter *iter) ast.From {
	requireToken(iter, scanner.FROM)

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
	switch tok.Kind {
	case scanner.PNAME_LN:
		return ast.PrefixedName(iter.next().Value)
	case scanner.PNAME_NS:
		return ast.PrefixedName(iter.next().Value)
	default:
		panic(fmt.Sprintf("unexpected token %v", tok))
	}
}

func IRIref(iter *iter) ast.IRIref {
	tok := iter.peek()
	if tok.Kind == scanner.IRI_REF {
		return ast.IRI(iter.next().Value)
	} else if predictPrefixedName(tok) {
		return PrefixedName(iter)
	} else {
		panic(fmt.Sprintf("unexpected token %v", tok))
	}
}

func String(iter *iter) string {
	tok := iter.peek()
	switch tok.Kind {
	case scanner.STRING_LITERAL1, scanner.STRING_LITERAL2, scanner.STRING_LITERAL_LONG1, scanner.STRING_LITERAL_LONG2:
		tok := iter.next()
		// XXX strip quotes
		return tok.Value
	default:
		panic(fmt.Sprintf("unexpected token %v", tok))
	}
}

func RDFLiteral(iter *iter) ast.RDFLiteral {
	s := String(iter)
	var lang string
	var typ ast.IRIref

	tok := iter.peek()
	if tok.Kind == scanner.LANGTAG {
		lang = iter.next().Value[1:]
	} else if tok.Kind == scanner.TYPE {
		iter.next()
		typ = IRIref(iter)
	}

	// XXX strip surrounding quotes
	return ast.RDFLiteral{Value: s, Lang: lang, Type: typ}
}

func NumericLiteralUnsigned(iter *iter) ast.RDFLiteral {
	tok := iter.peek()
	switch tok.Kind {
	case scanner.INTEGER:
		return ast.RDFLiteral{
			Value: iter.next().Value,
			Type:  ast.IRI("http://www.w3.org/2001/XMLSchema#integer"),
		}
	case scanner.DECIMAL:
		return ast.RDFLiteral{
			Value: iter.next().Value,
			Type:  ast.IRI("http://www.w3.org/2001/XMLSchema#decimal"),
		}
	case scanner.DOUBLE:
		return ast.RDFLiteral{
			Value: iter.next().Value,
			Type:  ast.IRI("http://www.w3.org/2001/XMLSchema#double"),
		}
	default:
		panic(fmt.Sprintf("unexpected token %v", tok))
	}
}

func NumericLiteralPositive(iter *iter) ast.RDFLiteral {
	tok := iter.peek()
	switch tok.Kind {
	case scanner.INTEGER_POSITIVE:
		return ast.RDFLiteral{
			Value: iter.next().Value[1:],
			Type:  ast.IRI("http://www.w3.org/2001/XMLSchema#integer"),
		}
	case scanner.DECIMAL_POSITIVE:
		return ast.RDFLiteral{
			Value: iter.next().Value[1:],
			Type:  ast.IRI("http://www.w3.org/2001/XMLSchema#decimal"),
		}
	case scanner.DOUBLE_POSITIVE:
		return ast.RDFLiteral{
			Value: iter.next().Value[1:],
			Type:  ast.IRI("http://www.w3.org/2001/XMLSchema#double"),
		}
	default:
		panic(fmt.Sprintf("unexpected token %v", tok))
	}
}

func NumericLiteralNegative(iter *iter) ast.RDFLiteral {
	tok := iter.peek()
	switch tok.Kind {
	case scanner.INTEGER_NEGATIVE:
		return ast.RDFLiteral{
			Value: iter.next().Value,
			Type:  ast.IRI("http://www.w3.org/2001/XMLSchema#integer"),
		}
	case scanner.DECIMAL_NEGATIVE:
		return ast.RDFLiteral{
			Value: iter.next().Value,
			Type:  ast.IRI("http://www.w3.org/2001/XMLSchema#decimal"),
		}
	case scanner.DOUBLE_NEGATIVE:
		return ast.RDFLiteral{
			Value: iter.next().Value,
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
	if tok.Kind != scanner.IDENT {
		panic(fmt.Sprintf("unexpected token %v", tok))
	}
	switch tok.Value {
	case "true": // XXX
		iter.next()
		return ast.RDFLiteral{
			Value: "true",
			Type:  ast.IRI("http://www.w3.org/2001/XMLSchema#boolean"),
		}
	case "false": // XXX
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
	switch tok.Kind {
	case scanner.BLANK_NODE_LABEL:
		iter.next()
		return iter.newBlankNode()
	case scanner.ANON:
		tok := iter.next()
		return ast.NamedBlankNode{
			Name: tok.Value,
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
	} else if tok.Kind == scanner.A {
		iter.next()
		return ast.IRI("http://www.w3.org/1999/02/22-rdf-syntax-ns#type")
	} else {
		panic(fmt.Sprintf("unexpected token %v", tok))
	}
}

func Collection(iter *iter) {
	panic("collections are unsupported")

	requireToken(iter, scanner.LPAREN)
	GraphNode(iter)
	for tok := iter.peek(); predictGraphNode(tok); tok = iter.peek() {
		GraphNode(iter)
	}
	requireToken(iter, scanner.RPAREN)
}

func BlankNodePropertyList(iter *iter) []verbObject {
	requireToken(iter, scanner.LBRACK)
	vos := PropertyListNotEmpty(iter)
	requireToken(iter, scanner.RBRACK)
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
	for tok := iter.peek(); tok.Kind == scanner.COMMA; tok = iter.peek() {
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
	for tok := iter.peek(); tok.Kind == scanner.SEMICOLON; tok = iter.peek() {
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
	if iter.peek().Kind == scanner.PERIOD {
		iter.next()
		if predictTriplesBlock(iter.peek()) {
			out = append(out, TriplesBlock(iter)...)
		}
	}
	return out
}

func OptionalGraphPattern(iter *iter) ast.GroupGraphPattern {
	requireToken(iter, scanner.OPTIONAL)
	ggp := GroupGraphPattern(iter)
	ggp.Optional = true
	return ggp
}

func GroupOrUnionGraphPattern(iter *iter) ast.Node {
	var patterns []ast.GroupGraphPattern
	patterns = append(patterns, GroupGraphPattern(iter))
	for tok := iter.peek(); tok.Kind == scanner.UNION; tok = iter.peek() {
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
	requireToken(iter, scanner.GRAPH)
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
	requireToken(iter, scanner.FILTER)
	return Constraint(iter)
}

func GroupGraphPattern(iter *iter) ast.GroupGraphPattern {
	requireToken(iter, scanner.LBRACE)

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
		optionalToken(iter, scanner.PERIOD)
		if predictTriplesBlock(iter.peek()) {
			basic.Patterns = append(basic.Patterns, TriplesBlock(iter)...)
		}
	}

	if len(basic.Patterns) > 0 {
		root.Patterns = append(root.Patterns, basic)
	}

	requireToken(iter, scanner.RBRACE)

	return root
}

func WhereClause(iter *iter) ast.GroupGraphPattern {
	tok := iter.peek()
	if tok.Kind == scanner.WHERE {
		iter.next()
	}

	return GroupGraphPattern(iter)
}

func requireIdent(iter *iter, value string) {
	tok := iter.peek()
	if tok.Value == value && tok.Kind == scanner.IDENT {
		iter.next()
	} else {
		panic(fmt.Sprintf("unexpected token %v", tok))
	}
}

func optionalToken(iter *iter, kind int) bool {
	tok := iter.peek()
	if tok.Kind == kind {
		iter.next()
		return true
	} else {
		return false
	}
}

func requireToken(iter *iter, kind int) token {
	tok := iter.peek()
	if tok.Kind == kind {
		return iter.next()
	} else {
		panic(fmt.Sprintf("unexpected token %v", tok))
	}
}

func RegexExpression(iter *iter) ast.Regex {
	node := ast.Regex{}

	requireIdent(iter, "REGEX")
	requireToken(iter, scanner.LPAREN)
	node.String = Expression(iter)
	requireToken(iter, scanner.COMMA)
	node.Pattern = Expression(iter)
	if iter.peek().Kind == scanner.COMMA {
		iter.next()
		node.Flags = Expression(iter)
	}
	requireToken(iter, scanner.RPAREN)

	return node
}

func BuiltInCall(iter *iter) ast.Expr {
	tok := iter.peek()
	switch tok.Value {
	case "STR", "LANG", "DATATYPE", "isIRI", "isURI", "isBLANK", "isLiteral": // XXX case insensitive
		fn := iter.next().Value
		requireToken(iter, scanner.LPAREN)
		arg1 := Expression(iter)
		requireToken(iter, scanner.RPAREN)
		return ast.Call{
			Func: ast.Builtin{Name: fn},
			Args: []ast.Expr{arg1},
		}
	case "LANGMATCHES", "sameTerm":
		fn := iter.next().Value
		requireToken(iter, scanner.LPAREN)
		arg1 := Expression(iter)
		requireToken(iter, scanner.COMMA)
		arg2 := Expression(iter)
		requireToken(iter, scanner.RPAREN)
		return ast.Call{
			Func: ast.Builtin{Name: fn},
			Args: []ast.Expr{arg1, arg2},
		}
	case "BOUND":
		iter.next()
		requireToken(iter, scanner.LPAREN)
		arg1 := Var(iter)
		requireToken(iter, scanner.RPAREN)
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
	if tok.Kind == scanner.NIL {
		iter.next()
		return []ast.Expr{}
	} else if tok.Kind == scanner.LPAREN {
		iter.next()
		var args []ast.Expr
		args = append(args, Expression(iter))
		for tok := iter.peek(); tok.Kind == scanner.COMMA; tok = iter.peek() {
			iter.next()
			args = append(args, Expression(iter))
		}
		requireToken(iter, scanner.RPAREN)
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
	if tok.Kind == scanner.NOT {
		iter.next()
		return ast.UnOp{Tok: scanner.NOT, X: PrimaryExpression(iter)}
	} else if tok.Kind == scanner.ADD {
		iter.next()
		return ast.UnOp{Tok: scanner.ADD, X: PrimaryExpression(iter)}
	} else if tok.Kind == scanner.SUB {
		iter.next()
		return ast.UnOp{Tok: scanner.SUB, X: PrimaryExpression(iter)}
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
		if tok.Kind == scanner.MUL {
			iter.next()
			expr = ast.BinOp{X: expr, Tok: scanner.MUL, Y: UnaryExpression(iter)}
		} else if tok.Kind == scanner.QUO {
			iter.next()
			expr = ast.BinOp{X: expr, Tok: scanner.QUO, Y: UnaryExpression(iter)}
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
		switch tok.Kind {
		case scanner.ADD:
			iter.next()
			expr = ast.BinOp{X: expr, Tok: scanner.ADD, Y: MultiplicativeExpression(iter)}
		case scanner.SUB:
			iter.next()
			expr = ast.BinOp{X: expr, Tok: scanner.SUB, Y: MultiplicativeExpression(iter)}
		}
		if predictNumericLiteralPositive(tok) {
			expr = ast.BinOp{X: expr, Tok: scanner.ADD, Y: NumericLiteralPositive(iter)}
		} else if predictNumericLiteralNegative(tok) {
			expr = ast.BinOp{X: expr, Tok: scanner.SUB, Y: NumericLiteralNegative(iter)}
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
	switch tok.Kind {
	case scanner.EQL, scanner.NEQ, scanner.LSS, scanner.GTR, scanner.LEQ, scanner.GEQ:
		iter.next()
		return ast.BinOp{X: expr, Tok: tok.Kind, Y: NumericExpression(iter)}
	default:
		return expr
	}
}

func ValueLogical(iter *iter) ast.Expr {
	return RelationalExpression(iter)
}

func ConditionalAndExpression(iter *iter) ast.Expr {
	expr := ValueLogical(iter)
	for tok := iter.peek(); tok.Kind == scanner.LAND; tok = iter.peek() {
		iter.next()
		expr = ast.And{X: expr, Y: ValueLogical(iter)}
	}
	return expr
}

func ConditionalOrExpression(iter *iter) ast.Expr {
	expr := ConditionalAndExpression(iter)
	for tok := iter.peek(); tok.Kind == scanner.LOR; tok = iter.peek() {
		iter.next()
		expr = ast.Or{X: expr, Y: ConditionalAndExpression(iter)}
	}
	return expr
}

func Expression(iter *iter) ast.Node {
	return ConditionalOrExpression(iter)
}

func BrackettedExpression(iter *iter) ast.Expr {
	requireToken(iter, scanner.LPAREN)
	expr := Expression(iter)
	requireToken(iter, scanner.RPAREN)
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
	if tok.Kind == scanner.ASC || tok.Kind == scanner.DESC {
		dir := iter.next().Value
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
	requireToken(iter, scanner.ORDER)
	requireToken(iter, scanner.BY)

	var out []ast.OrderCondition

	out = append(out, OrderCondition(iter))
	for tok := iter.peek(); predictOrderCondition(tok); tok = iter.peek() {
		out = append(out, OrderCondition(iter))
	}
	return out
}

func LimitClause(iter *iter) ast.RDFLiteral {
	requireToken(iter, scanner.LIMIT)
	return ast.RDFLiteral{
		Value: requireToken(iter, scanner.INTEGER).Value,
		Type:  "http://www.w3.org/2001/XMLSchema#integer",
	}
}

func OffsetClause(iter *iter) ast.RDFLiteral {
	requireToken(iter, scanner.OFFSET)
	return ast.RDFLiteral{
		Value: requireToken(iter, scanner.INTEGER).Value,
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
	if tok.Kind == scanner.DISTINCT {
		iter.next()
		query.Distinct = true
	} else if tok.Kind == scanner.REDUCED {
		iter.next()
		query.Reduced = true
	}

	tok = iter.peek()
	if predictVar(tok) {
		for tok := iter.peek(); predictVar(tok); tok = iter.peek() {
			query.Variables = append(query.Variables, Var(iter))
		}
	} else if tok.Kind == scanner.MUL {
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
	if iter.peek().Kind == scanner.PERIOD {
		iter.next()
		if predictConstructTriples(iter.peek()) {
			out = append(out, ConstructTriples(iter)...)
		}
	}
	return out
}

func ConstructTemplate(iter *iter) ast.Node {
	requireToken(iter, scanner.LBRACE)
	var out ast.Node
	if predictConstructTriples(iter.peek()) {
		out = ConstructTriples(iter)
	}
	requireToken(iter, scanner.RBRACE)
	return out
}

func ConstructQuery(iter *iter) ast.ConstructQuery {
	requireToken(iter, scanner.CONSTRUCT)

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

	requireToken(iter, scanner.DESCRIBE)
	tok := iter.peek()
	if predictVarOrIRIref(tok) {
		for tok := iter.peek(); predictVarOrIRIref(tok); tok = iter.peek() {
			query.Variables = append(query.Variables, VarOrIRIref(iter))
		}
	} else if tok.Kind == scanner.MUL {
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

	requireToken(iter, scanner.ASK)
	for tok := iter.peek(); predictDatasetClause(tok); tok = iter.peek() {
		query.Froms = append(query.Froms, DatasetClause(iter))
	}
	query.Where = WhereClause(iter)
	return query
}

func Parse(in string) ast.Query {
	iter := &iter{tokens: scanner.Scan(in)}

	query := ast.Query{}
	query.Base, query.Prefixes = Prologue(iter)
	switch tok := iter.peek(); tok.Kind {
	case scanner.SELECT:
		query.Query = SelectQuery(iter)
	case scanner.CONSTRUCT:
		query.Query = ConstructQuery(iter)
	case scanner.DESCRIBE:
		query.Query = DescribeQuery(iter)
	case scanner.ASK:
		query.Query = AskQuery(iter)
	default:
		panic(fmt.Sprintf("unexpected token %v", tok))
	}

	return query
}

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
