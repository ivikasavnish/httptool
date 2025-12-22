package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/vikasavnish/httptool/pkg/evaluator"
	"github.com/vikasavnish/httptool/pkg/executor"
	"github.com/vikasavnish/httptool/pkg/ir"
	"github.com/vikasavnish/httptool/pkg/parser"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "convert":
		handleConvert()
	case "exec", "execute":
		handleExecute()
	case "run":
		handleRun()
	case "validate":
		handleValidate()
	case "scenario":
		handleScenarioCommand()
	case "help", "--help", "-h":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

func handleScenarioCommand() {
	if len(os.Args) < 3 {
		printScenarioUsage()
		os.Exit(1)
	}

	subcommand := os.Args[2]

	switch subcommand {
	case "run":
		handleScenarioRun()
	case "validate":
		handleScenarioValidate()
	case "convert":
		handleScenarioConvert()
	default:
		fmt.Fprintf(os.Stderr, "Unknown scenario command: %s\n", subcommand)
		printScenarioUsage()
		os.Exit(1)
	}
}

func printScenarioUsage() {
	fmt.Println(`httptool scenario - Load Testing DSL

Usage:
  httptool scenario run <scenario.httpx>         Run a load testing scenario
  httptool scenario validate <scenario.httpx>    Validate scenario syntax
  httptool scenario convert <scenario.httpx>     Show compiled scenario info

Options:
  --scenario <name>   Run specific scenario (if file has multiple)
  --dry-run           Validate and show plan without executing
  --vus <N>           Override virtual users (future)
  --duration <D>      Override duration (future)

Examples:
  # Run scenario
  httptool scenario run examples/scenarios/simple-load.httpx

  # Run specific scenario
  httptool scenario run scenarios.httpx --scenario smoke_test

  # Dry run
  httptool scenario run user-journey.httpx --dry-run

  # Validate syntax
  httptool scenario validate scenario.httpx

Environment Variables:
  VERBOSE=1       Show per-VU details

Documentation:
  See examples/scenarios/ for example .httpx files
  See docs/dsl-spec.md for complete DSL reference
`)
}

func handleConvert() {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "Usage: httptool convert <curl-command>")
		os.Exit(1)
	}

	curlCmd := os.Args[2]
	p := parser.NewCurlParser()

	irSpec, err := p.Parse(curlCmd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Parse error: %v\n", err)
		os.Exit(1)
	}

	// Output as JSON
	output, err := json.MarshalIndent(irSpec, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "JSON marshal error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(string(output))
}

func handleExecute() {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "Usage: httptool exec <curl-command>")
		os.Exit(1)
	}

	curlCmd := os.Args[2]
	p := parser.NewCurlParser()

	irSpec, err := p.Parse(curlCmd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Parse error: %v\n", err)
		os.Exit(1)
	}

	executeIR(irSpec)
}

func handleRun() {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "Usage: httptool run <ir-file.json>")
		os.Exit(1)
	}

	irFile := os.Args[2]
	data, err := os.ReadFile(irFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to read file: %v\n", err)
		os.Exit(1)
	}

	var irSpec ir.IR
	if err := json.Unmarshal(data, &irSpec); err != nil {
		fmt.Fprintf(os.Stderr, "Invalid IR JSON: %v\n", err)
		os.Exit(1)
	}

	executeIR(&irSpec)
}

func handleValidate() {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "Usage: httptool validate <ir-file.json>")
		os.Exit(1)
	}

	irFile := os.Args[2]
	data, err := os.ReadFile(irFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to read file: %v\n", err)
		os.Exit(1)
	}

	var irSpec ir.IR
	if err := json.Unmarshal(data, &irSpec); err != nil {
		fmt.Fprintf(os.Stderr, "Invalid IR JSON: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("✓ IR is valid")
	fmt.Printf("  Version: %s\n", irSpec.Version)
	fmt.Printf("  Method:  %s\n", irSpec.Request.Method)
	fmt.Printf("  URL:     %s\n", irSpec.Request.URL)
}

func executeIR(irSpec *ir.IR) {
	// Create executor
	exec := executor.NewExecutor()

	// Execute request
	ctx, err := exec.Execute(irSpec)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Execution error: %v\n", err)
		os.Exit(1)
	}

	// Create evaluator manager
	timeout := 5 * time.Second
	if irSpec.Evaluation != nil && irSpec.Evaluation.TimeoutMs > 0 {
		timeout = time.Duration(irSpec.Evaluation.TimeoutMs) * time.Millisecond
	}

	evalMgr := evaluator.NewManager(timeout)

	// Run evaluator
	var decision *ir.EvaluatorDecision

	if irSpec.Evaluation != nil && irSpec.Evaluation.Evaluator != "" {
		decision, err = evalMgr.Evaluate(
			context.Background(),
			ctx,
			irSpec.Evaluation.Evaluator,
			irSpec.Evaluation.EvaluatorPath,
		)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Evaluation error: %v\n", err)
			// Fall back to default evaluator
			decision, _ = evaluator.DefaultEvaluator(ctx)
		}
	} else {
		// Use default evaluator
		decision, _ = evaluator.DefaultEvaluator(ctx)
	}

	// Output results
	printResults(ctx, decision)

	// Exit based on decision
	if decision.Decision == "fail" {
		os.Exit(1)
	}
}

func printResults(ctx *ir.EvaluationContext, decision *ir.EvaluatorDecision) {
	fmt.Printf("Request:  %s %s\n", ctx.Request.Method, ctx.Request.URL)
	fmt.Printf("Status:   %d\n", ctx.Response.Status)
	fmt.Printf("Latency:  %.2fms\n", ctx.Response.LatencyMs)
	fmt.Printf("Size:     %d bytes\n", ctx.Response.SizeBytes)

	if ctx.Response.Error != "" {
		fmt.Printf("Error:    %s\n", ctx.Response.Error)
	}

	fmt.Printf("\nDecision: %s\n", decision.Decision)
	if decision.Reason != "" {
		fmt.Printf("Reason:   %s\n", decision.Reason)
	}

	// Print headers if verbose
	if os.Getenv("VERBOSE") == "1" {
		fmt.Println("\nResponse Headers:")
		for k, v := range ctx.Response.Headers {
			fmt.Printf("  %s: %s\n", k, v)
		}
	}

	// Print body if verbose
	if os.Getenv("SHOW_BODY") == "1" {
		fmt.Println("\nResponse Body:")
		bodyJSON, err := json.MarshalIndent(ctx.Response.Body, "  ", "  ")
		if err == nil {
			fmt.Printf("  %s\n", string(bodyJSON))
		} else {
			fmt.Printf("  %v\n", ctx.Response.Body)
		}
	}
}

func printUsage() {
	fmt.Println(`httptool - HTTP Execution & Evaluation Engine

Usage:
  httptool convert <curl-command>    Convert curl command to IR JSON
  httptool exec <curl-command>       Execute curl command with evaluation
  httptool run <ir-file.json>        Execute from IR file
  httptool validate <ir-file.json>   Validate IR file
  httptool scenario <command>        Load testing scenarios (run, validate, convert)
  httptool help                      Show this help

Examples:
  # Convert curl to IR
  httptool convert 'curl https://api.example.com/users' > request.json

  # Execute curl directly
  httptool exec 'curl -X POST https://api.example.com/login -d "{\"user\":\"test\"}"'

  # Run from IR file
  httptool run request.json

  # Run load testing scenario
  httptool scenario run examples/scenarios/simple-load.httpx

Environment Variables:
  VERBOSE=1       Show response headers / per-VU details
  SHOW_BODY=1     Show response body

Documentation:
  https://github.com/vikasavnish/httptool
  See 'httptool scenario' for load testing DSL
`)
}
