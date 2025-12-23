package parser

import (
	"strings"
	"unicode"
)

// Lexer performs lexical analysis on input text
type Lexer struct {
	input        string
	position     int  // current position in input (points to current char)
	readPosition int  // current reading position in input (after current char)
	ch           byte // current char under examination
	line         int  // current line number
	column       int  // current column number
	inCurl       bool // true when inside curl command
}

// NewLexer creates a new lexer for the given input
func NewLexer(input string) *Lexer {
	l := &Lexer{
		input:  input,
		line:   1,
		column: 0,
	}
	l.readChar()
	return l
}

// readChar reads the next character and advances position
func (l *Lexer) readChar() {
	if l.readPosition >= len(l.input) {
		l.ch = 0 // ASCII code for NUL, signifies EOF
	} else {
		l.ch = l.input[l.readPosition]
	}
	l.position = l.readPosition
	l.readPosition++
	l.column++
}

// peekChar returns the next character without advancing
func (l *Lexer) peekChar() byte {
	if l.readPosition >= len(l.input) {
		return 0
	}
	return l.input[l.readPosition]
}

// peekAhead looks ahead n characters
func (l *Lexer) peekAhead(n int) byte {
	pos := l.readPosition + n - 1
	if pos >= len(l.input) {
		return 0
	}
	return l.input[pos]
}

// NextToken returns the next token from the input
func (l *Lexer) NextToken() Token {
	var tok Token

	l.skipWhitespace()

	tok.Line = l.line
	tok.Column = l.column

	// Handle curl command special mode
	if l.inCurl {
		return l.readCurlArg()
	}

	switch l.ch {
	case '#':
		tok.Type = COMMENT
		tok.Literal = l.readComment()
		return tok
	case '\n':
		tok.Type = NEWLINE
		tok.Literal = "\\n"
		l.line++
		l.column = 0
		l.readChar()
		return tok
	case '=':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok.Type = EQ
			tok.Literal = string(ch) + string(l.ch)
		} else {
			tok.Type = ASSIGN
			tok.Literal = string(l.ch)
		}
	case '!':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok.Type = NOT_EQ
			tok.Literal = string(ch) + string(l.ch)
		} else {
			tok.Type = ILLEGAL
			tok.Literal = string(l.ch)
		}
	case '<':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok.Type = LTE
			tok.Literal = string(ch) + string(l.ch)
		} else {
			tok.Type = LT
			tok.Literal = string(l.ch)
		}
	case '>':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok.Type = GTE
			tok.Literal = string(ch) + string(l.ch)
		} else {
			tok.Type = GT
			tok.Literal = string(l.ch)
		}
	case '-':
		if l.peekChar() == '>' {
			ch := l.ch
			l.readChar()
			tok.Type = ARROW
			tok.Literal = string(ch) + string(l.ch)
		} else {
			// In curl mode, this would be a flag
			tok.Type = ILLEGAL
			tok.Literal = string(l.ch)
		}
	case '$':
		if l.peekChar() == '{' {
			return l.readVariableRef()
		}
		tok.Type = DOLLAR
		tok.Literal = string(l.ch)
	case '{':
		tok.Type = LBRACE
		tok.Literal = string(l.ch)
	case '}':
		tok.Type = RBRACE
		tok.Literal = string(l.ch)
	case '(':
		tok.Type = LPAREN
		tok.Literal = string(l.ch)
	case ')':
		tok.Type = RPAREN
		tok.Literal = string(l.ch)
	case '[':
		tok.Type = LBRACKET
		tok.Literal = string(l.ch)
	case ']':
		tok.Type = RBRACKET
		tok.Literal = string(l.ch)
	case '.':
		tok.Type = DOT
		tok.Literal = string(l.ch)
	case ',':
		tok.Type = COMMA
		tok.Literal = string(l.ch)
	case ':':
		tok.Type = COLON
		tok.Literal = string(l.ch)
	case '\\':
		tok.Type = BACKSLASH
		tok.Literal = string(l.ch)
	case '|':
		tok.Type = PIPE
		tok.Literal = string(l.ch)
	case '"', '\'':
		tok.Type = STRING
		tok.Literal = l.readString(l.ch)
		return tok
	case 0:
		tok.Type = EOF
		tok.Literal = ""
		return tok
	default:
		if isLetter(l.ch) {
			tok.Literal = l.readIdentifier()
			tok.Type = LookupIdent(tok.Literal)

			// Special handling for 'curl' keyword - enter curl mode
			if tok.Type == CURL {
				l.inCurl = true
			}

			return tok
		} else if isDigit(l.ch) {
			literal := l.readNumber()

			// Check if this is a duration (e.g., 5m, 30s, 100ms)
			if l.ch != 0 && isLetter(l.ch) {
				unit := l.readDurationUnit()
				tok.Type = DURATION
				tok.Literal = literal + unit
				return tok
			}

			tok.Type = NUMBER
			tok.Literal = literal
			return tok
		} else {
			tok.Type = ILLEGAL
			tok.Literal = string(l.ch)
		}
	}

	l.readChar()
	return tok
}

// readCurlArg reads a curl argument/flag
func (l *Lexer) readCurlArg() Token {
	var tok Token
	tok.Line = l.line
	tok.Column = l.column

	// Skip whitespace
	l.skipWhitespace()

	// Check for newline without backslash - end of curl command
	if l.ch == '\n' {
		l.inCurl = false
		tok.Type = NEWLINE
		tok.Literal = "\\n"
		l.line++
		l.column = 0
		l.readChar()
		return tok
	}

	// Check for backslash-newline continuation
	if l.ch == '\\' && l.peekChar() == '\n' {
		l.readChar() // skip backslash
		l.readChar() // skip newline
		l.line++
		l.column = 0
		// Continue reading curl args
		return l.readCurlArg()
	}

	// Check for 'assert', 'extract', 'retry' - these end curl mode
	if isLetter(l.ch) {
		pos := l.position
		ident := l.readIdentifier()
		tokType := LookupIdent(ident)

		if tokType == ASSERT || tokType == EXTRACT || tokType == RETRY {
			l.inCurl = false
			return Token{
				Type:    tokType,
				Literal: ident,
				Line:    tok.Line,
				Column:  tok.Column,
			}
		}

		// Reset and read as curl arg
		l.position = pos
		l.readPosition = pos + 1
		l.ch = l.input[l.position]
	}

	// Handle variable references
	if l.ch == '$' && l.peekChar() == '{' {
		return l.readVariableRef()
	}

	// Read as curl arg (everything until whitespace or newline)
	if l.ch == '"' || l.ch == '\'' {
		tok.Type = STRING
		tok.Literal = l.readString(l.ch)
		return tok
	}

	// Read raw argument (may contain variable refs)
	tok.Type = STRING
	tok.Literal = l.readCurlComplexArg()
	return tok
}

// readCurlRawArg reads a raw curl argument
func (l *Lexer) readCurlRawArg() string {
	position := l.position

	for l.ch != 0 && l.ch != '\n' && l.ch != '\\' && !unicode.IsSpace(rune(l.ch)) {
		l.readChar()
	}

	return l.input[position:l.position]
}

// readCurlComplexArg reads a curl arg that may contain text and variable refs
// Returns just the raw text portion
func (l *Lexer) readCurlComplexArg() string {
	position := l.position

	for l.ch != 0 && l.ch != '\n' && l.ch != '\\' && !unicode.IsSpace(rune(l.ch)) {
		// Stop at variable reference - let it be read separately
		if l.ch == '$' && l.peekChar() == '{' {
			break
		}
		l.readChar()
	}

	return l.input[position:l.position]
}

// readComment reads a comment until end of line
func (l *Lexer) readComment() string {
	position := l.position + 1 // skip '#'

	for l.ch != '\n' && l.ch != 0 {
		l.readChar()
	}

	return strings.TrimSpace(l.input[position:l.position])
}

// readIdentifier reads an identifier
func (l *Lexer) readIdentifier() string {
	position := l.position

	for isLetter(l.ch) || isDigit(l.ch) || l.ch == '_' {
		l.readChar()
	}

	return l.input[position:l.position]
}

// readNumber reads a number
func (l *Lexer) readNumber() string {
	position := l.position

	for isDigit(l.ch) {
		l.readChar()
	}

	return l.input[position:l.position]
}

// readDurationUnit reads duration unit (ms, s, m, h)
func (l *Lexer) readDurationUnit() string {
	position := l.position

	for isLetter(l.ch) {
		l.readChar()
	}

	return l.input[position:l.position]
}

// readString reads a string literal
func (l *Lexer) readString(quote byte) string {
	var result strings.Builder

	l.readChar() // skip opening quote

	for l.ch != quote && l.ch != 0 {
		if l.ch == '\\' {
			l.readChar()
			switch l.ch {
			case 'n':
				result.WriteByte('\n')
			case 't':
				result.WriteByte('\t')
			case 'r':
				result.WriteByte('\r')
			case '\\':
				result.WriteByte('\\')
			case quote:
				result.WriteByte(quote)
			default:
				result.WriteByte(l.ch)
			}
		} else {
			if l.ch == '\n' {
				l.line++
				l.column = 0
			}
			result.WriteByte(l.ch)
		}
		l.readChar()
	}

	l.readChar() // skip closing quote

	return result.String()
}

// readVariableRef reads a variable reference ${var_name}
func (l *Lexer) readVariableRef() Token {
	tok := Token{
		Type:   VAR_REF,
		Line:   l.line,
		Column: l.column,
	}

	l.readChar() // skip '$'
	l.readChar() // skip '{'

	position := l.position

	for l.ch != '}' && l.ch != 0 {
		l.readChar()
	}

	tok.Literal = l.input[position:l.position]

	l.readChar() // skip '}'

	return tok
}

// skipWhitespace skips whitespace characters except newlines
func (l *Lexer) skipWhitespace() {
	for l.ch == ' ' || l.ch == '\t' || l.ch == '\r' {
		l.readChar()
	}
}

// isLetter checks if a character is a letter
func isLetter(ch byte) bool {
	return 'a' <= ch && ch <= 'z' || 'A' <= ch && ch <= 'Z' || ch == '_'
}

// isDigit checks if a character is a digit
func isDigit(ch byte) bool {
	return '0' <= ch && ch <= '9'
}
