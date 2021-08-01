package scanner

import "log"

const (
	ILLEGAL = iota

	ADD
	SUB
	MUL
	QUO

	EQL    // =
	LSS    // <
	GTR    // >
	ASSIGN // =
	NOT    // !

	NEQ      // !=
	LEQ      // <=
	GEQ      // >=

	LAND  // &&
	LOR   // ||

	LPAREN // (
	LBRACK // [
	LBRACE // {
	COMMA  // ,
	PERIOD // .

	RPAREN    // )
	RBRACK    // ]
	RBRACE    // }
	SEMICOLON // ;

	TYPE     // ^^

	IDENT
	A

	ANON

	// Keywords
	ASC
	ASK
	BASE
	BY
	CONSTRUCT
	DESC
	DESCRIBE
	DISTINCT
	FILTER
	FROM
	GRAPH
	LIMIT
	NAMED
	OFFSET
	OPTIONAL
	ORDER
	PREFIX
	REDUCED
	SELECT
	UNION
	WHERE

	BLANK_NODE_LABEL
	DECIMAL
	DECIMAL_NEGATIVE
	DECIMAL_POSITIVE
	DOUBLE
	DOUBLE_NEGATIVE
	DOUBLE_POSITIVE
	INTEGER
	INTEGER_NEGATIVE
	INTEGER_POSITIVE
	IRI_REF
	LANGTAG
	NIL
	PNAME_LN
	PNAME_NS
	STRING_LITERAL1
	STRING_LITERAL2
	STRING_LITERAL_LONG1
	STRING_LITERAL_LONG2
	VAR1
	VAR2

	EOF
)

%%{
	machine sparql_scanner;
	write data;

	IRI_REF					= '<' [^>]* '>';
	PN_CHARS_BASE			= [A-Z] | [a-z] | 128..255;
	PN_CHARS_U				= PN_CHARS_BASE | '_';
	PN_CHARS				= PN_CHARS_U | '-' | [0-9];
	PN_PREFIX				= PN_CHARS_BASE ((PN_CHARS|'.')* PN_CHARS)?;
	PNAME_NS				= PN_PREFIX? ':';
	PN_LOCAL				= ( PN_CHARS_U | [0-9] ) ((PN_CHARS|'.')* PN_CHARS)?;
	PNAME_LN				= PNAME_NS PN_LOCAL;
	BLANK_NODE_LABEL		= '_:' PN_LOCAL;
	VARNAME					= ( PN_CHARS_U | [0-9] ) ( PN_CHARS_U | [0-9] )*;
	VAR1					= '?' VARNAME;
	VAR2					= '$' VARNAME;
	LANGTAG					= '@' [a-zA-Z]+ ('-' [a-zA-Z0-9]+)*;
	INTEGER					= [0-9]+;
	DECIMAL					= [0-9]+ '.' [0-9]* | '.' [0-9]+;
	EXPONENT				= [eE] [+\-]? [0-9]+;
	DOUBLE					= [0-9]+ '.' [0-9]* EXPONENT | '.' ([0-9])+ EXPONENT | ([0-9])+ EXPONENT;
	INTEGER_POSITIVE		= '+' INTEGER;
	DECIMAL_POSITIVE		= '+' DECIMAL;
	DOUBLE_POSITIVE			= '+' DOUBLE;
	INTEGER_NEGATIVE		= '-' INTEGER;
	DECIMAL_NEGATIVE		= '-' DECIMAL;
	DOUBLE_NEGATIVE			= '-' DOUBLE;
	ECHAR					= '\\' [tbnrf\\"'];
	STRING_LITERAL1			= "'" ( ([^\x27\x5C\xA\xD]) | ECHAR )* "'";
	STRING_LITERAL2			= '"' ( ([^\x22\x5C\xA\xD]) | ECHAR )* '"';
	STRING_LITERAL_LONG1	= "'''" ( ( "'" | "''" )? ( [^'\\] | ECHAR ) )* "'''";
	STRING_LITERAL_LONG2	= '"""' ( ( '"' | '""' )? ( [^"\\] | ECHAR ) )* '"""';
	WS						= 0x20 | 0x09 | 0x0D | 0x0A;
	NIL						= '(' WS* ')';
	ANON					= '[' WS* ']';
	COMMENT = '#' [^\n]*;
		
	main := |*
		IRI_REF { emit(IRI_REF) };
		PNAME_LN { emit(PNAME_LN) };
		PNAME_NS { emit(PNAME_NS) };
		BLANK_NODE_LABEL { emit(BLANK_NODE_LABEL) };
		VAR1 { emit(VAR1) };
		VAR2 { emit(VAR2) };
		LANGTAG { emit(LANGTAG) };
		INTEGER { emit(INTEGER) };
		DECIMAL { emit(DECIMAL) };
		DOUBLE { emit(DOUBLE) };
		INTEGER_POSITIVE { emit(INTEGER_POSITIVE) };
		DECIMAL_POSITIVE { emit(DECIMAL_POSITIVE) };
		DOUBLE_POSITIVE { emit(DOUBLE_POSITIVE) };
		INTEGER_NEGATIVE { emit(INTEGER_NEGATIVE) };
		DECIMAL_NEGATIVE { emit(DECIMAL_NEGATIVE) };
		DOUBLE_NEGATIVE { emit(DOUBLE_NEGATIVE) };
		STRING_LITERAL1 { emit(STRING_LITERAL1) };
		STRING_LITERAL2 { emit(STRING_LITERAL2) };
		STRING_LITERAL_LONG1 { emit(STRING_LITERAL_LONG1) };
		STRING_LITERAL_LONG2 { emit(STRING_LITERAL_LONG2) };
		NIL { emit(NIL) };
		ANON { emit(ANON) };
		WS;

		COMMENT;

		'ASC'i { emit(ASC) };
		'ASK'i { emit(ASK) };
		'BASE'i { emit(BASE) };
		'BOUND'i { emit(IDENT) };
		'BY'i { emit(BY) };
		'CONSTRUCT'i { emit(CONSTRUCT) };
		'DATATYPE'i { emit(IDENT) };
		'DESC'i { emit(DESC) };
		'DESCRIBE'i { emit(DESCRIBE) };
		'DISTINCT'i { emit(DISTINCT) };
		'FILTER'i { emit(FILTER) };
		'FROM'i { emit(FROM) };
		'GRAPH'i { emit(GRAPH) };
		'LANG'i { emit(IDENT) };
		'LANGMATCHES'i { emit(IDENT) };
		'LIMIT'i { emit(LIMIT) };
		'NAMED'i { emit(NAMED) };
		'OFFSET'i { emit(OFFSET) };
		'OPTIONAL'i { emit(OPTIONAL) };
		'PREFIX'i { emit(PREFIX) };
		'REDUCED'i { emit(REDUCED) };
		'REGEX'i { emit(IDENT) };
		'SELECT'i { emit(SELECT) };
		'STR'i { emit(IDENT) };
		'UNION'i { emit(UNION) };
		'WHERE'i { emit(WHERE) };
		'ORDER'i { emit(ORDER) };
		'a' { emit(A) };
		'false'i { emit(IDENT) };
		'isBLANK'i { emit(IDENT) };
		'isIRI'i { emit(IDENT) };
		'isLITERAL'i { emit(IDENT) };
		'isURI'i { emit(IDENT) };
		'sameTerm'i { emit(IDENT) };
		'true'i { emit(IDENT) };

		'!' { emit(NOT) };
		'!=' { emit(NEQ) };
		'&&' { emit(LAND) };
		'(' { emit(LPAREN) };
		')' { emit(RPAREN) };
		'*' { emit(MUL) };
		'+' { emit(ADD) };
		',' { emit(COMMA) };
		'-' { emit(SUB) };
		'.' { emit(PERIOD) };
		'/' { emit(QUO) };
		';' { emit(SEMICOLON) };
		'<' { emit(LSS) };
		'<=' { emit(LEQ) };
		'=' { emit(EQL) };
		'>' { emit(GTR) };
		'>=' { emit(GEQ) };
		'{' { emit(LBRACE) };
		'||' { emit(LOR) };
		'}' { emit(RBRACE) };
		'[' { emit(LBRACK) };
		']' { emit(RBRACK) };
		'^^' { emit(TYPE) };


	*|;
}%%


type Token struct {
	Value string
	Kind  int
}

func Scan(data string) []Token {
	var cs, ts, te, act, p int
	pe := len(data)
	eof := -1

	var tokens []Token
	emit := func (kind int) {
		tokens=append(tokens, Token{data[ts:te], kind})
	}

	%%write init;
	%%write exec;

	log.Printf("%q", tokens)
	return tokens
}
