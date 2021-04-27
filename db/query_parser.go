package main

import (
	"github.com/alecthomas/participle"
	"github.com/alecthomas/participle/lexer"
	"github.com/alecthomas/participle/lexer/stateful"
)

type Boolean bool

func (b *Boolean) Capture(values []string) error {
	*b = values[0] == "true"
	return nil
}

type Expression struct {
	Or []*OrCondition `@@ ( "OR" @@ )*`
}

type OrCondition struct {
	And []*Condition `@@ ( "AND" @@ )*`
}

type Condition struct {
	Operand *ConditionOperand `  @@`
	Not     *Condition        `| "NOT" @@`
}

type ConditionOperand struct {
	Operand      *Operand      `@@`
	ConditionRHS *ConditionRHS `@@?`
}

type ConditionRHS struct {
	Compare *Compare `  @@`
}

type Compare struct {
	Operator string   `@( "<>" | "<=" | ">=" | "=" | "<" | ">" | "!=" )`
	Operand  *Operand `(  @@ )`
}

type Like struct {
	Not     bool     `[ @"NOT" ]`
	Operand *Operand `@@`
}

type Between struct {
	Start *Operand `@@`
	End   *Operand `"AND" @@`
}

type Operand struct {
	Summand []*Summand `@@ ( "|" "|" @@ )*`
}

type Summand struct {
	LHS *Factor `@@`
	Op  string  `[ @("+" | "-")`
	RHS *Factor `  @@ ]`
}

type Factor struct {
	LHS *Term  `@@`
	Op  string `( @("*" | "/" | "%")`
	RHS *Term  `  @@ )?`
}

type Term struct {
	Value         *Value      `  @@`
	SymbolRef     *SymbolRef  `| @@`
	SubExpression *Expression `| "(" @@ ")"`
}

type SymbolRef struct {
	Symbol     string        `@Ident @( "." Ident )*`
	Parameters []*Expression `( "(" @@ ( "," @@ )* ")" )?`
}

type Value struct {
	Wildcard bool     `(  @"*"`
	Number   *int64   ` | @Number`
	String   *string  ` | @String`
	Boolean  *Boolean ` | @("TRUE" | "FALSE")`
	Null     bool     ` | @"NULL")`
}

func Sum(a int, b int) int {
	return 1
}

var (
	sqlLexer = lexer.Must(stateful.NewSimple([]stateful.Rule{
		{`Keyword`, `(?i)AND|OR`, nil},
		{`Ident`, `[a-zA-Z_][a-zA-Z0-9_]*`, nil},
		{`Number`, `[-+]?\d*\.?\d+([eE][-+]?\d+)?`, nil},
		{`String`, `'[^']*'|"[^"]*"`, nil},
		{`Operators`, `<>|!=|<=|>=|[-+*/%,.()=<>]`, nil},
		{"whitespace", `\s+`, nil},
	}))
	parser = participle.MustBuild(
		&Expression{},
		participle.Lexer(sqlLexer),
		participle.Unquote("String"),
		participle.CaseInsensitive("Keyword"),
		// participle.Elide("Comment"),
		// Need to solve left recursion detection first, if possible.
		// participle.UseLookahead(),
	)
)

func parse(query string) Expression {
	ast := &Expression{}

	parser.ParseString(query, ast)

	return *ast
}
