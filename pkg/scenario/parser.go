package scenario

import (
	"bufio"
	"fmt"
	"regexp"
	"strings"
)

// Parser parses .httpx scenario files
type Parser struct {
	scanner *bufio.Scanner
	current string
	line    int
}

// NewParser creates a new scenario parser
func NewParser(input string) *Parser {
	return &Parser{
		scanner: bufio.NewScanner(strings.NewReader(input)),
		line:    0,
	}
}

// Parse parses the input and returns a Scenario
func (p *Parser) Parse() (*Scenario, error) {
	scenario := &Scenario{
		Variables: make(map[string]string),
		Data:      make(map[string][]map[string]any),
		Requests:  make(map[string]*Request),
		Scenarios: make(map[string]*ScenarioDefinition),
	}

	for p.scanner.Scan() {
		p.line++
		p.current = strings.TrimSpace(p.scanner.Text())

		// Skip empty lines and comments
		if p.current == "" || strings.HasPrefix(p.current, "#") {
			continue
		}

		// Parse top-level blocks
		if err := p.parseBlock(scenario); err != nil {
			return nil, fmt.Errorf("line %d: %w", p.line, err)
		}
	}

	if err := p.scanner.Err(); err != nil {
		return nil, err
	}

	return scenario, nil
}

func (p *Parser) parseBlock(scenario *Scenario) error {
	// Variable definition: var name = value
	if strings.HasPrefix(p.current, "var ") {
		return p.parseVariable(scenario)
	}

	// Data definition: data name = [...]
	if strings.HasPrefix(p.current, "data ") {
		return p.parseData(scenario)
	}

	// Request definition: request name { ... } or req name: curl ...
	if strings.HasPrefix(p.current, "request ") || strings.HasPrefix(p.current, "req ") {
		return p.parseRequest(scenario)
	}

	// Scenario definition: scenario name { ... }
	if strings.HasPrefix(p.current, "scenario ") {
		return p.parseScenario(scenario)
	}

	// Setup/teardown
	if strings.HasPrefix(p.current, "setup {") {
		return p.parseSetupTeardown(scenario, true)
	}

	if strings.HasPrefix(p.current, "teardown {") {
		return p.parseSetupTeardown(scenario, false)
	}

	return fmt.Errorf("unexpected line: %s", p.current)
}

func (p *Parser) parseVariable(scenario *Scenario) error {
	// var name = value
	re := regexp.MustCompile(`var\s+(\w+)\s*=\s*(.+)`)
	matches := re.FindStringSubmatch(p.current)
	if len(matches) != 3 {
		return fmt.Errorf("invalid variable definition: %s", p.current)
	}

	name := matches[1]
	value := strings.Trim(matches[2], `"'`)
	scenario.Variables[name] = value

	return nil
}

func (p *Parser) parseData(scenario *Scenario) error {
	// Simplified: data name = [...]
	// For now, just mark as placeholder
	// Real implementation would parse JSON/array
	re := regexp.MustCompile(`data\s+(\w+)\s*=`)
	matches := re.FindStringSubmatch(p.current)
	if len(matches) != 2 {
		return fmt.Errorf("invalid data definition: %s", p.current)
	}

	// name := matches[1]
	// TODO: Parse array data
	// scenario.Data[name] = []map[string]any{}

	return nil
}

func (p *Parser) parseRequest(scenario *Scenario) error {
	// request name { ... } or req name: curl ...
	var name string
	var isBlock bool

	// Check for shorthand: req name: curl ...
	if strings.Contains(p.current, ":") {
		re := regexp.MustCompile(`req\s+(\w+):\s*(.+)`)
		matches := re.FindStringSubmatch(p.current)
		if len(matches) == 3 {
			name = matches[1]
			curlCmd := matches[2]

			// Remove trailing pipes and extract/assert if present
			parts := strings.Split(curlCmd, "|")
			curlCmd = strings.TrimSpace(parts[0])

			req := &Request{
				Name:    name,
				CurlCmd: curlCmd,
				Extract: make(map[string]string),
			}

			// Parse inline pipes: | extract ... | assert ...
			for i := 1; i < len(parts); i++ {
				part := strings.TrimSpace(parts[i])
				if strings.HasPrefix(part, "extract ") {
					p.parseExtractInline(req, part)
				} else if strings.HasPrefix(part, "assert ") {
					p.parseAssertInline(req, part)
				}
			}

			scenario.Requests[name] = req
			return nil
		}
	}

	// Block style: request name { ... }
	re := regexp.MustCompile(`request\s+(\w+)\s*\{`)
	matches := re.FindStringSubmatch(p.current)
	if len(matches) == 2 {
		name = matches[1]
		isBlock = true
	} else {
		return fmt.Errorf("invalid request definition: %s", p.current)
	}

	if isBlock {
		req := &Request{
			Name:    name,
			Extract: make(map[string]string),
		}

		// Read block until }
		var curlLines []string
		for p.scanner.Scan() {
			p.line++
			line := strings.TrimSpace(p.scanner.Text())

			if line == "}" || line == "end" {
				break
			}

			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}

			// curl command (multi-line)
			if strings.HasPrefix(line, "curl ") || (len(curlLines) > 0 && strings.HasSuffix(curlLines[len(curlLines)-1], "\\")) {
				curlLines = append(curlLines, strings.TrimSuffix(line, "\\"))
				continue
			}

			// Parse block directives
			if strings.HasPrefix(line, "extract {") {
				if err := p.parseExtractBlock(req); err != nil {
					return err
				}
				continue
			}

			if strings.HasPrefix(line, "extract ") {
				p.parseExtractInline(req, line)
				continue
			}

			if strings.HasPrefix(line, "assert {") {
				if err := p.parseAssertBlock(req); err != nil {
					return err
				}
				continue
			}

			if strings.HasPrefix(line, "assert ") {
				p.parseAssertInline(req, line)
				continue
			}

			if strings.HasPrefix(line, "retry {") {
				if err := p.parseRetryBlock(req); err != nil {
					return err
				}
				continue
			}
		}

		req.CurlCmd = strings.Join(curlLines, " ")
		scenario.Requests[name] = req
	}

	return nil
}

func (p *Parser) parseExtractInline(req *Request, line string) {
	// extract token=$.access_token, user_id=$.user.id
	line = strings.TrimPrefix(line, "extract ")
	parts := strings.Split(line, ",")
	for _, part := range parts {
		kv := strings.Split(strings.TrimSpace(part), "=")
		if len(kv) == 2 {
			req.Extract[strings.TrimSpace(kv[0])] = strings.TrimSpace(kv[1])
		}
	}
}

func (p *Parser) parseExtractBlock(req *Request) error {
	for p.scanner.Scan() {
		p.line++
		line := strings.TrimSpace(p.scanner.Text())

		if line == "}" {
			break
		}

		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// token = $.access_token
		kv := strings.Split(line, "=")
		if len(kv) == 2 {
			req.Extract[strings.TrimSpace(kv[0])] = strings.TrimSpace(kv[1])
		}
	}
	return nil
}

func (p *Parser) parseAssertInline(req *Request, line string) {
	// assert status==200, latency<500ms
	line = strings.TrimPrefix(line, "assert ")
	parts := strings.Split(line, ",")
	for _, part := range parts {
		assertion := p.parseAssertion(strings.TrimSpace(part))
		if assertion != nil {
			req.Assert = append(req.Assert, *assertion)
		}
	}
}

func (p *Parser) parseAssertBlock(req *Request) error {
	for p.scanner.Scan() {
		p.line++
		line := strings.TrimSpace(p.scanner.Text())

		if line == "}" {
			break
		}

		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		assertion := p.parseAssertion(line)
		if assertion != nil {
			req.Assert = append(req.Assert, *assertion)
		}
	}
	return nil
}

func (p *Parser) parseAssertion(line string) *Assertion {
	// status == 200
	// latency < 500ms
	// body.success == true

	for _, op := range []string{"==", "!=", "<", ">", "<=", ">=", "contains", "in"} {
		if strings.Contains(line, op) {
			parts := strings.Split(line, op)
			if len(parts) == 2 {
				field := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])

				assertType := AssertStatus
				if strings.HasPrefix(field, "body.") {
					assertType = AssertBody
				} else if strings.HasPrefix(field, "header.") {
					assertType = AssertHeader
				} else if field == "latency" || strings.HasPrefix(field, "latency_ms") {
					assertType = AssertLatency
				}

				return &Assertion{
					Type:     assertType,
					Field:    field,
					Operator: op,
					Value:    value,
				}
			}
		}
	}

	return nil
}

func (p *Parser) parseRetryBlock(req *Request) error {
	req.Retry = &RetryConfig{
		Backoff: BackoffExponential,
	}

	for p.scanner.Scan() {
		p.line++
		line := strings.TrimSpace(p.scanner.Text())

		if line == "}" {
			break
		}

		// Parse retry config: max_attempts = 5
		kv := strings.Split(line, "=")
		if len(kv) == 2 {
			key := strings.TrimSpace(kv[0])
			value := strings.TrimSpace(kv[1])

			switch key {
			case "max_attempts":
				fmt.Sscanf(value, "%d", &req.Retry.MaxAttempts)
			case "backoff":
				req.Retry.Backoff = BackoffStrategy(value)
			case "base_delay":
				req.Retry.BaseDelay = value
			case "max_delay":
				req.Retry.MaxDelay = value
			}
		}
	}

	return nil
}

func (p *Parser) parseScenario(scenario *Scenario) error {
	// scenario name { ... }
	re := regexp.MustCompile(`scenario\s+(\w+)\s*\{`)
	matches := re.FindStringSubmatch(p.current)
	if len(matches) != 2 {
		return fmt.Errorf("invalid scenario definition: %s", p.current)
	}

	name := matches[1]
	scenarioDef := &ScenarioDefinition{
		Name: name,
	}

	// Parse scenario block
	for p.scanner.Scan() {
		p.line++
		line := strings.TrimSpace(p.scanner.Text())

		if line == "}" || line == "end" {
			break
		}

		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// load { ... } or load: ...
		if strings.HasPrefix(line, "load {") || strings.HasPrefix(line, "load:") || strings.HasPrefix(line, "load ") {
			if err := p.parseLoad(scenarioDef, line); err != nil {
				return err
			}
			continue
		}

		// run ... (flow definition)
		if strings.HasPrefix(line, "run ") {
			if err := p.parseFlow(scenarioDef, line); err != nil {
				return err
			}
			continue
		}
	}

	scenario.Scenarios[name] = scenarioDef
	return nil
}

func (p *Parser) parseLoad(scenarioDef *ScenarioDefinition, line string) error {
	scenarioDef.Load = &LoadConfig{}

	// Shorthand: load 10 vus for 5m
	re := regexp.MustCompile(`load\s+(\d+)\s+vus\s+for\s+(\S+)`)
	matches := re.FindStringSubmatch(line)
	if len(matches) == 3 {
		fmt.Sscanf(matches[1], "%d", &scenarioDef.Load.VUs)
		scenarioDef.Load.Duration = matches[2]
		return nil
	}

	// Inline: load: vus=10, duration=5m
	if strings.Contains(line, ":") {
		parts := strings.Split(line, ":")[1]
		for _, part := range strings.Split(parts, ",") {
			kv := strings.Split(strings.TrimSpace(part), "=")
			if len(kv) == 2 {
				key := strings.TrimSpace(kv[0])
				value := strings.TrimSpace(kv[1])

				switch key {
				case "vus":
					fmt.Sscanf(value, "%d", &scenarioDef.Load.VUs)
				case "duration":
					scenarioDef.Load.Duration = value
				case "rps":
					fmt.Sscanf(value, "%d", &scenarioDef.Load.RPS)
				}
			}
		}
		return nil
	}

	// Block style: load { ... }
	if strings.HasPrefix(line, "load {") {
		for p.scanner.Scan() {
			p.line++
			line := strings.TrimSpace(p.scanner.Text())

			if line == "}" {
				break
			}

			kv := strings.Split(line, "=")
			if len(kv) == 2 {
				key := strings.TrimSpace(kv[0])
				value := strings.TrimSpace(kv[1])

				switch key {
				case "vus":
					fmt.Sscanf(value, "%d", &scenarioDef.Load.VUs)
				case "duration":
					scenarioDef.Load.Duration = value
				case "rps":
					fmt.Sscanf(value, "%d", &scenarioDef.Load.RPS)
				case "iterations":
					fmt.Sscanf(value, "%d", &scenarioDef.Load.Iterations)
				}
			}
		}
	}

	return nil
}

func (p *Parser) parseFlow(scenarioDef *ScenarioDefinition, line string) error {
	// run login -> get_profile
	// run login { run get_profile }

	line = strings.TrimPrefix(line, "run ")

	// Sequential with ->
	if strings.Contains(line, "->") {
		steps := strings.Split(line, "->")
		flow := &Flow{
			Type:  FlowSequential,
			Steps: make([]string, 0),
		}
		for _, step := range steps {
			flow.Steps = append(flow.Steps, strings.TrimSpace(step))
		}
		scenarioDef.Flow = flow
		return nil
	}

	// Single step
	scenarioDef.Flow = &Flow{
		Type:  FlowSequential,
		Steps: []string{strings.TrimSpace(line)},
	}

	return nil
}

func (p *Parser) parseSetupTeardown(scenario *Scenario, isSetup bool) error {
	steps := make([]string, 0)

	for p.scanner.Scan() {
		p.line++
		line := strings.TrimSpace(p.scanner.Text())

		if line == "}" {
			break
		}

		if strings.HasPrefix(line, "run ") {
			step := strings.TrimPrefix(line, "run ")
			steps = append(steps, strings.TrimSpace(step))
		}
	}

	if isSetup {
		scenario.Setup = steps
	} else {
		scenario.Teardown = steps
	}

	return nil
}
