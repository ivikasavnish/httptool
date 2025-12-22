package parser

import (
	"fmt"
	"strings"
)

// tokenize splits a curl command into tokens, respecting quotes and escapes
func tokenize(input string) ([]string, error) {
	var tokens []string
	var current strings.Builder
	var inQuote rune // 0, '"', or '\''
	escaped := false

	input = strings.TrimSpace(input)

	for i, ch := range input {
		if escaped {
			current.WriteRune(ch)
			escaped = false
			continue
		}

		if ch == '\\' {
			// Check if this is an escape sequence
			if i+1 < len(input) {
				next := rune(input[i+1])
				if next == '"' || next == '\'' || next == '\\' || next == ' ' {
					escaped = true
					continue
				}
			}
			// Not an escape, write it
			current.WriteRune(ch)
			continue
		}

		if inQuote != 0 {
			if ch == inQuote {
				inQuote = 0
			} else {
				current.WriteRune(ch)
			}
			continue
		}

		if ch == '"' || ch == '\'' {
			inQuote = ch
			continue
		}

		if ch == ' ' || ch == '\t' || ch == '\n' {
			if current.Len() > 0 {
				tokens = append(tokens, current.String())
				current.Reset()
			}
			continue
		}

		current.WriteRune(ch)
	}

	if inQuote != 0 {
		return nil, fmt.Errorf("unclosed quote: %c", inQuote)
	}

	if current.Len() > 0 {
		tokens = append(tokens, current.String())
	}

	return tokens, nil
}
