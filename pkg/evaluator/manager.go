package evaluator

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"time"

	"github.com/vikasavnish/httptool/pkg/ir"
)

// Manager handles evaluator execution with safety controls
type Manager struct {
	timeout time.Duration
}

// NewManager creates a new evaluator manager
func NewManager(timeout time.Duration) *Manager {
	return &Manager{
		timeout: timeout,
	}
}

// Evaluate runs an evaluator and returns its decision
func (m *Manager) Evaluate(ctx context.Context, evalCtx *ir.EvaluationContext, evaluatorType string, evaluatorPath string) (*ir.EvaluatorDecision, error) {
	// Serialize context to JSON
	contextJSON, err := json.Marshal(evalCtx)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal evaluation context: %w", err)
	}

	// Select evaluator
	var cmd *exec.Cmd
	switch evaluatorType {
	case "bun":
		cmd = m.createBunCommand(evaluatorPath, contextJSON)
	case "python":
		cmd = m.createPythonCommand(evaluatorPath, contextJSON)
	case "go":
		cmd = m.createGoCommand(evaluatorPath, contextJSON)
	default:
		return nil, fmt.Errorf("unsupported evaluator type: %s", evaluatorType)
	}

	// Execute with timeout
	execCtx, cancel := context.WithTimeout(ctx, m.timeout)
	defer cancel()

	cmd = exec.CommandContext(execCtx, cmd.Args[0], cmd.Args[1:]...)
	cmd.Stdin = bytes.NewReader(contextJSON)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Run evaluator
	err = cmd.Run()
	if err != nil {
		if execCtx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("evaluator timeout after %v", m.timeout)
		}
		return nil, fmt.Errorf("evaluator failed: %w (stderr: %s)", err, stderr.String())
	}

	// Parse decision
	var decision ir.EvaluatorDecision
	if err := json.Unmarshal(stdout.Bytes(), &decision); err != nil {
		return nil, fmt.Errorf("failed to parse evaluator output: %w (output: %s)", err, stdout.String())
	}

	// Validate decision
	if err := m.validateDecision(&decision); err != nil {
		return nil, fmt.Errorf("invalid evaluator decision: %w", err)
	}

	return &decision, nil
}

func (m *Manager) createBunCommand(evaluatorPath string, contextJSON []byte) *exec.Cmd {
	if evaluatorPath == "" {
		evaluatorPath = "evaluator.js" // default
	}
	return exec.Command("bun", "run", evaluatorPath)
}

func (m *Manager) createPythonCommand(evaluatorPath string, contextJSON []byte) *exec.Cmd {
	if evaluatorPath == "" {
		evaluatorPath = "evaluator.py" // default
	}
	// Try mojo first, fall back to python3
	if _, err := exec.LookPath("mojo"); err == nil {
		return exec.Command("mojo", evaluatorPath)
	}
	return exec.Command("python3", evaluatorPath)
}

func (m *Manager) createGoCommand(evaluatorPath string, contextJSON []byte) *exec.Cmd {
	if evaluatorPath == "" {
		evaluatorPath = "./evaluator" // default binary
	}
	return exec.Command(evaluatorPath)
}

func (m *Manager) validateDecision(decision *ir.EvaluatorDecision) error {
	// Validate decision type
	validDecisions := map[string]bool{
		"pass":   true,
		"retry":  true,
		"fail":   true,
		"branch": true,
	}

	if !validDecisions[decision.Decision] {
		return fmt.Errorf("invalid decision type: %s", decision.Decision)
	}

	// Validate branch decision has goto
	if decision.Decision == "branch" && (decision.Actions == nil || decision.Actions.Goto == "") {
		return fmt.Errorf("branch decision requires 'goto' action")
	}

	// Validate retry has delay
	if decision.Decision == "retry" && decision.Actions != nil {
		if decision.Actions.RetryAfterMs < 0 {
			return fmt.Errorf("retry_after_ms cannot be negative")
		}
	}

	return nil
}

// DefaultEvaluator provides a simple pass-through evaluator
func DefaultEvaluator(ctx *ir.EvaluationContext) (*ir.EvaluatorDecision, error) {
	decision := &ir.EvaluatorDecision{
		Decision: "pass",
		Reason:   "default evaluator",
	}

	// Simple logic: fail on 4xx/5xx
	if ctx.Response.Status >= 400 {
		decision.Decision = "fail"
		decision.Reason = fmt.Sprintf("HTTP %d error", ctx.Response.Status)
	}

	return decision, nil
}
