package ir

// EvaluationContext is passed to evaluators
type EvaluationContext struct {
	IR       *IR              `json:"ir"`
	Request  *ExecutedRequest `json:"request"`
	Response *Response        `json:"response"`
	Vars     map[string]any   `json:"vars"`
}

// ExecutedRequest represents the actual HTTP request that was sent
type ExecutedRequest struct {
	Method  string            `json:"method"`
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers"`
	Body    any               `json:"body,omitempty"`
}

// Response represents the HTTP response received
type Response struct {
	Status    int               `json:"status"`
	Headers   map[string]string `json:"headers"`
	Body      any               `json:"body,omitempty"`
	LatencyMs float64           `json:"latency_ms"`
	SizeBytes int64             `json:"size_bytes,omitempty"`
	Error     string            `json:"error,omitempty"`
}

// EvaluatorDecision represents the decision output from an evaluator
type EvaluatorDecision struct {
	Decision string             `json:"decision"` // pass, retry, fail, branch
	Reason   string             `json:"reason,omitempty"`
	Mutations *Mutations        `json:"mutations,omitempty"`
	Actions  *Actions           `json:"actions,omitempty"`
	Metadata map[string]any     `json:"metadata,omitempty"`
}

// Mutations represents changes to apply for next attempt
type Mutations struct {
	Headers map[string]string `json:"headers,omitempty"`
	Query   map[string]string `json:"query,omitempty"`
	Body    any               `json:"body,omitempty"`
	Vars    map[string]any    `json:"vars,omitempty"`
}

// Actions represents execution control actions
type Actions struct {
	RetryAfterMs int                       `json:"retry_after_ms,omitempty"`
	MaxRetries   int                       `json:"max_retries,omitempty"`
	Goto         string                    `json:"goto,omitempty"`
	Extract      map[string]ExtractRule    `json:"extract,omitempty"`
}

// ExtractRule defines how to extract data from response
type ExtractRule struct {
	JSONPath string `json:"jsonpath,omitempty"`
	Regex    string `json:"regex,omitempty"`
	Default  string `json:"default,omitempty"`
}
