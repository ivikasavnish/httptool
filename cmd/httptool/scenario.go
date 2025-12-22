package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/vikasavnish/httptool/pkg/scenario"
)

func handleScenarioRun() {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "Usage: httptool scenario run <scenario.httpx> [--scenario name] [--vus N] [--duration D]")
		os.Exit(1)
	}

	scenarioFile := os.Args[3]

	// Read scenario file
	data, err := os.ReadFile(scenarioFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to read scenario file: %v\n", err)
		os.Exit(1)
	}

	// Parse scenario
	fmt.Printf("📋 Parsing scenario: %s\n", scenarioFile)
	parser := scenario.NewParser(string(data))
	s, err := parser.Parse()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Parse error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ Parsed successfully\n")
	fmt.Printf("  Variables: %d\n", len(s.Variables))
	fmt.Printf("  Requests: %d\n", len(s.Requests))
	fmt.Printf("  Scenarios: %d\n", len(s.Scenarios))

	// Determine which scenario to run
	scenarioName := findScenarioToRun(s, os.Args)
	if scenarioName == "" {
		fmt.Fprintln(os.Stderr, "No scenario found to run")
		os.Exit(1)
	}

	fmt.Printf("\n🚀 Preparing scenario: %s\n", scenarioName)

	// Compile scenario
	compiler := scenario.NewCompiler()
	compiled, err := compiler.Compile(s, scenarioName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Compilation error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ Compiled successfully\n")
	fmt.Printf("  Main flow: %d request(s)\n", len(compiled.Main))
	if len(compiled.Setup) > 0 {
		fmt.Printf("  Setup: %d request(s)\n", len(compiled.Setup))
	}
	if len(compiled.Teardown) > 0 {
		fmt.Printf("  Teardown: %d request(s)\n", len(compiled.Teardown))
	}

	// Display load config
	fmt.Printf("\n⚡ Load Configuration:\n")
	if compiled.Load.VUs > 0 {
		fmt.Printf("  Virtual Users: %d\n", compiled.Load.VUs)
		fmt.Printf("  Duration: %s\n", compiled.Load.Duration)
	} else if compiled.Load.RPS > 0 {
		fmt.Printf("  Requests/sec: %d\n", compiled.Load.RPS)
		fmt.Printf("  Duration: %s\n", compiled.Load.Duration)
	} else if compiled.Load.Iterations > 0 {
		fmt.Printf("  Iterations: %d\n", compiled.Load.Iterations)
		fmt.Printf("  Virtual Users: %d\n", compiled.Load.VUs)
	}

	// Check for dry-run
	if hasFlag(os.Args, "--dry-run") {
		fmt.Println("\n✓ Dry run complete (no execution)")
		return
	}

	// Execute scenario
	fmt.Printf("\n🏃 Executing scenario...\n\n")
	executor := scenario.NewExecutor()

	startTime := time.Now()
	result, err := executor.Execute(context.Background(), compiled)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Execution error: %v\n", err)
		os.Exit(1)
	}

	// Print results
	printScenarioResults(result, startTime)
}

func handleScenarioValidate() {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "Usage: httptool scenario validate <scenario.httpx>")
		os.Exit(1)
	}

	scenarioFile := os.Args[3]

	// Read scenario file
	data, err := os.ReadFile(scenarioFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to read file: %v\n", err)
		os.Exit(1)
	}

	// Parse scenario
	parser := scenario.NewParser(string(data))
	s, err := parser.Parse()
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Validation failed: %v\n", err)
		os.Exit(1)
	}

	// Validate scenarios can be compiled
	compiler := scenario.NewCompiler()
	for name := range s.Scenarios {
		_, err := compiler.Compile(s, name)
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ Scenario '%s' compilation failed: %v\n", name, err)
			os.Exit(1)
		}
	}

	fmt.Println("✓ Scenario file is valid")
	fmt.Printf("  Variables: %d\n", len(s.Variables))
	fmt.Printf("  Requests: %d\n", len(s.Requests))
	fmt.Printf("  Scenarios: %d\n", len(s.Scenarios))

	if len(s.Scenarios) > 0 {
		fmt.Println("\nScenarios:")
		for name := range s.Scenarios {
			fmt.Printf("  - %s\n", name)
		}
	}
}

func handleScenarioConvert() {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "Usage: httptool scenario convert <scenario.httpx>")
		os.Exit(1)
	}

	scenarioFile := os.Args[3]

	// Read and parse
	data, err := os.ReadFile(scenarioFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to read file: %v\n", err)
		os.Exit(1)
	}

	parser := scenario.NewParser(string(data))
	s, err := parser.Parse()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Parse error: %v\n", err)
		os.Exit(1)
	}

	// Find scenario to convert
	scenarioName := findScenarioToRun(s, os.Args)
	if scenarioName == "" {
		fmt.Fprintln(os.Stderr, "No scenario found")
		os.Exit(1)
	}

	// Compile
	compiler := scenario.NewCompiler()
	compiled, err := compiler.Compile(s, scenarioName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Compilation error: %v\n", err)
		os.Exit(1)
	}

	// Output compiled scenario info
	fmt.Printf("Scenario: %s\n", compiled.Name)
	fmt.Printf("Load: VUs=%d, Duration=%s, RPS=%d, Iterations=%d\n",
		compiled.Load.VUs, compiled.Load.Duration, compiled.Load.RPS, compiled.Load.Iterations)
	fmt.Printf("Variables: %d\n", len(compiled.Variables))
	fmt.Printf("Setup: %d requests\n", len(compiled.Setup))
	fmt.Printf("Main flow: %d top-level requests\n", len(compiled.Main))
	fmt.Printf("Teardown: %d requests\n", len(compiled.Teardown))
}

func printScenarioResults(result *scenario.ScenarioResult, startTime time.Time) {
	duration := result.EndTime.Sub(result.StartTime)

	fmt.Println("\n" + strings.Repeat("=", 70))
	fmt.Printf("  Scenario: %s\n", result.Name)
	fmt.Println(strings.Repeat("=", 70))
	fmt.Println()

	fmt.Printf("⏱  Duration: %v\n", duration)
	fmt.Printf("👥 VUs: %d\n", len(result.VUResults))
	fmt.Println()

	if result.Stats != nil {
		fmt.Println("📊 Results:")
		fmt.Printf("  Total Requests:      %d\n", result.Stats.TotalRequests)
		fmt.Printf("  ✓ Successful:        %d (%.1f%%)\n",
			result.Stats.SuccessRequests,
			float64(result.Stats.SuccessRequests)/float64(result.Stats.TotalRequests)*100)
		fmt.Printf("  ✗ Failed:            %d (%.1f%%)\n",
			result.Stats.FailedRequests,
			float64(result.Stats.FailedRequests)/float64(result.Stats.TotalRequests)*100)
		fmt.Println()

		fmt.Println("⚡ Latency:")
		fmt.Printf("  Avg:  %8.2f ms\n", result.Stats.AvgLatency)
		fmt.Printf("  Min:  %8.2f ms\n", result.Stats.MinLatency)
		fmt.Printf("  Max:  %8.2f ms\n", result.Stats.MaxLatency)
		fmt.Println()

		fmt.Printf("📦 Data Transferred: %.2f MB\n", float64(result.Stats.TotalBytes)/(1024*1024))
		fmt.Println()

		if result.Stats.TotalRequests > 0 {
			rps := float64(result.Stats.TotalRequests) / duration.Seconds()
			fmt.Printf("🚀 Throughput: %.2f req/sec\n", rps)
		}
	}

	fmt.Println(strings.Repeat("=", 70))
	fmt.Println()

	// Show per-VU summary if verbose
	if os.Getenv("VERBOSE") == "1" {
		fmt.Println("Per-VU Results:")
		for _, vu := range result.VUResults {
			fmt.Printf("  VU %d: %d iterations\n", vu.VUID, len(vu.Iterations))
		}
		fmt.Println()
	}
}

func findScenarioToRun(s *scenario.Scenario, args []string) string {
	// Check for --scenario flag
	for i, arg := range args {
		if arg == "--scenario" && i+1 < len(args) {
			return args[i+1]
		}
	}

	// If only one scenario, use it
	if len(s.Scenarios) == 1 {
		for name := range s.Scenarios {
			return name
		}
	}

	// Try to find a scenario named "default" or "main"
	for _, name := range []string{"default", "main", "test"} {
		if _, ok := s.Scenarios[name]; ok {
			return name
		}
	}

	// Return first scenario
	for name := range s.Scenarios {
		return name
	}

	return ""
}

func hasFlag(args []string, flag string) bool {
	for _, arg := range args {
		if arg == flag {
			return true
		}
	}
	return false
}
