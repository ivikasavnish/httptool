package orchestrator

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/vikasavnish/httptool/pkg/evaluator"
	"github.com/vikasavnish/httptool/pkg/executor"
	"github.com/vikasavnish/httptool/pkg/ir"
)

// Result represents a single execution result
type Result struct {
	IR         *ir.IR
	Context    *ir.EvaluationContext
	Decision   *ir.EvaluatorDecision
	Error      error
	StartTime  time.Time
	EndTime    time.Time
	Attempt    int
}

// Stats holds execution statistics
type Stats struct {
	Total       int
	Success     int
	Failed      int
	Retried     int
	AvgLatency  float64
	MinLatency  float64
	MaxLatency  float64
	TotalBytes  int64
}

// Orchestrator manages execution flow with retries and load testing
type Orchestrator struct {
	executor  *executor.Executor
	evaluator *evaluator.Manager
	maxRetries int
}

// NewOrchestrator creates a new orchestrator
func NewOrchestrator(maxRetries int, evalTimeout time.Duration) *Orchestrator {
	return &Orchestrator{
		executor:   executor.NewExecutor(),
		evaluator:  evaluator.NewManager(evalTimeout),
		maxRetries: maxRetries,
	}
}

// ExecuteOne executes a single IR with retry logic
func (o *Orchestrator) ExecuteOne(ctx context.Context, irSpec *ir.IR) (*Result, error) {
	result := &Result{
		IR:        irSpec,
		StartTime: time.Now(),
		Attempt:   1,
	}

	for attempt := 1; attempt <= o.maxRetries; attempt++ {
		result.Attempt = attempt

		// Set attempt in vars
		if irSpec.Evaluation == nil {
			irSpec.Evaluation = ir.DefaultEvaluation()
		}
		if irSpec.Evaluation.Vars == nil {
			irSpec.Evaluation.Vars = make(map[string]any)
		}
		irSpec.Evaluation.Vars["attempt"] = attempt

		// Execute
		execCtx, err := o.executor.Execute(irSpec)
		if err != nil {
			result.Error = err
			result.EndTime = time.Now()
			return result, err
		}

		result.Context = execCtx

		// Evaluate
		evalType := "bun"
		evalPath := ""
		if irSpec.Evaluation != nil {
			if irSpec.Evaluation.Evaluator != "" {
				evalType = irSpec.Evaluation.Evaluator
			}
			evalPath = irSpec.Evaluation.EvaluatorPath
		}

		decision, err := o.evaluator.Evaluate(ctx, execCtx, evalType, evalPath)
		if err != nil {
			// Fall back to default evaluator
			decision, _ = evaluator.DefaultEvaluator(execCtx)
		}

		result.Decision = decision

		// Handle decision
		switch decision.Decision {
		case "pass":
			result.EndTime = time.Now()
			return result, nil

		case "fail":
			result.EndTime = time.Now()
			result.Error = fmt.Errorf("evaluation failed: %s", decision.Reason)
			return result, result.Error

		case "retry":
			// Apply mutations
			if decision.Mutations != nil {
				o.applyMutations(irSpec, decision.Mutations)
			}

			// Wait before retry
			if decision.Actions != nil && decision.Actions.RetryAfterMs > 0 {
				select {
				case <-ctx.Done():
					result.EndTime = time.Now()
					return result, ctx.Err()
				case <-time.After(time.Duration(decision.Actions.RetryAfterMs) * time.Millisecond):
				}
			}

			// Check max retries override
			if decision.Actions != nil && decision.Actions.MaxRetries > 0 {
				o.maxRetries = decision.Actions.MaxRetries
			}

			continue

		case "branch":
			// TODO: Implement branching logic
			result.EndTime = time.Now()
			return result, fmt.Errorf("branching not yet implemented")
		}
	}

	result.EndTime = time.Now()
	result.Error = fmt.Errorf("max retries exceeded: %d", o.maxRetries)
	return result, result.Error
}

// ExecuteConcurrent runs multiple executions concurrently
func (o *Orchestrator) ExecuteConcurrent(ctx context.Context, irSpecs []*ir.IR, concurrency int) ([]*Result, *Stats) {
	results := make([]*Result, len(irSpecs))
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, concurrency)

	for i, spec := range irSpecs {
		wg.Add(1)
		go func(index int, irSpec *ir.IR) {
			defer wg.Done()

			// Acquire semaphore
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			result, _ := o.ExecuteOne(ctx, irSpec)
			results[index] = result
		}(i, spec)
	}

	wg.Wait()

	stats := o.calculateStats(results)
	return results, stats
}

// ExecuteLoad runs load testing with specified duration and rate
func (o *Orchestrator) ExecuteLoad(ctx context.Context, irSpec *ir.IR, duration time.Duration, rps int) ([]*Result, *Stats) {
	var results []*Result
	var mu sync.Mutex
	ticker := time.NewTicker(time.Second / time.Duration(rps))
	defer ticker.Stop()

	deadline := time.Now().Add(duration)

	for {
		select {
		case <-ctx.Done():
			stats := o.calculateStats(results)
			return results, stats

		case <-ticker.C:
			if time.Now().After(deadline) {
				stats := o.calculateStats(results)
				return results, stats
			}

			go func() {
				result, _ := o.ExecuteOne(context.Background(), irSpec)
				mu.Lock()
				results = append(results, result)
				mu.Unlock()
			}()
		}
	}
}

// Replay executes stored IR files in sequence
func (o *Orchestrator) Replay(ctx context.Context, irSpecs []*ir.IR) ([]*Result, *Stats) {
	results := make([]*Result, 0, len(irSpecs))

	for _, spec := range irSpecs {
		result, err := o.ExecuteOne(ctx, spec)
		results = append(results, result)

		if err != nil && result.Decision != nil && result.Decision.Decision == "fail" {
			// Stop on fail
			break
		}
	}

	stats := o.calculateStats(results)
	return results, stats
}

func (o *Orchestrator) applyMutations(irSpec *ir.IR, mutations *ir.Mutations) {
	if mutations.Headers != nil {
		if irSpec.Request.Headers == nil {
			irSpec.Request.Headers = make(map[string]string)
		}
		for k, v := range mutations.Headers {
			irSpec.Request.Headers[k] = v
		}
	}

	if mutations.Query != nil {
		if irSpec.Request.Query == nil {
			irSpec.Request.Query = make(map[string]any)
		}
		for k, v := range mutations.Query {
			irSpec.Request.Query[k] = v
		}
	}

	if mutations.Body != nil {
		// Update body content
		if irSpec.Request.Body != nil {
			irSpec.Request.Body.Content = mutations.Body
		}
	}

	if mutations.Vars != nil {
		if irSpec.Evaluation == nil {
			irSpec.Evaluation = ir.DefaultEvaluation()
		}
		if irSpec.Evaluation.Vars == nil {
			irSpec.Evaluation.Vars = make(map[string]any)
		}
		for k, v := range mutations.Vars {
			irSpec.Evaluation.Vars[k] = v
		}
	}
}

func (o *Orchestrator) calculateStats(results []*Result) *Stats {
	stats := &Stats{
		Total:      len(results),
		MinLatency: 999999,
	}

	var totalLatency float64

	for _, result := range results {
		if result.Context != nil {
			latency := result.Context.Response.LatencyMs
			totalLatency += latency

			if latency < stats.MinLatency {
				stats.MinLatency = latency
			}
			if latency > stats.MaxLatency {
				stats.MaxLatency = latency
			}

			stats.TotalBytes += result.Context.Response.SizeBytes
		}

		if result.Error != nil || (result.Decision != nil && result.Decision.Decision == "fail") {
			stats.Failed++
		} else {
			stats.Success++
		}

		if result.Attempt > 1 {
			stats.Retried++
		}
	}

	if stats.Total > 0 {
		stats.AvgLatency = totalLatency / float64(stats.Total)
	}

	return stats
}
