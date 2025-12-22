package scenario

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/vikasavnish/httptool/pkg/ir"
	"github.com/vikasavnish/httptool/pkg/parser"
)

// Compiler compiles scenarios to executable IR trees
type Compiler struct {
	parser *parser.CurlParser
	vars   map[string]string
}

// NewCompiler creates a new scenario compiler
func NewCompiler() *Compiler {
	return &Compiler{
		parser: parser.NewCurlParser(),
		vars:   make(map[string]string),
	}
}

// Compile compiles a scenario to executable form
func (c *Compiler) Compile(scenario *Scenario, scenarioName string) (*CompiledScenario, error) {
	scenarioDef, ok := scenario.Scenarios[scenarioName]
	if !ok {
		return nil, fmt.Errorf("scenario '%s' not found", scenarioName)
	}

	// Merge global variables
	for k, v := range scenario.Variables {
		c.vars[k] = v
	}

	compiled := &CompiledScenario{
		Name:      scenarioName,
		Load:      scenarioDef.Load,
		Variables: c.vars,
	}

	// Compile setup
	for _, setupReq := range scenario.Setup {
		request, ok := scenario.Requests[setupReq]
		if !ok {
			return nil, fmt.Errorf("setup request '%s' not found", setupReq)
		}

		irSpec, err := c.compileRequest(request)
		if err != nil {
			return nil, fmt.Errorf("failed to compile setup '%s': %w", setupReq, err)
		}

		compiled.Setup = append(compiled.Setup, irSpec)
	}

	// Compile main flow
	if scenarioDef.Flow != nil {
		nodes, err := c.compileFlow(scenario, scenarioDef.Flow)
		if err != nil {
			return nil, fmt.Errorf("failed to compile flow: %w", err)
		}
		compiled.Main = nodes
	}

	// Compile teardown
	for _, teardownReq := range scenario.Teardown {
		request, ok := scenario.Requests[teardownReq]
		if !ok {
			return nil, fmt.Errorf("teardown request '%s' not found", teardownReq)
		}

		irSpec, err := c.compileRequest(request)
		if err != nil {
			return nil, fmt.Errorf("failed to compile teardown '%s': %w", teardownReq, err)
		}

		compiled.Teardown = append(compiled.Teardown, irSpec)
	}

	return compiled, nil
}

func (c *Compiler) compileFlow(scenario *Scenario, flow *Flow) ([]*RequestNode, error) {
	var nodes []*RequestNode

	for _, stepName := range flow.Steps {
		request, ok := scenario.Requests[stepName]
		if !ok {
			return nil, fmt.Errorf("request '%s' not found", stepName)
		}

		node, err := c.compileRequestNode(scenario, request)
		if err != nil {
			return nil, fmt.Errorf("failed to compile request '%s': %w", stepName, err)
		}

		nodes = append(nodes, node)
	}

	return nodes, nil
}

func (c *Compiler) compileRequestNode(scenario *Scenario, request *Request) (*RequestNode, error) {
	// Compile curl to IR
	irSpec, err := c.compileRequest(request)
	if err != nil {
		return nil, err
	}

	node := &RequestNode{
		IR:        irSpec,
		Extract:   request.Extract,
		Assert:    request.Assert,
		Condition: request.Condition,
		Parallel:  request.Parallel,
	}

	// Compile children
	for _, childName := range request.Children {
		childReq, ok := scenario.Requests[childName]
		if !ok {
			return nil, fmt.Errorf("child request '%s' not found", childName)
		}

		childNode, err := c.compileRequestNode(scenario, childReq)
		if err != nil {
			return nil, fmt.Errorf("failed to compile child '%s': %w", childName, err)
		}

		node.Children = append(node.Children, childNode)
	}

	return node, nil
}

func (c *Compiler) compileRequest(request *Request) (*ir.IR, error) {
	// Replace variables in curl command
	curlCmd := c.replaceVariables(request.CurlCmd)

	// Parse curl to IR
	irSpec, err := c.parser.Parse(curlCmd)
	if err != nil {
		return nil, fmt.Errorf("failed to parse curl: %w", err)
	}

	// Add metadata
	if irSpec.Metadata == nil {
		irSpec.Metadata = &ir.Metadata{}
	}
	irSpec.Metadata.Source = "scenario"

	// Configure retry if specified
	if request.Retry != nil {
		// Store retry config in evaluation vars
		if irSpec.Evaluation == nil {
			irSpec.Evaluation = ir.DefaultEvaluation()
		}
		if irSpec.Evaluation.Vars == nil {
			irSpec.Evaluation.Vars = make(map[string]any)
		}

		irSpec.Evaluation.Vars["retry_max_attempts"] = request.Retry.MaxAttempts
		irSpec.Evaluation.Vars["retry_backoff"] = string(request.Retry.Backoff)
		irSpec.Evaluation.Vars["retry_base_delay"] = request.Retry.BaseDelay
		irSpec.Evaluation.Vars["retry_max_delay"] = request.Retry.MaxDelay
	}

	return irSpec, nil
}

func (c *Compiler) replaceVariables(input string) string {
	result := input

	// Replace ${var} with actual values
	re := regexp.MustCompile(`\$\{(\w+)\}`)
	result = re.ReplaceAllStringFunc(result, func(match string) string {
		varName := match[2 : len(match)-1] // Remove ${ and }

		// Check for built-in variables
		switch varName {
		case "VU", "__VU":
			return "${__VU}" // Preserve for runtime replacement
		case "ITER", "__ITER":
			return "${__ITER}"
		case "TIME", "__TIME":
			return "${__TIME}"
		case "UUID", "__RANDOM":
			return "${__RANDOM}"
		case "COUNTER", "__COUNTER":
			return "${__COUNTER}"
		}

		// Check user variables
		if value, ok := c.vars[varName]; ok {
			return value
		}

		// Check environment variables
		if strings.HasPrefix(varName, "env.") {
			// Will be replaced at runtime
			return match
		}

		// Preserve extracted variables for runtime
		return match
	})

	return result
}

// ReplaceRuntimeVariables replaces variables at execution time
func ReplaceRuntimeVariables(input string, vu int, iter int, extractedVars map[string]any) string {
	result := input

	// Replace built-in variables
	result = strings.ReplaceAll(result, "${__VU}", fmt.Sprintf("%d", vu))
	result = strings.ReplaceAll(result, "${VU}", fmt.Sprintf("%d", vu))
	result = strings.ReplaceAll(result, "${__ITER}", fmt.Sprintf("%d", iter))
	result = strings.ReplaceAll(result, "${ITER}", fmt.Sprintf("%d", iter))

	// Replace extracted variables
	for key, value := range extractedVars {
		placeholder := fmt.Sprintf("${%s}", key)
		result = strings.ReplaceAll(result, placeholder, fmt.Sprintf("%v", value))
	}

	return result
}
