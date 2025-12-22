package scenario

import "github.com/vikasavnish/httptool/pkg/ir"

// Scenario represents a complete load testing scenario
type Scenario struct {
	Name        string
	Description string
	Tags        map[string]string
	Variables   map[string]string
	Data        map[string][]map[string]any
	Requests    map[string]*Request
	Scenarios   map[string]*ScenarioDefinition
	Setup       []string // Request names to run before scenario
	Teardown    []string // Request names to run after scenario
}

// ScenarioDefinition defines a test scenario
type ScenarioDefinition struct {
	Name      string
	Load      *LoadConfig
	Flow      *Flow
	ThinkTime *ThinkTime
}

// Request represents a named HTTP request block
type Request struct {
	Name       string
	CurlCmd    string
	Extract    map[string]string // var_name -> extraction rule
	Assert     []Assertion
	Retry      *RetryConfig
	Children   []string          // Names of child requests
	Parallel   bool              // Execute children in parallel
	Condition  string            // Conditional execution: "${var} == value"
	ForEach    *ForEachLoop
}

// LoadConfig defines load testing parameters
type LoadConfig struct {
	VUs        int
	Duration   string // "5m", "30s"
	RPS        int    // Requests per second
	Iterations int
	RampUp     string
	Stages     []*Stage
}

// Stage represents a load stage
type Stage struct {
	Duration string
	VUs      int
	RPS      int
}

// Flow represents execution flow
type Flow struct {
	Type     FlowType // sequential, parallel, conditional
	Steps    []string // Request names
	Children []*Flow
	Condition string
}

// FlowType defines flow execution type
type FlowType string

const (
	FlowSequential  FlowType = "sequential"
	FlowParallel    FlowType = "parallel"
	FlowConditional FlowType = "conditional"
	FlowNested      FlowType = "nested"
)

// Assertion represents a response assertion
type Assertion struct {
	Type     AssertType
	Field    string
	Operator string
	Value    any
}

// AssertType defines assertion type
type AssertType string

const (
	AssertStatus  AssertType = "status"
	AssertLatency AssertType = "latency"
	AssertBody    AssertType = "body"
	AssertHeader  AssertType = "header"
)

// RetryConfig defines retry behavior
type RetryConfig struct {
	MaxAttempts int
	Backoff     BackoffStrategy
	BaseDelay   string
	MaxDelay    string
}

// BackoffStrategy defines retry backoff
type BackoffStrategy string

const (
	BackoffFixed       BackoffStrategy = "fixed"
	BackoffExponential BackoffStrategy = "exponential"
	BackoffLinear      BackoffStrategy = "linear"
)

// ThinkTime defines delays between requests
type ThinkTime struct {
	Duration string
	Variance float64 // ±variance (0-1)
}

// ForEachLoop defines iteration over data
type ForEachLoop struct {
	ItemVar  string // e.g. "product"
	DataName string // e.g. "products"
}

// CompiledScenario represents a scenario compiled to IR tree
type CompiledScenario struct {
	Name      string
	Load      *LoadConfig
	Setup     []*ir.IR
	Main      []*RequestNode
	Teardown  []*ir.IR
	Variables map[string]string
}

// RequestNode represents a node in the request execution tree
type RequestNode struct {
	IR         *ir.IR
	Extract    map[string]string
	Assert     []Assertion
	Children   []*RequestNode
	Parallel   bool
	Condition  string
	ThinkTime  *ThinkTime
}
