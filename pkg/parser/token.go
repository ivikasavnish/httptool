package parser

import "fmt"

// TokenType represents the type of a token
type TokenType int

const (
	// Special tokens
	ILLEGAL TokenType = iota
	EOF
	COMMENT

	// Literals
	IDENT     // identifier
	NUMBER    // 123
	STRING    // "abc" or 'abc'
	DURATION  // 5m, 30s, 100ms
	VAR_REF   // ${variable}

	// Keywords
	VAR
	REQUEST
	SCENARIO
	LOAD
	RUN
	IF
	ELSE
	ASSERT
	EXTRACT
	RETRY
	CURL
	VUS
	RPS
	FOR
	ITERATIONS
	WITH
	IN
	STATUS
	LATENCY
	BODY
	MAX_ATTEMPTS
	BACKOFF
	BASE_DELAY
	THINK
	TRUE
	FALSE

	// Operators
	ASSIGN       // =
	EQ           // ==
	NOT_EQ       // !=
	LT           // <
	GT           // >
	LTE          // <=
	GTE          // >=
	ARROW        // ->
	DOLLAR       // $
	DOT          // .
	COMMA        // ,
	COLON        // :
	BACKSLASH    // \
	PIPE         // |

	// Delimiters
	LBRACE    // {
	RBRACE    // }
	LPAREN    // (
	RPAREN    // )
	LBRACKET  // [
	RBRACKET  // ]

	// Special
	NEWLINE
)

var keywords = map[string]TokenType{
	"var":          VAR,
	"request":      REQUEST,
	"scenario":     SCENARIO,
	"load":         LOAD,
	"run":          RUN,
	"if":           IF,
	"else":         ELSE,
	"assert":     ASSERT,
	"extract":    EXTRACT,
	"retry":      RETRY,
	"curl":       CURL,
	"vus":        VUS,
	"rps":        RPS,
	"for":        FOR,
	"iterations": ITERATIONS,
	"with":       WITH,
	"in":         IN,
	"status":     STATUS,
	"latency":    LATENCY,
	"body":       BODY,
	"think":      THINK,
	"true":       TRUE,
	"false":      FALSE,
	// Note: max_attempts, backoff, base_delay are NOT keywords
	// They are field names in retry blocks and should be IDENT tokens
}

// Token represents a lexical token
type Token struct {
	Type    TokenType
	Literal string
	Line    int
	Column  int
}

// Position returns a formatted position string
func (t Token) Position() string {
	return fmt.Sprintf("%d:%d", t.Line, t.Column)
}

// String returns a string representation of the token
func (t Token) String() string {
	switch t.Type {
	case EOF:
		return "EOF"
	case NEWLINE:
		return "NEWLINE"
	case COMMENT:
		return fmt.Sprintf("COMMENT(%s)", t.Literal)
	case IDENT:
		return fmt.Sprintf("IDENT(%s)", t.Literal)
	case NUMBER:
		return fmt.Sprintf("NUMBER(%s)", t.Literal)
	case STRING:
		return fmt.Sprintf("STRING(%s)", t.Literal)
	case DURATION:
		return fmt.Sprintf("DURATION(%s)", t.Literal)
	case VAR_REF:
		return fmt.Sprintf("VAR_REF(%s)", t.Literal)
	default:
		if t.Literal != "" {
			return t.Literal
		}
		return tokenTypeNames[t.Type]
	}
}

var tokenTypeNames = map[TokenType]string{
	ILLEGAL:      "ILLEGAL",
	EOF:          "EOF",
	COMMENT:      "COMMENT",
	IDENT:        "IDENT",
	NUMBER:       "NUMBER",
	STRING:       "STRING",
	DURATION:     "DURATION",
	VAR_REF:      "VAR_REF",
	VAR:          "var",
	REQUEST:      "request",
	SCENARIO:     "scenario",
	LOAD:         "load",
	RUN:          "run",
	IF:           "if",
	ELSE:         "else",
	ASSERT:       "assert",
	EXTRACT:      "extract",
	RETRY:        "retry",
	CURL:         "curl",
	VUS:          "vus",
	RPS:          "rps",
	FOR:          "for",
	ITERATIONS:   "iterations",
	WITH:         "with",
	IN:           "in",
	STATUS:       "status",
	LATENCY:      "latency",
	BODY:         "body",
	MAX_ATTEMPTS: "max_attempts",
	BACKOFF:      "backoff",
	BASE_DELAY:   "base_delay",
	THINK:        "think",
	TRUE:         "true",
	FALSE:        "false",
	ASSIGN:       "=",
	EQ:           "==",
	NOT_EQ:       "!=",
	LT:           "<",
	GT:           ">",
	LTE:          "<=",
	GTE:          ">=",
	ARROW:        "->",
	DOLLAR:       "$",
	DOT:          ".",
	COMMA:        ",",
	COLON:        ":",
	BACKSLASH:    "\\",
	PIPE:         "|",
	LBRACE:       "{",
	RBRACE:       "}",
	LPAREN:       "(",
	RPAREN:       ")",
	LBRACKET:     "[",
	RBRACKET:     "]",
	NEWLINE:      "NEWLINE",
}

// LookupIdent checks if identifier is a keyword
func LookupIdent(ident string) TokenType {
	if tok, ok := keywords[ident]; ok {
		return tok
	}
	return IDENT
}
