package ir

import "time"

// Version represents the IR schema version
const Version = "1.0"

// IR represents the complete Intermediate Representation
type IR struct {
	Version    string      `json:"version"`
	Metadata   *Metadata   `json:"metadata,omitempty"`
	Request    Request     `json:"request"`
	Transport  *Transport  `json:"transport,omitempty"`
	Hooks      *Hooks      `json:"hooks,omitempty"`
	Evaluation *Evaluation `json:"evaluation,omitempty"`
}

// Metadata contains request metadata
type Metadata struct {
	ID        string            `json:"id,omitempty"`
	Source    string            `json:"source,omitempty"` // curl, k6, locust, postman, har, manual, openapi
	CreatedAt *time.Time        `json:"created_at,omitempty"`
	TraceID   string            `json:"trace_id,omitempty"`
	Tags      map[string]string `json:"tags,omitempty"`
}

// Request represents the HTTP request specification
type Request struct {
	Method  string              `json:"method"`
	URL     string              `json:"url"`
	Query   map[string]any      `json:"query,omitempty"`
	Headers map[string]string   `json:"headers,omitempty"`
	Cookies map[string]string   `json:"cookies,omitempty"`
	Body    *Body               `json:"body,omitempty"`
	Auth    *Auth               `json:"auth,omitempty"`
}

// Body represents request body in various formats
type Body struct {
	Type          string `json:"type"` // json, form, text, multipart, binary
	Content       any    `json:"content,omitempty"`
	ContentBase64 string `json:"content_base64,omitempty"`
}

// Auth represents authentication configuration
type Auth struct {
	Type     string `json:"type"` // basic, bearer
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
	Token    string `json:"token,omitempty"`
}

// Transport represents transport layer configuration
type Transport struct {
	TLSVerify      bool   `json:"tls_verify"`
	FollowRedirects bool   `json:"follow_redirects"`
	MaxRedirects   int    `json:"max_redirects"`
	Proxy          string `json:"proxy,omitempty"`
	TimeoutMs      int    `json:"timeout_ms"`
	ClientCert     string `json:"client_cert,omitempty"`
	ClientKey      string `json:"client_key,omitempty"`
}

// DefaultTransport returns transport with safe defaults
func DefaultTransport() *Transport {
	return &Transport{
		TLSVerify:      true,
		FollowRedirects: true,
		MaxRedirects:   10,
		TimeoutMs:      30000,
	}
}

// Hooks represents lifecycle hooks
type Hooks struct {
	PreRequest   string `json:"pre_request,omitempty"`
	PostResponse string `json:"post_response,omitempty"`
}

// Evaluation represents evaluator configuration
type Evaluation struct {
	Evaluator     string         `json:"evaluator,omitempty"`      // bun, python, go, wasm
	EvaluatorPath string         `json:"evaluator_path,omitempty"` // path to custom evaluator
	TimeoutMs     int            `json:"timeout_ms"`
	Vars          map[string]any `json:"vars,omitempty"`
}

// DefaultEvaluation returns evaluation config with safe defaults
func DefaultEvaluation() *Evaluation {
	return &Evaluation{
		Evaluator: "bun",
		TimeoutMs: 5000,
		Vars:      make(map[string]any),
	}
}
