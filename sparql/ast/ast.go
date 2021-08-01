package ast

type Node interface{}

type Prefix struct {
	Name string
	IRI  IRI
}

type IRI string
type Var string

type SelectQuery struct {
	Distinct         bool
	Reduced          bool
	Variables        []Var
	VarStar          bool
	Froms            []From
	Where            GroupGraphPattern
	SolutionModifier SolutionModifier
}

type ConstructQuery struct {
	Template         Node
	Froms            []From
	Where            GroupGraphPattern
	SolutionModifier SolutionModifier
}

type DescribeQuery struct {
	Variables        []Node
	VarStar          bool
	Froms            []From
	Where            GroupGraphPattern
	SolutionModifier SolutionModifier
}

type AskQuery struct {
	Froms []From
	Where GroupGraphPattern
}

type DatasetClause struct {
	Clause Node // one of DefaultGraphClause or NamedGraphClause
}

type From struct {
	Named bool
	IRI   IRIref
}

// One of IRI or PrefixedName
type IRIref interface{}

type PrefixedName string

type DefaultGraphClause struct{}
type NamedGraphClause struct{}

type SolutionModifier struct {
	Order  []OrderCondition
	Limit  RDFLiteral
	Offset RDFLiteral
}

type OrderCondition struct {
	Direction  string
	Expression Expr
}

type Query struct {
	Base     IRI
	Prefixes []Prefix

	Query Node // one of SelectQuery, ConstructQuery, DewcribeQuery or AskQuery
}

type Regex struct {
	String  Expr
	Pattern Expr
	Flags   Expr
}

type Expr interface{}

type Or struct {
	X Node
	Y Node
}

type And struct {
	X Node
	Y Node
}

type BinOp struct {
	X   Expr
	Tok int
	Y   Expr
}

type UnOp struct {
	Tok int
	X   Expr
}

type RDFLiteral struct {
	Value string
	Lang  string
	Type  IRIref
}

type Builtin struct {
	Name string
}

type Call struct {
	Func Expr
	Args []Expr
}

type GroupGraphPattern struct {
	Optional bool
	Patterns []Node
	Filters  []Expr
}

type Union struct {
	Patterns []GroupGraphPattern
}

type BasicGraphPattern struct {
	Patterns []Triple
}

type Triple struct {
	Subject   Node
	Predicate Node
	Object    Node
}

type BlankNode struct {
	ID int
}

type NamedBlankNode struct {
	Name string
}

type GraphGraphPattern struct {
	Name    Node
	Pattern GroupGraphPattern
}
