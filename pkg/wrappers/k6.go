package wrappers

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/vikasavnish/httptool/pkg/ir"
)

// K6Wrapper converts k6 test scripts to IR
type K6Wrapper struct{}

// K6Request represents a k6 HTTP request
type K6Request struct {
	Method  string            `json:"method"`
	URL     string            `json:"url"`
	Body    interface{}       `json:"body,omitempty"`
	Params  *K6Params         `json:"params,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
}

// K6Params represents k6 request parameters
type K6Params struct {
	Headers       map[string]string `json:"headers,omitempty"`
	Cookies       map[string]string `json:"cookies,omitempty"`
	Tags          map[string]string `json:"tags,omitempty"`
	Auth          string            `json:"auth,omitempty"`
	Timeout       string            `json:"timeout,omitempty"`
	ResponseType  string            `json:"responseType,omitempty"`
	Redirects     int               `json:"redirects,omitempty"`
}

// NewK6Wrapper creates a new k6 wrapper
func NewK6Wrapper() *K6Wrapper {
	return &K6Wrapper{}
}

// Convert transforms a k6 request to IR
func (w *K6Wrapper) Convert(k6Req *K6Request) (*ir.IR, error) {
	result := &ir.IR{
		Version: ir.Version,
		Metadata: &ir.Metadata{
			ID:        uuid.New().String(),
			Source:    "k6",
			CreatedAt: timePtr(time.Now()),
		},
		Request: ir.Request{
			Method:  k6Req.Method,
			URL:     k6Req.URL,
			Headers: make(map[string]string),
		},
		Transport:  ir.DefaultTransport(),
		Evaluation: ir.DefaultEvaluation(),
	}

	// Merge headers
	if k6Req.Headers != nil {
		for k, v := range k6Req.Headers {
			result.Request.Headers[k] = v
		}
	}

	if k6Req.Params != nil {
		// Add params headers
		if k6Req.Params.Headers != nil {
			for k, v := range k6Req.Params.Headers {
				result.Request.Headers[k] = v
			}
		}

		// Add cookies
		if k6Req.Params.Cookies != nil {
			result.Request.Cookies = k6Req.Params.Cookies
		}

		// Add tags
		if k6Req.Params.Tags != nil {
			result.Metadata.Tags = k6Req.Params.Tags
		}

		// Parse timeout
		if k6Req.Params.Timeout != "" {
			// Simple parsing: "30s" -> 30000ms
			var seconds float64
			fmt.Sscanf(k6Req.Params.Timeout, "%fs", &seconds)
			if seconds > 0 {
				result.Transport.TimeoutMs = int(seconds * 1000)
			}
		}

		// Set redirects
		if k6Req.Params.Redirects > 0 {
			result.Transport.MaxRedirects = k6Req.Params.Redirects
		}
	}

	// Parse body
	if k6Req.Body != nil {
		body, err := w.convertBody(k6Req.Body)
		if err != nil {
			return nil, err
		}
		result.Request.Body = body
	}

	return result, nil
}

func (w *K6Wrapper) convertBody(body interface{}) (*ir.Body, error) {
	// Check if it's a JSON object
	if jsonObj, ok := body.(map[string]interface{}); ok {
		return &ir.Body{
			Type:    "json",
			Content: jsonObj,
		}, nil
	}

	// Check if it's a string
	if str, ok := body.(string); ok {
		// Try to parse as JSON
		var jsonData interface{}
		if err := json.Unmarshal([]byte(str), &jsonData); err == nil {
			return &ir.Body{
				Type:    "json",
				Content: jsonData,
			}, nil
		}

		// Default to text
		return &ir.Body{
			Type:    "text",
			Content: str,
		}, nil
	}

	return nil, fmt.Errorf("unsupported body type: %T", body)
}

// ConvertFromJSON parses k6 request from JSON string
func (w *K6Wrapper) ConvertFromJSON(jsonStr string) (*ir.IR, error) {
	var k6Req K6Request
	if err := json.Unmarshal([]byte(jsonStr), &k6Req); err != nil {
		return nil, fmt.Errorf("failed to parse k6 JSON: %w", err)
	}

	return w.Convert(&k6Req)
}

func timePtr(t time.Time) *time.Time {
	return &t
}
