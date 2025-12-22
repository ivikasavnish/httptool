package scenario

import "time"

// ScenarioResult holds the results of a scenario execution
type ScenarioResult struct {
	Name      string
	StartTime time.Time
	EndTime   time.Time
	SetupVars map[string]any
	VUResults []*VUResult
	Stats     *Stats
}

// VUResult holds results for a single virtual user
type VUResult struct {
	VUID       int
	Iterations []*IterationResult
}

// IterationResult holds results for a single iteration
type IterationResult struct {
	IterationNum int
	StartTime    time.Time
	EndTime      time.Time
	Requests     []*RequestResult
}

// RequestResult holds results for a single request
type RequestResult struct {
	URL               string
	Method            string
	Status            int
	Latency           time.Duration
	Size              int64
	Error             string
	AssertionsFailed  int
	StartTime         time.Time
}

// Stats holds aggregated statistics
type Stats struct {
	TotalRequests   int
	SuccessRequests int
	FailedRequests  int
	TotalBytes      int64
	TotalLatency    float64
	AvgLatency      float64
	MinLatency      float64
	MaxLatency      float64
}

// PrintSummary prints a human-readable summary
// NOTE: This is not used - results are printed in scenario.go handler
