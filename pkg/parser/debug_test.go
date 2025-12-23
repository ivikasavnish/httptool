package parser

import (
	"fmt"
	"testing"
)

func TestDebug_DottedPath(t *testing.T) {
	input := `$.data.user.id`
	l := NewLexer(input)

	fmt.Println("Tokens for:", input)
	for {
		tok := l.NextToken()
		fmt.Printf("  %s (literal=%q)\n", tok, tok.Literal)
		if tok.Type == EOF {
			break
		}
	}
}

func TestDebug_ExtractLine(t *testing.T) {
	input := `user_id = $.data.user.id`
	l := NewLexer(input)

	fmt.Println("\nTokens for:", input)
	for {
		tok := l.NextToken()
		fmt.Printf("  %s (literal=%q)\n", tok, tok.Literal)
		if tok.Type == EOF {
			break
		}
	}
}

func TestDebug_CookieExtract(t *testing.T) {
	input := `session = cookie:session_token`
	l := NewLexer(input)

	fmt.Println("\nTokens for:", input)
	for {
		tok := l.NextToken()
		fmt.Printf("  %s (literal=%q)\n", tok, tok.Literal)
		if tok.Type == EOF {
			break
		}
	}
}

func TestDebug_AssertAfterCurl(t *testing.T) {
	input := `request get_user {
	curl https://api.example.com/users/123

	assert status == 200
	assert body.name != null
}`

	l := NewLexer(input)
	fmt.Println("\nTokens for request with assertions:")
	for {
		tok := l.NextToken()
		fmt.Printf("  %s (literal=%q)\n", tok, tok.Literal)
		if tok.Type == EOF {
			break
		}
	}
}

func TestDebug_ConditionalFlow(t *testing.T) {
	input := `scenario conditional_test {
	load 10 vus for 1m
	if ${feature_enabled} == "true" {
		run new_api
	} else {
		run old_api
	}
}`

	l := NewLexer(input)
	fmt.Println("\nTokens for conditional flow in scenario:")
	for {
		tok := l.NextToken()
		fmt.Printf("  %s (literal=%q)\n", tok, tok.Literal)
		if tok.Type == EOF {
			break
		}
	}
}

func TestDebug_RetryConfig(t *testing.T) {
	input := `retry {
	max_attempts = 3
	backoff = exponential
	base_delay = 100ms
}`

	l := NewLexer(input)
	fmt.Println("\nTokens for retry config:")
	for {
		tok := l.NextToken()
		fmt.Printf("  %s (literal=%q)\n", tok, tok.Literal)
		if tok.Type == EOF {
			break
		}
	}
}

