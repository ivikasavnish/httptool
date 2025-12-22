package scenario

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/vikasavnish/httptool/pkg/evaluator"
	"github.com/vikasavnish/httptool/pkg/executor"
	"github.com/vikasavnish/httptool/pkg/ir"
)

// Executor runs compiled scenarios
type Executor struct {
	httpExecutor *executor.Executor
	evalManager  *evaluator.Manager
}

// NewExecutor creates a new scenario executor
func NewExecutor() *Executor {
	return &Executor{
		httpExecutor: executor.NewExecutor(),
		evalManager:  evaluator.NewManager(5 * time.Second),
	}
}

// Execute runs a compiled scenario
func (e *Executor) Execute(ctx context.Context, scenario *CompiledScenario) (*ScenarioResult, error) {
	result := &ScenarioResult{
		Name:      scenario.Name,
		StartTime: time.Now(),
		VUResults: make([]*VUResult, 0),
	}

	// Run setup
	if len(scenario.Setup) > 0 {
		setupVars := make(map[string]any)
		for _, irSpec := range scenario.Setup {
			execCtx, err := e.httpExecutor.Execute(irSpec)
			if err != nil {
				return nil, fmt.Errorf("setup failed: %w", err)
			}

			// Extract variables from setup
			extractedVars := e.extractVariables(execCtx, nil)
			for k, v := range extractedVars {
				setupVars[k] = v
			}
		}

		// Store setup vars for VUs to use
		result.SetupVars = setupVars
	}

	// Determine execution mode
	if scenario.Load == nil {
		return nil, fmt.Errorf("no load configuration specified")
	}

	// Execute based on load config
	if scenario.Load.VUs > 0 && scenario.Load.Duration != "" {
		e.executeVUs(ctx, scenario, result)
	} else if scenario.Load.RPS > 0 {
		e.executeRPS(ctx, scenario, result)
	} else if scenario.Load.Iterations > 0 {
		e.executeIterations(ctx, scenario, result)
	} else {
		return nil, fmt.Errorf("invalid load configuration")
	}

	result.EndTime = time.Now()

	// Run teardown
	if len(scenario.Teardown) > 0 {
		for _, irSpec := range scenario.Teardown {
			_, err := e.httpExecutor.Execute(irSpec)
			if err != nil {
				// Log but don't fail
				fmt.Printf("Teardown warning: %v\n", err)
			}
		}
	}

	// Calculate stats
	result.Stats = e.calculateStats(result.VUResults)

	return result, nil
}

func (e *Executor) executeVUs(ctx context.Context, scenario *CompiledScenario, result *ScenarioResult) {
	duration, _ := parseDuration(scenario.Load.Duration)
	deadline := time.Now().Add(duration)

	var wg sync.WaitGroup
	var mu sync.Mutex

	// Start VUs
	for vu := 1; vu <= scenario.Load.VUs; vu++ {
		wg.Add(1)
		go func(vuID int) {
			defer wg.Done()

			vuResult := &VUResult{
				VUID:       vuID,
				Iterations: make([]*IterationResult, 0),
			}

			iteration := 1
			for time.Now().Before(deadline) {
				select {
				case <-ctx.Done():
					return
				default:
				}

				iterResult := e.executeIteration(ctx, scenario, vuID, iteration, result.SetupVars)
				vuResult.Iterations = append(vuResult.Iterations, iterResult)
				iteration++
			}

			mu.Lock()
			result.VUResults = append(result.VUResults, vuResult)
			mu.Unlock()
		}(vu)
	}

	wg.Wait()
}

func (e *Executor) executeRPS(ctx context.Context, scenario *CompiledScenario, result *ScenarioResult) {
	duration, _ := parseDuration(scenario.Load.Duration)
	deadline := time.Now().Add(duration)
	ticker := time.NewTicker(time.Second / time.Duration(scenario.Load.RPS))
	defer ticker.Stop()

	var mu sync.Mutex
	vuID := 1
	iteration := 1

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if time.Now().After(deadline) {
				return
			}

			go func(vu, iter int) {
				iterResult := e.executeIteration(ctx, scenario, vu, iter, result.SetupVars)

				mu.Lock()
				// Find or create VU result
				var vuResult *VUResult
				for _, vr := range result.VUResults {
					if vr.VUID == vu {
						vuResult = vr
						break
					}
				}
				if vuResult == nil {
					vuResult = &VUResult{
						VUID:       vu,
						Iterations: make([]*IterationResult, 0),
					}
					result.VUResults = append(result.VUResults, vuResult)
				}
				vuResult.Iterations = append(vuResult.Iterations, iterResult)
				mu.Unlock()
			}(vuID, iteration)

			iteration++
			if iteration%scenario.Load.RPS == 0 {
				vuID++
			}
		}
	}
}

func (e *Executor) executeIterations(ctx context.Context, scenario *CompiledScenario, result *ScenarioResult) {
	vus := scenario.Load.VUs
	if vus == 0 {
		vus = 1
	}

	iterPerVU := scenario.Load.Iterations / vus
	remainder := scenario.Load.Iterations % vus

	var wg sync.WaitGroup
	var mu sync.Mutex

	for vu := 1; vu <= vus; vu++ {
		wg.Add(1)
		iterations := iterPerVU
		if vu <= remainder {
			iterations++
		}

		go func(vuID int, maxIter int) {
			defer wg.Done()

			vuResult := &VUResult{
				VUID:       vuID,
				Iterations: make([]*IterationResult, 0),
			}

			for iter := 1; iter <= maxIter; iter++ {
				select {
				case <-ctx.Done():
					return
				default:
				}

				iterResult := e.executeIteration(ctx, scenario, vuID, iter, result.SetupVars)
				vuResult.Iterations = append(vuResult.Iterations, iterResult)
			}

			mu.Lock()
			result.VUResults = append(result.VUResults, vuResult)
			mu.Unlock()
		}(vu, iterations)
	}

	wg.Wait()
}

func (e *Executor) executeIteration(ctx context.Context, scenario *CompiledScenario, vu int, iter int, setupVars map[string]any) *IterationResult {
	iterResult := &IterationResult{
		IterationNum: iter,
		StartTime:    time.Now(),
		Requests:     make([]*RequestResult, 0),
	}

	// Execution context with extracted variables
	execVars := make(map[string]any)
	for k, v := range setupVars {
		execVars[k] = v
	}

	// Execute request tree
	for _, node := range scenario.Main {
		e.executeNode(ctx, node, vu, iter, execVars, iterResult)
	}

	iterResult.EndTime = time.Now()
	return iterResult
}

func (e *Executor) executeNode(ctx context.Context, node *RequestNode, vu int, iter int, vars map[string]any, iterResult *IterationResult) {
	// Check condition
	if node.Condition != "" && !e.evaluateCondition(node.Condition, vars) {
		return
	}

	// Clone IR and replace runtime variables
	irSpec := e.cloneIRWithVars(node.IR, vu, iter, vars)

	// Execute request
	execCtx, err := e.httpExecutor.Execute(irSpec)

	reqResult := &RequestResult{
		URL:       irSpec.Request.URL,
		Method:    irSpec.Request.Method,
		StartTime: time.Now(),
	}

	if err != nil {
		reqResult.Error = err.Error()
		iterResult.Requests = append(iterResult.Requests, reqResult)
		return
	}

	reqResult.Status = execCtx.Response.Status
	reqResult.Latency = time.Duration(execCtx.Response.LatencyMs * 1000000)
	reqResult.Size = execCtx.Response.SizeBytes

	// Check assertions
	for _, assertion := range node.Assert {
		if !e.checkAssertion(assertion, execCtx) {
			reqResult.AssertionsFailed++
			reqResult.Error = fmt.Sprintf("assertion failed: %s %s %v", assertion.Field, assertion.Operator, assertion.Value)
		}
	}

	// Extract variables
	if len(node.Extract) > 0 {
		extracted := e.extractVariables(execCtx, node.Extract)
		for k, v := range extracted {
			vars[k] = v
		}
	}

	iterResult.Requests = append(iterResult.Requests, reqResult)

	// Execute children
	if len(node.Children) > 0 {
		if node.Parallel {
			var wg sync.WaitGroup
			for _, child := range node.Children {
				wg.Add(1)
				go func(childNode *RequestNode) {
					defer wg.Done()
					e.executeNode(ctx, childNode, vu, iter, vars, iterResult)
				}(child)
			}
			wg.Wait()
		} else {
			for _, child := range node.Children {
				e.executeNode(ctx, child, vu, iter, vars, iterResult)
			}
		}
	}

	// Think time
	if node.ThinkTime != nil {
		thinkDuration, _ := parseDuration(node.ThinkTime.Duration)
		if node.ThinkTime.Variance > 0 {
			variance := node.ThinkTime.Variance
			factor := 1.0 + (rand.Float64()*2-1)*variance
			thinkDuration = time.Duration(float64(thinkDuration) * factor)
		}
		time.Sleep(thinkDuration)
	}
}

func (e *Executor) cloneIRWithVars(irSpec *ir.IR, vu int, iter int, vars map[string]any) *ir.IR {
	// Deep clone IR
	data, _ := json.Marshal(irSpec)
	var cloned ir.IR
	json.Unmarshal(data, &cloned)

	// Replace variables in URL
	cloned.Request.URL = ReplaceRuntimeVariables(cloned.Request.URL, vu, iter, vars)

	// Replace in headers
	for k, v := range cloned.Request.Headers {
		cloned.Request.Headers[k] = ReplaceRuntimeVariables(v, vu, iter, vars)
	}

	// Replace in body
	if cloned.Request.Body != nil {
		if cloned.Request.Body.Type == "json" {
			bodyJSON, _ := json.Marshal(cloned.Request.Body.Content)
			bodyStr := ReplaceRuntimeVariables(string(bodyJSON), vu, iter, vars)
			var newContent any
			json.Unmarshal([]byte(bodyStr), &newContent)
			cloned.Request.Body.Content = newContent
		} else if cloned.Request.Body.Type == "text" {
			if str, ok := cloned.Request.Body.Content.(string); ok {
				cloned.Request.Body.Content = ReplaceRuntimeVariables(str, vu, iter, vars)
			}
		}
	}

	return &cloned
}

func (e *Executor) extractVariables(execCtx *ir.EvaluationContext, extractRules map[string]string) map[string]any {
	extracted := make(map[string]any)

	if extractRules == nil {
		return extracted
	}

	for varName, rule := range extractRules {
		// JSONPath extraction: $.field.subfield
		if strings.HasPrefix(rule, "$.") {
			value := e.extractJSONPath(execCtx.Response.Body, rule)
			if value != nil {
				extracted[varName] = value
			}
		}

		// Regex extraction: regex:pattern
		if strings.HasPrefix(rule, "regex:") {
			pattern := strings.TrimPrefix(rule, "regex:")
			value := e.extractRegex(execCtx.Response.Body, pattern)
			if value != "" {
				extracted[varName] = value
			}
		}

		// Header extraction: header:Header-Name
		if strings.HasPrefix(rule, "header:") {
			headerName := strings.TrimPrefix(rule, "header:")
			if value, ok := execCtx.Response.Headers[headerName]; ok {
				extracted[varName] = value
			}
		}
	}

	return extracted
}

func (e *Executor) extractJSONPath(body any, path string) any {
	// Simplified JSONPath (only supports simple paths like $.field.subfield)
	if bodyMap, ok := body.(map[string]any); ok {
		path = strings.TrimPrefix(path, "$.")
		parts := strings.Split(path, ".")

		current := any(bodyMap)
		for _, part := range parts {
			if m, ok := current.(map[string]any); ok {
				current = m[part]
			} else {
				return nil
			}
		}
		return current
	}
	return nil
}

func (e *Executor) extractRegex(body any, pattern string) string {
	bodyStr := fmt.Sprintf("%v", body)
	re := regexp.MustCompile(pattern)
	matches := re.FindStringSubmatch(bodyStr)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

func (e *Executor) evaluateCondition(condition string, vars map[string]any) bool {
	// Simplified condition evaluation: ${var} == value
	for k, v := range vars {
		placeholder := fmt.Sprintf("${%s}", k)
		condition = strings.ReplaceAll(condition, placeholder, fmt.Sprintf("%v", v))
	}

	// Simple evaluation (extend for complex logic)
	if strings.Contains(condition, "==") {
		parts := strings.Split(condition, "==")
		if len(parts) == 2 {
			return strings.TrimSpace(parts[0]) == strings.TrimSpace(parts[1])
		}
	}

	return true
}

func (e *Executor) checkAssertion(assertion Assertion, execCtx *ir.EvaluationContext) bool {
	switch assertion.Type {
	case AssertStatus:
		expected := fmt.Sprintf("%v", assertion.Value)
		actual := fmt.Sprintf("%d", execCtx.Response.Status)
		return e.compareValues(actual, assertion.Operator, expected)

	case AssertLatency:
		latency := execCtx.Response.LatencyMs
		expected := fmt.Sprintf("%v", assertion.Value)
		// Parse expected (e.g., "500ms", "1s")
		expectedMs := parseLatency(expected)
		return e.compareValues(fmt.Sprintf("%f", latency), assertion.Operator, fmt.Sprintf("%f", expectedMs))

	case AssertBody:
		// Extract field from body
		field := strings.TrimPrefix(assertion.Field, "body.")
		value := e.extractJSONPath(execCtx.Response.Body, "$."+field)
		return e.compareValues(fmt.Sprintf("%v", value), assertion.Operator, fmt.Sprintf("%v", assertion.Value))
	}

	return true
}

func (e *Executor) compareValues(actual, operator, expected string) bool {
	actual = strings.TrimSpace(actual)
	expected = strings.TrimSpace(expected)

	switch operator {
	case "==":
		return actual == expected
	case "!=":
		return actual != expected
	case "contains":
		return strings.Contains(actual, expected)
	// Add more operators as needed
	default:
		return true
	}
}

func (e *Executor) calculateStats(vuResults []*VUResult) *Stats {
	stats := &Stats{}

	for _, vuResult := range vuResults {
		for _, iterResult := range vuResult.Iterations {
			for _, reqResult := range iterResult.Requests {
				stats.TotalRequests++
				stats.TotalBytes += reqResult.Size

				latencyMs := float64(reqResult.Latency.Milliseconds())
				stats.TotalLatency += latencyMs

				if latencyMs < stats.MinLatency || stats.MinLatency == 0 {
					stats.MinLatency = latencyMs
				}
				if latencyMs > stats.MaxLatency {
					stats.MaxLatency = latencyMs
				}

				if reqResult.Error != "" || reqResult.AssertionsFailed > 0 {
					stats.FailedRequests++
				} else {
					stats.SuccessRequests++
				}
			}
		}
	}

	if stats.TotalRequests > 0 {
		stats.AvgLatency = stats.TotalLatency / float64(stats.TotalRequests)
	}

	return stats
}

// Helper functions
func parseDuration(s string) (time.Duration, error) {
	// Simple parser: "5m", "30s", "1h"
	return time.ParseDuration(s)
}

func parseLatency(s string) float64 {
	// Parse "500ms", "1s" to milliseconds
	d, _ := time.ParseDuration(s)
	return float64(d.Milliseconds())
}
