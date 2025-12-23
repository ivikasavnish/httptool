package parser

import (
	"fmt"
	"strconv"
	"strings"
)

// Parser parses tokens into an AST
type Parser struct {
	lexer        *Lexer
	currentToken Token
	peekToken    Token
	errors       []string
}

// NewParser creates a new parser
func NewParser(lexer *Lexer) *Parser {
	p := &Parser{
		lexer:  lexer,
		errors: []string{},
	}

	// Read two tokens to initialize current and peek
	p.nextToken()
	p.nextToken()

	return p
}

// Errors returns parsing errors
func (p *Parser) Errors() []string {
	return p.errors
}

// nextToken advances to the next token
func (p *Parser) nextToken() {
	p.currentToken = p.peekToken
	p.peekToken = p.lexer.NextToken()
}

// currentTokenIs checks if current token is of given type
func (p *Parser) currentTokenIs(t TokenType) bool {
	return p.currentToken.Type == t
}

// peekTokenIs checks if peek token is of given type
func (p *Parser) peekTokenIs(t TokenType) bool {
	return p.peekToken.Type == t
}

// expectPeek checks peek token and advances if it matches
func (p *Parser) expectPeek(t TokenType) bool {
	if p.peekTokenIs(t) {
		p.nextToken()
		return true
	}
	p.peekError(t)
	return false
}

// peekError adds an error for unexpected peek token
func (p *Parser) peekError(t TokenType) {
	msg := fmt.Sprintf("expected next token to be %s, got %s instead at %s",
		tokenTypeNames[t], tokenTypeNames[p.peekToken.Type], p.peekToken.Position())
	p.errors = append(p.errors, msg)
}

// error adds a parsing error
func (p *Parser) error(msg string) {
	fullMsg := fmt.Sprintf("%s at %s", msg, p.currentToken.Position())
	p.errors = append(p.errors, fullMsg)
}

// skipNewlines skips all newline tokens
func (p *Parser) skipNewlines() {
	for p.currentTokenIs(NEWLINE) {
		p.nextToken()
	}
}

// skipCommentsAndNewlines skips comments and newlines
func (p *Parser) skipCommentsAndNewlines() {
	for p.currentTokenIs(NEWLINE) || p.currentTokenIs(COMMENT) {
		p.nextToken()
	}
}

// Parse parses the input and returns an AST
func (p *Parser) Parse() *Program {
	program := &Program{
		Statements: []Statement{},
		Pos:        Position{Line: 1, Column: 1},
	}

	for !p.currentTokenIs(EOF) {
		p.skipCommentsAndNewlines()

		if p.currentTokenIs(EOF) {
			break
		}

		stmt := p.parseStatement()
		if stmt != nil {
			program.Statements = append(program.Statements, stmt)
		}

		p.nextToken()
	}

	return program
}

// parseStatement parses a statement
func (p *Parser) parseStatement() Statement {
	switch p.currentToken.Type {
	case VAR:
		return p.parseVariableDeclaration()
	case REQUEST:
		return p.parseRequestDeclaration()
	case SCENARIO:
		return p.parseScenarioDeclaration()
	case COMMENT:
		return p.parseComment()
	default:
		p.error(fmt.Sprintf("unexpected token %s", tokenTypeNames[p.currentToken.Type]))
		return nil
	}
}

// parseComment parses a comment
func (p *Parser) parseComment() *Comment {
	return &Comment{
		Text: p.currentToken.Literal,
		Pos:  Position{Line: p.currentToken.Line, Column: p.currentToken.Column},
	}
}

// parseVariableDeclaration parses: var name = value
func (p *Parser) parseVariableDeclaration() *VariableDeclaration {
	stmt := &VariableDeclaration{
		Pos: Position{Line: p.currentToken.Line, Column: p.currentToken.Column},
	}

	if !p.expectPeek(IDENT) {
		return nil
	}

	stmt.Name = p.currentToken.Literal

	if !p.expectPeek(ASSIGN) {
		return nil
	}

	p.nextToken()
	stmt.Value = p.parseExpression()

	return stmt
}

// parseRequestDeclaration parses a request block
func (p *Parser) parseRequestDeclaration() *RequestDeclaration {
	stmt := &RequestDeclaration{
		Pos:         Position{Line: p.currentToken.Line, Column: p.currentToken.Column},
		Assertions:  []*Assertion{},
		Extractions: []*Extraction{},
	}

	if !p.expectPeek(IDENT) {
		return nil
	}

	stmt.Name = p.currentToken.Literal

	if !p.expectPeek(LBRACE) {
		return nil
	}

	p.nextToken()
	p.skipCommentsAndNewlines()

	// Parse request body (curl, assert, extract, retry)
	for !p.currentTokenIs(RBRACE) && !p.currentTokenIs(EOF) {
		switch p.currentToken.Type {
		case CURL:
			stmt.CurlCommand = p.parseCurlCommand()
		case ASSERT:
			assertions := p.parseAssertBlock()
			stmt.Assertions = append(stmt.Assertions, assertions...)
		case EXTRACT:
			extractions := p.parseExtractBlock()
			stmt.Extractions = append(stmt.Extractions, extractions...)
		case RETRY:
			stmt.RetryConfig = p.parseRetryBlock()
		case COMMENT:
			p.nextToken()
		case NEWLINE:
			p.nextToken()
		default:
			p.error(fmt.Sprintf("unexpected token in request block: %s", tokenTypeNames[p.currentToken.Type]))
			p.nextToken()
		}

		p.skipCommentsAndNewlines()
	}

	return stmt
}

// parseCurlCommand parses a curl command
func (p *Parser) parseCurlCommand() *CurlCommand {
	cmd := &CurlCommand{
		Pos:      Position{Line: p.currentToken.Line, Column: p.currentToken.Column},
		Headers:  make(map[string]string),
		Cookies:  make(map[string]string),
		RawArgs:  []string{},
		URLParts: []Expression{},
	}

	p.nextToken() // consume 'curl'

	var urlBuilder strings.Builder

	// Read curl arguments until newline (that's not escaped)
	for !p.currentTokenIs(NEWLINE) && !p.currentTokenIs(EOF) &&
	    !p.currentTokenIs(ASSERT) && !p.currentTokenIs(EXTRACT) && !p.currentTokenIs(RETRY) {

		switch p.currentToken.Type {
		case STRING:
			arg := p.currentToken.Literal
			cmd.RawArgs = append(cmd.RawArgs, arg)

			// Parse curl flags
			if strings.HasPrefix(arg, "-H") || arg == "-H" {
				// Header flag
				if arg == "-H" {
					p.nextToken()
					if p.currentTokenIs(STRING) {
						cmd.RawArgs = append(cmd.RawArgs, p.currentToken.Literal)
						p.parseHeader(cmd, p.currentToken.Literal)
					}
				} else {
					// -H'header: value'
					headerVal := strings.TrimPrefix(arg, "-H")
					p.parseHeader(cmd, headerVal)
				}
			} else if strings.HasPrefix(arg, "-d") || arg == "-d" {
				// Data/body flag
				if arg == "-d" {
					p.nextToken()
					if p.currentTokenIs(STRING) {
						cmd.Body = p.currentToken.Literal
						cmd.RawArgs = append(cmd.RawArgs, p.currentToken.Literal)
					}
				} else {
					cmd.Body = strings.TrimPrefix(arg, "-d")
				}
			} else if strings.HasPrefix(arg, "-X") || arg == "-X" {
				// Method flag
				if arg == "-X" {
					p.nextToken()
					if p.currentTokenIs(STRING) {
						cmd.Method = p.currentToken.Literal
						cmd.RawArgs = append(cmd.RawArgs, p.currentToken.Literal)
					}
				} else {
					cmd.Method = strings.TrimPrefix(arg, "-X")
				}
			} else if strings.HasPrefix(arg, "-b") || arg == "-b" {
				// Cookie flag
				if arg == "-b" {
					p.nextToken()
					if p.currentTokenIs(STRING) {
						cmd.RawArgs = append(cmd.RawArgs, p.currentToken.Literal)
						p.parseCookies(cmd, p.currentToken.Literal)
					}
				} else {
					cookieVal := strings.TrimPrefix(arg, "-b")
					p.parseCookies(cmd, cookieVal)
				}
			} else if !strings.HasPrefix(arg, "-") {
				// URL or URL part
				urlBuilder.WriteString(arg)
				cmd.URLParts = append(cmd.URLParts, &StringLiteral{
					Value: arg,
					Pos:   Position{Line: p.currentToken.Line, Column: p.currentToken.Column},
				})
			}

		case VAR_REF:
			// Variable reference in URL
			varName := p.currentToken.Literal
			urlBuilder.WriteString("${" + varName + "}")
			cmd.URLParts = append(cmd.URLParts, &VariableReference{
				Name: varName,
				Pos:  Position{Line: p.currentToken.Line, Column: p.currentToken.Column},
			})
		}

		p.nextToken()
	}

	cmd.URL = urlBuilder.String()

	// Default method
	if cmd.Method == "" {
		if cmd.Body != "" {
			cmd.Method = "POST"
		} else {
			cmd.Method = "GET"
		}
	}

	return cmd
}

// parseHeader parses a header string
func (p *Parser) parseHeader(cmd *CurlCommand, header string) {
	parts := strings.SplitN(header, ":", 2)
	if len(parts) == 2 {
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		cmd.Headers[key] = value
	}
}

// parseCookies parses cookie string
func (p *Parser) parseCookies(cmd *CurlCommand, cookies string) {
	pairs := strings.Split(cookies, ";")
	for _, pair := range pairs {
		kv := strings.SplitN(strings.TrimSpace(pair), "=", 2)
		if len(kv) == 2 {
			cmd.Cookies[kv[0]] = kv[1]
		}
	}
}

// parseAssertBlock parses assert block or single assertion
func (p *Parser) parseAssertBlock() []*Assertion {
	assertions := []*Assertion{}
	pos := Position{Line: p.currentToken.Line, Column: p.currentToken.Column}

	p.nextToken() // consume 'assert'

	// Single line assertion: assert status == 200
	if !p.currentTokenIs(LBRACE) {
		assertion := p.parseAssertion(pos)
		if assertion != nil {
			assertions = append(assertions, assertion)
		}
		// parseAssertion leaves us on the value token
		// Advance to move past it
		p.nextToken()
		return assertions
	}

	// Block assertion: assert { ... }
	p.nextToken() // consume '{'
	p.skipNewlines()

	for !p.currentTokenIs(RBRACE) && !p.currentTokenIs(EOF) {
		assertion := p.parseAssertion(pos)
		if assertion != nil {
			assertions = append(assertions, assertion)
		}

		p.nextToken() // advance past assertion value
		p.skipNewlines()
	}

	// Consume the closing brace
	if p.currentTokenIs(RBRACE) {
		p.nextToken()
	}

	return assertions
}

// parseAssertion parses a single assertion
func (p *Parser) parseAssertion(pos Position) *Assertion {
	assertion := &Assertion{
		Pos: pos,
	}

	// Field (status, latency, body.field, etc.)
	field := p.currentToken.Literal

	// Handle dotted paths like body.name
	for p.peekTokenIs(DOT) {
		p.nextToken() // consume field
		p.nextToken() // consume dot
		field += "." + p.currentToken.Literal
	}

	assertion.Field = field

	p.nextToken()

	// Operator
	if p.currentTokenIs(IN) {
		assertion.Operator = "in"
		p.nextToken()

		// Parse array: [200, 201, 204]
		if !p.currentTokenIs(LBRACKET) {
			p.error("expected '[' after 'in'")
			return nil
		}

		p.nextToken()
		assertion.Values = []Expression{}

		for !p.currentTokenIs(RBRACKET) && !p.currentTokenIs(EOF) {
			expr := p.parseExpression()
			assertion.Values = append(assertion.Values, expr)

			p.nextToken()
			if p.currentTokenIs(COMMA) {
				p.nextToken()
			}
		}
		// Now on ']', don't consume it - let caller handle
	} else {
		// Binary operator
		assertion.Operator = p.currentToken.Literal
		p.nextToken()
		assertion.Value = p.parseExpression()
		// Now on the value token, don't consume - let caller handle
	}

	return assertion
}

// parseExtractBlock parses extract block
func (p *Parser) parseExtractBlock() []*Extraction {
	extractions := []*Extraction{}

	p.nextToken()

	if !p.currentTokenIs(LBRACE) {
		p.error("expected '{' after 'extract'")
		return nil
	}

	p.nextToken()
	p.skipNewlines()

	for !p.currentTokenIs(RBRACE) && !p.currentTokenIs(EOF) {
		if p.currentTokenIs(IDENT) {
			extraction := &Extraction{
				Variable: p.currentToken.Literal,
				Pos:      Position{Line: p.currentToken.Line, Column: p.currentToken.Column},
			}

			if !p.expectPeek(ASSIGN) {
				return extractions
			}

			p.nextToken()

			// Parse extraction path - could be dotted path like $.data.user.id
			var pathBuilder strings.Builder

			// Check for type prefix
			if p.currentTokenIs(IDENT) {
				literal := p.currentToken.Literal
				pathBuilder.WriteString(literal)

				// Check for type:value pattern (cookie:name, header:Authorization)
				if p.peekTokenIs(COLON) {
					p.nextToken() // move to COLON
					pathBuilder.WriteString(":")
					p.nextToken() // move to value after COLON
					pathBuilder.WriteString(p.currentToken.Literal)
				}
			} else if p.currentTokenIs(DOLLAR) {
				// JSONPath like $.data.user.id
				pathBuilder.WriteString("$")

				// Handle dotted path
				for p.peekTokenIs(DOT) {
					p.nextToken() // move to DOT
					pathBuilder.WriteString(".")
					p.nextToken() // move to IDENT after DOT
					pathBuilder.WriteString(p.currentToken.Literal)
				}
			}

			path := pathBuilder.String()

			// Determine extraction type
			if strings.HasPrefix(path, "$.") {
				extraction.Type = ExtractJSONPath
			} else if strings.HasPrefix(path, "regex:") {
				extraction.Type = ExtractRegex
				path = strings.TrimPrefix(path, "regex:")
			} else if strings.HasPrefix(path, "header:") {
				extraction.Type = ExtractHeader
				path = strings.TrimPrefix(path, "header:")
			} else if strings.HasPrefix(path, "cookie:") {
				extraction.Type = ExtractCookie
				path = strings.TrimPrefix(path, "cookie:")
			} else {
				extraction.Type = ExtractJSONPath
			}

			extraction.Path = path
			extractions = append(extractions, extraction)
		}

		p.nextToken()
		p.skipNewlines()
	}

	// Consume the closing brace
	if p.currentTokenIs(RBRACE) {
		p.nextToken()
	}

	return extractions
}

// parseRetryBlock parses retry configuration
func (p *Parser) parseRetryBlock() *RetryConfig {
	config := &RetryConfig{
		Pos: Position{Line: p.currentToken.Line, Column: p.currentToken.Column},
	}

	p.nextToken()

	if !p.currentTokenIs(LBRACE) {
		p.error("expected '{' after 'retry'")
		return nil
	}

	p.nextToken()
	p.skipNewlines()

	for !p.currentTokenIs(RBRACE) && !p.currentTokenIs(EOF) {
		if p.currentTokenIs(NEWLINE) {
			p.nextToken()
			continue
		}

		if p.currentTokenIs(IDENT) {
			key := p.currentToken.Literal

			if !p.expectPeek(ASSIGN) {
				return config
			}

			p.nextToken()

			switch key {
			case "max_attempts":
				if p.currentTokenIs(NUMBER) {
					config.MaxAttempts, _ = strconv.Atoi(p.currentToken.Literal)
				}
			case "backoff":
				if p.currentTokenIs(IDENT) {
					config.Backoff = p.currentToken.Literal
				}
			case "base_delay":
				if p.currentTokenIs(DURATION) {
					config.BaseDelay = p.currentToken.Literal
				}
			}

			p.nextToken()
			p.skipNewlines()
		} else {
			// Unknown token, skip it
			p.nextToken()
		}
	}

	// Consume the closing brace
	if p.currentTokenIs(RBRACE) {
		p.nextToken()
	}

	return config
}

// parseScenarioDeclaration parses a scenario block
func (p *Parser) parseScenarioDeclaration() *ScenarioDeclaration {
	stmt := &ScenarioDeclaration{
		Pos:  Position{Line: p.currentToken.Line, Column: p.currentToken.Column},
		Flow: []FlowStatement{},
	}

	if !p.expectPeek(IDENT) {
		return nil
	}

	stmt.Name = p.currentToken.Literal

	if !p.expectPeek(LBRACE) {
		return nil
	}

	p.nextToken()
	p.skipCommentsAndNewlines()

	// Parse scenario body
	for !p.currentTokenIs(RBRACE) && !p.currentTokenIs(EOF) {
		switch p.currentToken.Type {
		case LOAD:
			stmt.LoadConfig = p.parseLoadConfig()
		case RUN:
			flow := p.parseFlowStatement()
			if flow != nil {
				stmt.Flow = append(stmt.Flow, flow)
			}
		case IF:
			flow := p.parseConditionalFlow()
			if flow != nil {
				stmt.Flow = append(stmt.Flow, flow)
			}
		case COMMENT:
			p.nextToken()
		case NEWLINE:
			p.nextToken()
		default:
			p.error(fmt.Sprintf("unexpected token in scenario block: %s", tokenTypeNames[p.currentToken.Type]))
			p.nextToken()
		}

		p.skipCommentsAndNewlines()
	}

	return stmt
}

// parseLoadConfig parses load configuration
func (p *Parser) parseLoadConfig() *LoadConfig {
	config := &LoadConfig{
		Pos: Position{Line: p.currentToken.Line, Column: p.currentToken.Column},
	}

	p.nextToken()

	// Shorthand: load 10 vus for 5m
	if p.currentTokenIs(NUMBER) {
		num, _ := strconv.Atoi(p.currentToken.Literal)

		if p.peekTokenIs(VUS) {
			config.VUs = num
			p.nextToken() // consume number
			p.nextToken() // consume 'vus'

			if p.currentTokenIs(FOR) {
				p.nextToken() // consume 'for'
				if p.currentTokenIs(DURATION) {
					config.Duration = p.currentToken.Literal
					p.nextToken() // consume duration
				}
			}
		} else if p.peekTokenIs(RPS) {
			config.RPS = num
			p.nextToken() // consume number
			p.nextToken() // consume 'rps'

			if p.currentTokenIs(FOR) {
				p.nextToken() // consume 'for'
				if p.currentTokenIs(DURATION) {
					config.Duration = p.currentToken.Literal
					p.nextToken() // consume duration
				}
			}
		} else if p.peekTokenIs(ITERATIONS) {
			config.Iterations = num
			p.nextToken() // consume number
			p.nextToken() // consume 'iterations'

			if p.currentTokenIs(WITH) {
				p.nextToken() // consume 'with'
				if p.currentTokenIs(NUMBER) {
					config.VUs, _ = strconv.Atoi(p.currentToken.Literal)
					p.nextToken() // consume number
					if p.currentTokenIs(VUS) {
						p.nextToken() // consume 'vus'
					}
				}
			}
		}

		return config
	}

	// Block style: load { vus = 10, duration = 5m }
	if p.currentTokenIs(LBRACE) {
		p.nextToken()
		p.skipNewlines()

		for !p.currentTokenIs(RBRACE) && !p.currentTokenIs(EOF) {
			if p.currentTokenIs(IDENT) || p.currentTokenIs(VUS) || p.currentTokenIs(RPS) {
				key := p.currentToken.Literal

				if !p.expectPeek(ASSIGN) {
					return config
				}

				p.nextToken()

				switch key {
				case "vus":
					if p.currentTokenIs(NUMBER) {
						config.VUs, _ = strconv.Atoi(p.currentToken.Literal)
					}
				case "rps":
					if p.currentTokenIs(NUMBER) {
						config.RPS, _ = strconv.Atoi(p.currentToken.Literal)
					}
				case "iterations":
					if p.currentTokenIs(NUMBER) {
						config.Iterations, _ = strconv.Atoi(p.currentToken.Literal)
					}
				case "duration":
					if p.currentTokenIs(DURATION) {
						config.Duration = p.currentToken.Literal
					}
				}
			}

			p.nextToken()
			p.skipNewlines()
		}

		// Consume the closing brace
		if p.currentTokenIs(RBRACE) {
			p.nextToken()
		}
	}

	return config
}

// parseFlowStatement parses a flow statement
func (p *Parser) parseFlowStatement() FlowStatement {
	pos := Position{Line: p.currentToken.Line, Column: p.currentToken.Column}

	p.nextToken() // consume 'run'

	if !p.currentTokenIs(IDENT) {
		p.error("expected identifier after 'run'")
		return nil
	}

	firstRequest := p.currentToken.Literal

	// Check for sequential flow: run req1 -> req2 -> req3
	if p.peekTokenIs(ARROW) {
		steps := []string{firstRequest}

		for p.peekTokenIs(ARROW) {
			p.nextToken() // consume current
			p.nextToken() // consume '->'

			if !p.currentTokenIs(IDENT) {
				p.error("expected identifier after '->'")
				break
			}

			steps = append(steps, p.currentToken.Literal)
		}

		// Consume the last identifier
		p.nextToken()

		return &SequentialFlow{
			Steps: steps,
			Pos:   pos,
		}
	}

	// Check for nested flow: run parent { run child }
	if p.peekTokenIs(LBRACE) {
		p.nextToken() // consume identifier
		p.nextToken() // consume '{'
		p.skipNewlines()

		children := []FlowStatement{}

		for !p.currentTokenIs(RBRACE) && !p.currentTokenIs(EOF) {
			if p.currentTokenIs(RUN) {
				child := p.parseFlowStatement()
				if child != nil {
					children = append(children, child)
				}
			} else if p.currentTokenIs(IF) {
				child := p.parseConditionalFlow()
				if child != nil {
					children = append(children, child)
				}
			}

			p.nextToken()
			p.skipNewlines()
		}

		// Consume the closing brace
		if p.currentTokenIs(RBRACE) {
			p.nextToken()
		}

		return &NestedFlow{
			Parent:   firstRequest,
			Children: children,
			Pos:      pos,
		}
	}

	// Simple run statement
	// Consume the identifier
	p.nextToken()

	return &RunStatement{
		RequestName: firstRequest,
		Pos:         pos,
	}
}

// parseConditionalFlow parses if/else flow
func (p *Parser) parseConditionalFlow() *ConditionalFlow {
	flow := &ConditionalFlow{
		Pos:       Position{Line: p.currentToken.Line, Column: p.currentToken.Column},
		ThenBlock: []FlowStatement{},
		ElseBlock: []FlowStatement{},
	}

	p.nextToken() // consume 'if'

	// Parse condition
	flow.Condition = p.parseCondition()

	// parseCondition leaves us on the right value, advance past it
	p.nextToken()

	// Skip any newlines before the opening brace
	p.skipNewlines()

	if !p.currentTokenIs(LBRACE) {
		p.error("expected '{' after condition")
		return nil
	}

	p.nextToken()
	p.skipNewlines()

	// Parse then block
	for !p.currentTokenIs(RBRACE) && !p.currentTokenIs(EOF) {
		if p.currentTokenIs(RUN) {
			stmt := p.parseFlowStatement()
			if stmt != nil {
				flow.ThenBlock = append(flow.ThenBlock, stmt)
			}
		}
		p.nextToken()
		p.skipNewlines()
	}

	// Now on RBRACE of then block
	// Check for else block
	if p.peekTokenIs(ELSE) {
		p.nextToken() // consume '}', now on 'else'
		p.nextToken() // consume 'else', now on '{'

		if !p.currentTokenIs(LBRACE) {
			p.error("expected '{' after 'else'")
			return flow
		}

		p.nextToken() // consume '{'
		p.skipNewlines()

		for !p.currentTokenIs(RBRACE) && !p.currentTokenIs(EOF) {
			if p.currentTokenIs(RUN) {
				stmt := p.parseFlowStatement()
				if stmt != nil {
					flow.ElseBlock = append(flow.ElseBlock, stmt)
				}
			}
			p.nextToken()
			p.skipNewlines()
		}

		// Consume else block's closing brace
		if p.currentTokenIs(RBRACE) {
			p.nextToken()
		}
	} else {
		// No else block, consume then block's closing brace
		if p.currentTokenIs(RBRACE) {
			p.nextToken()
		}
	}

	return flow
}

// parseCondition parses a condition expression
func (p *Parser) parseCondition() *Condition {
	cond := &Condition{
		Pos: Position{Line: p.currentToken.Line, Column: p.currentToken.Column},
	}

	cond.Left = p.parseExpression()

	p.nextToken()

	// Operator
	cond.Operator = p.currentToken.Literal

	p.nextToken()

	cond.Right = p.parseExpression()

	return cond
}

// parseExpression parses an expression
func (p *Parser) parseExpression() Expression {
	switch p.currentToken.Type {
	case STRING:
		return &StringLiteral{
			Value: p.currentToken.Literal,
			Pos:   Position{Line: p.currentToken.Line, Column: p.currentToken.Column},
		}
	case NUMBER:
		val, _ := strconv.Atoi(p.currentToken.Literal)
		return &NumberLiteral{
			Value: val,
			Pos:   Position{Line: p.currentToken.Line, Column: p.currentToken.Column},
		}
	case DURATION:
		return &DurationLiteral{
			Value: p.currentToken.Literal,
			Pos:   Position{Line: p.currentToken.Line, Column: p.currentToken.Column},
		}
	case VAR_REF:
		return &VariableReference{
			Name: p.currentToken.Literal,
			Pos:  Position{Line: p.currentToken.Line, Column: p.currentToken.Column},
		}
	case IDENT:
		return &Identifier{
			Name: p.currentToken.Literal,
			Pos:  Position{Line: p.currentToken.Line, Column: p.currentToken.Column},
		}
	case TRUE:
		return &BooleanLiteral{
			Value: true,
			Pos:   Position{Line: p.currentToken.Line, Column: p.currentToken.Column},
		}
	case FALSE:
		return &BooleanLiteral{
			Value: false,
			Pos:   Position{Line: p.currentToken.Line, Column: p.currentToken.Column},
		}
	default:
		p.error(fmt.Sprintf("unexpected expression token: %s", tokenTypeNames[p.currentToken.Type]))
		return nil
	}
}
