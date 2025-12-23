package parser

import (
	"testing"
)

func TestLexer_BasicTokens(t *testing.T) {
	input := `var base_url = "https://example.com"

request test_request {
	curl https://example.com/api

	assert status == 200
}

scenario load_test {
	load 10 vus for 5m
	run test_request
}`

	tests := []struct {
		expectedType    TokenType
		expectedLiteral string
	}{
		{VAR, "var"},
		{IDENT, "base_url"},
		{ASSIGN, "="},
		{STRING, "https://example.com"},
		{NEWLINE, "\\n"},
		{NEWLINE, "\\n"},
		{REQUEST, "request"},
		{IDENT, "test_request"},
		{LBRACE, "{"},
		{NEWLINE, "\\n"},
		{CURL, "curl"},
		{STRING, "https://example.com/api"},
		{NEWLINE, "\\n"},
		{NEWLINE, "\\n"},
		{ASSERT, "assert"},
		{STATUS, "status"},
		{EQ, "=="},
		{NUMBER, "200"},
		{NEWLINE, "\\n"},
		{RBRACE, "}"},
		{NEWLINE, "\\n"},
		{NEWLINE, "\\n"},
		{SCENARIO, "scenario"},
		{IDENT, "load_test"},
		{LBRACE, "{"},
		{NEWLINE, "\\n"},
		{LOAD, "load"},
		{NUMBER, "10"},
		{VUS, "vus"},
		{FOR, "for"},
		{DURATION, "5m"},
		{NEWLINE, "\\n"},
		{RUN, "run"},
		{IDENT, "test_request"},
		{NEWLINE, "\\n"},
		{RBRACE, "}"},
		{EOF, ""},
	}

	l := NewLexer(input)

	for i, tt := range tests {
		tok := l.NextToken()

		if tok.Type != tt.expectedType {
			t.Fatalf("tests[%d] - tokentype wrong. expected=%q, got=%q (literal=%q)",
				i, tokenTypeNames[tt.expectedType], tokenTypeNames[tok.Type], tok.Literal)
		}

		if tok.Literal != tt.expectedLiteral {
			t.Fatalf("tests[%d] - literal wrong. expected=%q, got=%q",
				i, tt.expectedLiteral, tok.Literal)
		}
	}
}

func TestLexer_CurlCommand(t *testing.T) {
	input := `request login {
	curl 'https://api.example.com/login' \
		-H 'Content-Type: application/json' \
		-d '{"user":"admin"}'

	assert status == 200
}`

	l := NewLexer(input)

	expectedTokens := []TokenType{
		REQUEST, IDENT, LBRACE, NEWLINE,
		CURL, STRING, STRING, STRING, STRING, STRING, NEWLINE, NEWLINE,
		ASSERT, STATUS, EQ, NUMBER, NEWLINE,
		RBRACE, EOF,
	}

	for i, expected := range expectedTokens {
		tok := l.NextToken()
		if tok.Type != expected {
			t.Fatalf("tests[%d] - tokentype wrong. expected=%q, got=%q (literal=%q)",
				i, tokenTypeNames[expected], tokenTypeNames[tok.Type], tok.Literal)
		}
	}
}

func TestLexer_CurlWithVariableRef(t *testing.T) {
	input := `request test {
	curl ${base_url}/api/users

	assert status == 200
}`

	l := NewLexer(input)

	expectedTokens := []struct {
		typ     TokenType
		literal string
	}{
		{REQUEST, "request"},
		{IDENT, "test"},
		{LBRACE, "{"},
		{NEWLINE, "\\n"},
		{CURL, "curl"},
		{VAR_REF, "base_url"},
		{STRING, "/api/users"},
		{NEWLINE, "\\n"},
		{NEWLINE, "\\n"},
		{ASSERT, "assert"},
		{STATUS, "status"},
		{EQ, "=="},
		{NUMBER, "200"},
		{NEWLINE, "\\n"},
		{RBRACE, "}"},
		{EOF, ""},
	}

	for i, expected := range expectedTokens {
		tok := l.NextToken()
		if tok.Type != expected.typ {
			t.Fatalf("tests[%d] - tokentype wrong. expected=%q, got=%q (literal=%q)",
				i, tokenTypeNames[expected.typ], tokenTypeNames[tok.Type], tok.Literal)
		}
		if expected.literal != "" && tok.Literal != expected.literal {
			t.Fatalf("tests[%d] - literal wrong. expected=%q, got=%q",
				i, expected.literal, tok.Literal)
		}
	}
}

func TestLexer_Comments(t *testing.T) {
	input := `# This is a comment
var x = "test" # inline comment`

	l := NewLexer(input)

	tests := []struct {
		expectedType    TokenType
		expectedLiteral string
	}{
		{COMMENT, "This is a comment"},
		{NEWLINE, "\\n"},
		{VAR, "var"},
		{IDENT, "x"},
		{ASSIGN, "="},
		{STRING, "test"},
		{COMMENT, "inline comment"},
		{EOF, ""},
	}

	for i, tt := range tests {
		tok := l.NextToken()

		if tok.Type != tt.expectedType {
			t.Fatalf("tests[%d] - tokentype wrong. expected=%q, got=%q",
				i, tokenTypeNames[tt.expectedType], tokenTypeNames[tok.Type])
		}

		if tok.Literal != tt.expectedLiteral {
			t.Fatalf("tests[%d] - literal wrong. expected=%q, got=%q",
				i, tt.expectedLiteral, tok.Literal)
		}
	}
}

func TestLexer_Operators(t *testing.T) {
	input := `== != < > <= >= = ->`

	l := NewLexer(input)

	tests := []TokenType{
		EQ, NOT_EQ, LT, GT, LTE, GTE, ASSIGN, ARROW, EOF,
	}

	for i, expected := range tests {
		tok := l.NextToken()
		if tok.Type != expected {
			t.Fatalf("tests[%d] - tokentype wrong. expected=%q, got=%q",
				i, tokenTypeNames[expected], tokenTypeNames[tok.Type])
		}
	}
}

func TestLexer_Duration(t *testing.T) {
	input := `5m 30s 100ms 2h`

	l := NewLexer(input)

	tests := []struct {
		expectedType    TokenType
		expectedLiteral string
	}{
		{DURATION, "5m"},
		{DURATION, "30s"},
		{DURATION, "100ms"},
		{DURATION, "2h"},
		{EOF, ""},
	}

	for i, tt := range tests {
		tok := l.NextToken()

		if tok.Type != tt.expectedType {
			t.Fatalf("tests[%d] - tokentype wrong. expected=%q, got=%q",
				i, tokenTypeNames[tt.expectedType], tokenTypeNames[tok.Type])
		}

		if tok.Literal != tt.expectedLiteral {
			t.Fatalf("tests[%d] - literal wrong. expected=%q, got=%q",
				i, tt.expectedLiteral, tok.Literal)
		}
	}
}

func TestLexer_VariableReference(t *testing.T) {
	input := `${base_url} ${user_id}`

	l := NewLexer(input)

	tests := []struct {
		expectedType    TokenType
		expectedLiteral string
	}{
		{VAR_REF, "base_url"},
		{VAR_REF, "user_id"},
		{EOF, ""},
	}

	for i, tt := range tests {
		tok := l.NextToken()

		if tok.Type != tt.expectedType {
			t.Fatalf("tests[%d] - tokentype wrong. expected=%q, got=%q",
				i, tokenTypeNames[tt.expectedType], tokenTypeNames[tok.Type])
		}

		if tok.Literal != tt.expectedLiteral {
			t.Fatalf("tests[%d] - literal wrong. expected=%q, got=%q",
				i, tt.expectedLiteral, tok.Literal)
		}
	}
}

func TestLexer_SequentialFlow(t *testing.T) {
	input := `run login -> get_profile -> update_settings`

	l := NewLexer(input)

	tests := []TokenType{
		RUN, IDENT, ARROW, IDENT, ARROW, IDENT, EOF,
	}

	for i, expected := range tests {
		tok := l.NextToken()
		if tok.Type != expected {
			t.Fatalf("tests[%d] - tokentype wrong. expected=%q, got=%q (literal=%q)",
				i, tokenTypeNames[expected], tokenTypeNames[tok.Type], tok.Literal)
		}
	}
}

func TestLexer_LoadConfig(t *testing.T) {
	input := `load {
	vus = 10
	duration = 5m
	rps = 100
}`

	l := NewLexer(input)

	tests := []TokenType{
		LOAD, LBRACE, NEWLINE,
		VUS, ASSIGN, NUMBER, NEWLINE,
		IDENT, ASSIGN, DURATION, NEWLINE,
		RPS, ASSIGN, NUMBER, NEWLINE,
		RBRACE, EOF,
	}

	for i, expected := range tests {
		tok := l.NextToken()
		if tok.Type != expected {
			t.Fatalf("tests[%d] - tokentype wrong. expected=%q, got=%q (literal=%q)",
				i, tokenTypeNames[expected], tokenTypeNames[tok.Type], tok.Literal)
		}
	}
}
