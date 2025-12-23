package parser

import (
	"testing"
)

func TestParser_VariableDeclaration(t *testing.T) {
	input := `var base_url = "https://api.example.com"`

	l := NewLexer(input)
	p := NewParser(l)
	program := p.Parse()

	checkParserErrors(t, p)

	if len(program.Statements) != 1 {
		t.Fatalf("program.Statements does not contain 1 statement. got=%d", len(program.Statements))
	}

	stmt, ok := program.Statements[0].(*VariableDeclaration)
	if !ok {
		t.Fatalf("statement is not *VariableDeclaration. got=%T", program.Statements[0])
	}

	if stmt.Name != "base_url" {
		t.Errorf("stmt.Name not 'base_url'. got=%s", stmt.Name)
	}

	strLit, ok := stmt.Value.(*StringLiteral)
	if !ok {
		t.Fatalf("stmt.Value is not *StringLiteral. got=%T", stmt.Value)
	}

	if strLit.Value != "https://api.example.com" {
		t.Errorf("strLit.Value not 'https://api.example.com'. got=%s", strLit.Value)
	}
}

func TestParser_RequestDeclaration(t *testing.T) {
	input := `request get_user {
	curl https://api.example.com/users/123

	assert status == 200
	assert body.name != null
}`

	l := NewLexer(input)
	p := NewParser(l)
	program := p.Parse()

	checkParserErrors(t, p)

	if len(program.Statements) != 1 {
		t.Fatalf("program.Statements does not contain 1 statement. got=%d", len(program.Statements))
	}

	stmt, ok := program.Statements[0].(*RequestDeclaration)
	if !ok {
		t.Fatalf("statement is not *RequestDeclaration. got=%T", program.Statements[0])
	}

	if stmt.Name != "get_user" {
		t.Errorf("stmt.Name not 'get_user'. got=%s", stmt.Name)
	}

	if stmt.CurlCommand == nil {
		t.Fatal("stmt.CurlCommand is nil")
	}

	if stmt.CurlCommand.URL != "https://api.example.com/users/123" {
		t.Errorf("curl URL wrong. got=%s", stmt.CurlCommand.URL)
	}

	if len(stmt.Assertions) != 2 {
		t.Fatalf("expected 2 assertions. got=%d", len(stmt.Assertions))
	}
}

func TestParser_CurlWithHeaders(t *testing.T) {
	input := `request login {
	curl 'https://api.example.com/login' \
		-H 'Content-Type: application/json' \
		-d '{"user":"admin"}'
}`

	l := NewLexer(input)
	p := NewParser(l)
	program := p.Parse()

	checkParserErrors(t, p)

	stmt, ok := program.Statements[0].(*RequestDeclaration)
	if !ok {
		t.Fatalf("statement is not *RequestDeclaration. got=%T", program.Statements[0])
	}

	curl := stmt.CurlCommand
	if curl == nil {
		t.Fatal("CurlCommand is nil")
	}

	if curl.URL != "https://api.example.com/login" {
		t.Errorf("URL wrong. got=%s", curl.URL)
	}

	if curl.Headers["Content-Type"] != "application/json" {
		t.Errorf("Content-Type header wrong. got=%s", curl.Headers["Content-Type"])
	}

	if curl.Body != `{"user":"admin"}` {
		t.Errorf("Body wrong. got=%s", curl.Body)
	}

	if curl.Method != "POST" {
		t.Errorf("Method wrong. got=%s", curl.Method)
	}
}

func TestParser_CurlWithVariableRef(t *testing.T) {
	input := `request test {
	curl ${base_url}/api/users
}`

	l := NewLexer(input)
	p := NewParser(l)
	program := p.Parse()

	checkParserErrors(t, p)

	stmt := program.Statements[0].(*RequestDeclaration)
	curl := stmt.CurlCommand

	if curl.URL != "${base_url}/api/users" {
		t.Errorf("URL wrong. got=%s", curl.URL)
	}

	if len(curl.URLParts) != 2 {
		t.Fatalf("expected 2 URL parts. got=%d", len(curl.URLParts))
	}

	varRef, ok := curl.URLParts[0].(*VariableReference)
	if !ok {
		t.Fatalf("URLParts[0] not *VariableReference. got=%T", curl.URLParts[0])
	}

	if varRef.Name != "base_url" {
		t.Errorf("variable name wrong. got=%s", varRef.Name)
	}
}

func TestParser_ExtractBlock(t *testing.T) {
	input := `request get_data {
	curl https://api.example.com/data

	extract {
		user_id = $.data.user.id
		session = cookie:session_token
		auth_header = header:Authorization
	}
}`

	l := NewLexer(input)
	p := NewParser(l)
	program := p.Parse()

	checkParserErrors(t, p)

	stmt := program.Statements[0].(*RequestDeclaration)

	if len(stmt.Extractions) != 3 {
		t.Fatalf("expected 3 extractions. got=%d", len(stmt.Extractions))
	}

	// Check JSONPath extraction
	if stmt.Extractions[0].Variable != "user_id" {
		t.Errorf("extraction variable wrong. got=%s", stmt.Extractions[0].Variable)
	}
	if stmt.Extractions[0].Type != ExtractJSONPath {
		t.Errorf("extraction type wrong. got=%d", stmt.Extractions[0].Type)
	}

	// Check cookie extraction
	if stmt.Extractions[1].Type != ExtractCookie {
		t.Errorf("extraction type wrong. got=%d", stmt.Extractions[1].Type)
	}

	// Check header extraction
	if stmt.Extractions[2].Type != ExtractHeader {
		t.Errorf("extraction type wrong. got=%d", stmt.Extractions[2].Type)
	}
}

func TestParser_ScenarioWithLoadConfig(t *testing.T) {
	input := `scenario load_test {
	load 50 vus for 2m
	run get_user
}`

	l := NewLexer(input)
	p := NewParser(l)
	program := p.Parse()

	checkParserErrors(t, p)

	stmt, ok := program.Statements[0].(*ScenarioDeclaration)
	if !ok {
		t.Fatalf("statement is not *ScenarioDeclaration. got=%T", program.Statements[0])
	}

	if stmt.Name != "load_test" {
		t.Errorf("scenario name wrong. got=%s", stmt.Name)
	}

	if stmt.LoadConfig.VUs != 50 {
		t.Errorf("VUs wrong. got=%d", stmt.LoadConfig.VUs)
	}

	if stmt.LoadConfig.Duration != "2m" {
		t.Errorf("Duration wrong. got=%s", stmt.LoadConfig.Duration)
	}

	if len(stmt.Flow) != 1 {
		t.Fatalf("expected 1 flow statement. got=%d", len(stmt.Flow))
	}
}

func TestParser_ScenarioWithRPS(t *testing.T) {
	input := `scenario rps_test {
	load 100 rps for 30s
	run api_call
}`

	l := NewLexer(input)
	p := NewParser(l)
	program := p.Parse()

	checkParserErrors(t, p)

	stmt := program.Statements[0].(*ScenarioDeclaration)

	if stmt.LoadConfig.RPS != 100 {
		t.Errorf("RPS wrong. got=%d", stmt.LoadConfig.RPS)
	}

	if stmt.LoadConfig.Duration != "30s" {
		t.Errorf("Duration wrong. got=%s", stmt.LoadConfig.Duration)
	}
}

func TestParser_SequentialFlow(t *testing.T) {
	input := `scenario flow_test {
	load 10 vus for 1m
	run login -> get_profile -> update_settings
}`

	l := NewLexer(input)
	p := NewParser(l)
	program := p.Parse()

	checkParserErrors(t, p)

	stmt := program.Statements[0].(*ScenarioDeclaration)
	flow, ok := stmt.Flow[0].(*SequentialFlow)
	if !ok {
		t.Fatalf("flow is not *SequentialFlow. got=%T", stmt.Flow[0])
	}

	if len(flow.Steps) != 3 {
		t.Fatalf("expected 3 steps. got=%d", len(flow.Steps))
	}

	expectedSteps := []string{"login", "get_profile", "update_settings"}
	for i, expected := range expectedSteps {
		if flow.Steps[i] != expected {
			t.Errorf("step %d wrong. expected=%s, got=%s", i, expected, flow.Steps[i])
		}
	}
}

func TestParser_NestedFlow(t *testing.T) {
	input := `scenario nested_test {
	load 5 vus for 30s
	run login {
		run get_profile
		run update_settings
	}
}`

	l := NewLexer(input)
	p := NewParser(l)
	program := p.Parse()

	checkParserErrors(t, p)

	stmt := program.Statements[0].(*ScenarioDeclaration)
	flow, ok := stmt.Flow[0].(*NestedFlow)
	if !ok {
		t.Fatalf("flow is not *NestedFlow. got=%T", stmt.Flow[0])
	}

	if flow.Parent != "login" {
		t.Errorf("parent wrong. got=%s", flow.Parent)
	}

	if len(flow.Children) != 2 {
		t.Fatalf("expected 2 children. got=%d", len(flow.Children))
	}
}

func TestParser_ConditionalFlow(t *testing.T) {
	input := `scenario conditional_test {
	load 10 vus for 1m
	if ${feature_enabled} == "true" {
		run new_api
	} else {
		run old_api
	}
}`

	l := NewLexer(input)
	p := NewParser(l)
	program := p.Parse()

	checkParserErrors(t, p)

	stmt := program.Statements[0].(*ScenarioDeclaration)
	flow, ok := stmt.Flow[0].(*ConditionalFlow)
	if !ok {
		t.Fatalf("flow is not *ConditionalFlow. got=%T", stmt.Flow[0])
	}

	if flow.Condition.Operator != "==" {
		t.Errorf("condition operator wrong. got=%s", flow.Condition.Operator)
	}

	if len(flow.ThenBlock) != 1 {
		t.Fatalf("expected 1 then statement. got=%d", len(flow.ThenBlock))
	}

	if len(flow.ElseBlock) != 1 {
		t.Fatalf("expected 1 else statement. got=%d", len(flow.ElseBlock))
	}
}

func TestParser_AssertionWithIn(t *testing.T) {
	input := `request test {
	curl https://api.example.com
	assert status in [200, 201, 204]
}`

	l := NewLexer(input)
	p := NewParser(l)
	program := p.Parse()

	checkParserErrors(t, p)

	stmt := program.Statements[0].(*RequestDeclaration)

	if len(stmt.Assertions) != 1 {
		t.Fatalf("expected 1 assertion. got=%d", len(stmt.Assertions))
	}

	assertion := stmt.Assertions[0]
	if assertion.Operator != "in" {
		t.Errorf("operator wrong. got=%s", assertion.Operator)
	}

	if len(assertion.Values) != 3 {
		t.Fatalf("expected 3 values. got=%d", len(assertion.Values))
	}
}

func TestParser_LoadBlockStyle(t *testing.T) {
	input := `scenario block_load {
	load {
		vus = 10
		duration = 5m
	}
	run test
}`

	l := NewLexer(input)
	p := NewParser(l)
	program := p.Parse()

	checkParserErrors(t, p)

	stmt := program.Statements[0].(*ScenarioDeclaration)

	if stmt.LoadConfig.VUs != 10 {
		t.Errorf("VUs wrong. got=%d", stmt.LoadConfig.VUs)
	}

	if stmt.LoadConfig.Duration != "5m" {
		t.Errorf("Duration wrong. got=%s", stmt.LoadConfig.Duration)
	}
}

func TestParser_RetryConfig(t *testing.T) {
	input := `request with_retry {
	curl https://api.example.com

	retry {
		max_attempts = 3
		backoff = exponential
		base_delay = 100ms
	}
}`

	l := NewLexer(input)
	p := NewParser(l)
	program := p.Parse()

	checkParserErrors(t, p)

	stmt := program.Statements[0].(*RequestDeclaration)

	if stmt.RetryConfig == nil {
		t.Fatal("RetryConfig is nil")
	}

	if stmt.RetryConfig.MaxAttempts != 3 {
		t.Errorf("MaxAttempts wrong. got=%d", stmt.RetryConfig.MaxAttempts)
	}

	if stmt.RetryConfig.Backoff != "exponential" {
		t.Errorf("Backoff wrong. got=%s", stmt.RetryConfig.Backoff)
	}

	if stmt.RetryConfig.BaseDelay != "100ms" {
		t.Errorf("BaseDelay wrong. got=%s", stmt.RetryConfig.BaseDelay)
	}
}

func TestParser_CompleteScenario(t *testing.T) {
	input := `# Complete scenario example
var base_url = "https://api.example.com"

request login {
	curl ${base_url}/auth/login \
		-H 'Content-Type: application/json' \
		-d '{"user":"admin","pass":"secret"}'

	extract {
		token = $.data.access_token
	}

	assert status == 200
	assert body.success == true
}

request get_users {
	curl ${base_url}/users

	assert status == 200
}

scenario user_flow {
	load 20 vus for 2m
	run login -> get_users
}`

	l := NewLexer(input)
	p := NewParser(l)
	program := p.Parse()

	checkParserErrors(t, p)

	// Should have: 1 var + 2 requests + 1 scenario = 4 statements (comments filtered)
	if len(program.Statements) != 4 {
		t.Fatalf("expected 4 statements. got=%d", len(program.Statements))
	}

	// Check variable
	varDecl, ok := program.Statements[0].(*VariableDeclaration)
	if !ok {
		t.Fatalf("statement 0 is not *VariableDeclaration. got=%T", program.Statements[0])
	}
	if varDecl.Name != "base_url" {
		t.Errorf("variable name wrong. got=%s", varDecl.Name)
	}

	// Check first request
	req1, ok := program.Statements[1].(*RequestDeclaration)
	if !ok {
		t.Fatalf("statement 1 is not *RequestDeclaration. got=%T", program.Statements[1])
	}
	if req1.Name != "login" {
		t.Errorf("request name wrong. got=%s", req1.Name)
	}

	// Check scenario
	scenario, ok := program.Statements[3].(*ScenarioDeclaration)
	if !ok {
		t.Fatalf("statement 3 is not *ScenarioDeclaration. got=%T", program.Statements[3])
	}
	if scenario.Name != "user_flow" {
		t.Errorf("scenario name wrong. got=%s", scenario.Name)
	}
}

func checkParserErrors(t *testing.T, p *Parser) {
	errors := p.Errors()
	if len(errors) == 0 {
		return
	}

	t.Errorf("parser has %d errors", len(errors))
	for _, msg := range errors {
		t.Errorf("parser error: %s", msg)
	}
	t.FailNow()
}
