package executor

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/vikasavnish/httptool/pkg/ir"
)

// Executor executes HTTP requests from IR (no business logic)
type Executor struct {
	client    *http.Client
	cookieJar *CookieJar
}

// NewExecutor creates a new HTTP executor
func NewExecutor() *Executor {
	return &Executor{
		client: &http.Client{
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse // Let IR control redirects
			},
		},
		cookieJar: NewCookieJar(),
	}
}

// NewExecutorWithCookieJar creates an executor with a specific cookie jar
func NewExecutorWithCookieJar(jar *CookieJar) *Executor {
	return &Executor{
		client: &http.Client{
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse // Let IR control redirects
			},
		},
		cookieJar: jar,
	}
}

// Execute runs an HTTP request and returns evaluation context
func (e *Executor) Execute(irSpec *ir.IR) (*ir.EvaluationContext, error) {
	// Configure transport
	transport := e.buildTransport(irSpec.Transport)
	e.client.Transport = transport
	e.client.Timeout = time.Duration(irSpec.Transport.TimeoutMs) * time.Millisecond

	// Handle redirects
	if irSpec.Transport.FollowRedirects {
		maxRedirects := irSpec.Transport.MaxRedirects
		e.client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			if len(via) >= maxRedirects {
				return fmt.Errorf("stopped after %d redirects", maxRedirects)
			}
			return nil
		}
	}

	// Build HTTP request
	req, err := e.buildRequest(irSpec)
	if err != nil {
		return nil, fmt.Errorf("failed to build request: %w", err)
	}

	// Add cookies from jar
	if e.cookieJar != nil {
		jarCookies, _ := e.cookieJar.GetCookies(req.URL.String())
		for _, cookie := range jarCookies {
			req.AddCookie(cookie)
		}
	}

	// Execute request
	start := time.Now()
	resp, err := e.client.Do(req)
	latencyMs := float64(time.Since(start).Microseconds()) / 1000.0

	// Build evaluation context
	ctx := &ir.EvaluationContext{
		IR: irSpec,
		Request: &ir.ExecutedRequest{
			Method:  req.Method,
			URL:     req.URL.String(),
			Headers: flattenHeaders(req.Header),
		},
		Response: &ir.Response{
			LatencyMs: latencyMs,
		},
		Vars: make(map[string]any),
	}

	// Copy evaluation vars
	if irSpec.Evaluation != nil && irSpec.Evaluation.Vars != nil {
		for k, v := range irSpec.Evaluation.Vars {
			ctx.Vars[k] = v
		}
	}

	// Add request body to context
	if irSpec.Request.Body != nil {
		ctx.Request.Body = irSpec.Request.Body.Content
	}

	// Handle execution error
	if err != nil {
		ctx.Response.Error = err.Error()
		ctx.Response.Status = 0
		return ctx, nil // Return context even on error
	}
	defer resp.Body.Close()

	// Parse response
	ctx.Response.Status = resp.StatusCode
	ctx.Response.Headers = flattenHeaders(resp.Header)

	// Extract and store cookies from response
	if e.cookieJar != nil {
		responseCookies := resp.Cookies()
		if len(responseCookies) > 0 {
			e.cookieJar.SetCookies(req.URL.String(), responseCookies)
		}
	}

	// Read body
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		ctx.Response.Error = fmt.Sprintf("failed to read response body: %v", err)
		return ctx, nil
	}

	ctx.Response.SizeBytes = int64(len(bodyBytes))

	// Try to parse as JSON, otherwise keep as string
	var jsonBody any
	if err := json.Unmarshal(bodyBytes, &jsonBody); err == nil {
		ctx.Response.Body = jsonBody
	} else {
		ctx.Response.Body = string(bodyBytes)
	}

	return ctx, nil
}

// GetCookieJar returns the executor's cookie jar
func (e *Executor) GetCookieJar() *CookieJar {
	return e.cookieJar
}

func (e *Executor) buildTransport(transport *ir.Transport) *http.Transport {
	t := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: !transport.TLSVerify,
		},
	}

	if transport.Proxy != "" {
		proxyURL, err := url.Parse(transport.Proxy)
		if err == nil {
			t.Proxy = http.ProxyURL(proxyURL)
		}
	}

	return t
}

func (e *Executor) buildRequest(irSpec *ir.IR) (*http.Request, error) {
	req := &irSpec.Request

	// Build URL with query params
	reqURL := req.URL
	if len(req.Query) > 0 {
		parsedURL, err := url.Parse(reqURL)
		if err != nil {
			return nil, fmt.Errorf("invalid URL: %w", err)
		}

		q := parsedURL.Query()
		for key, value := range req.Query {
			switch v := value.(type) {
			case string:
				q.Add(key, v)
			case []string:
				for _, val := range v {
					q.Add(key, val)
				}
			case []any:
				for _, val := range v {
					q.Add(key, fmt.Sprintf("%v", val))
				}
			default:
				q.Add(key, fmt.Sprintf("%v", v))
			}
		}
		parsedURL.RawQuery = q.Encode()
		reqURL = parsedURL.String()
	}

	// Build body
	var body io.Reader
	if req.Body != nil {
		bodyReader, contentType, err := e.buildBody(req.Body)
		if err != nil {
			return nil, err
		}
		body = bodyReader

		// Set Content-Type if not already set
		if contentType != "" && req.Headers["Content-Type"] == "" {
			if req.Headers == nil {
				req.Headers = make(map[string]string)
			}
			req.Headers["Content-Type"] = contentType
		}
	}

	// Create HTTP request
	httpReq, err := http.NewRequest(req.Method, reqURL, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	for key, value := range req.Headers {
		httpReq.Header.Set(key, value)
	}

	// Set cookies
	if len(req.Cookies) > 0 {
		for name, value := range req.Cookies {
			httpReq.AddCookie(&http.Cookie{
				Name:  name,
				Value: value,
			})
		}
	}

	// Set auth
	if req.Auth != nil {
		switch req.Auth.Type {
		case "basic":
			httpReq.SetBasicAuth(req.Auth.Username, req.Auth.Password)
		case "bearer":
			httpReq.Header.Set("Authorization", "Bearer "+req.Auth.Token)
		}
	}

	return httpReq, nil
}

func (e *Executor) buildBody(body *ir.Body) (io.Reader, string, error) {
	switch body.Type {
	case "json":
		jsonBytes, err := json.Marshal(body.Content)
		if err != nil {
			return nil, "", fmt.Errorf("failed to marshal JSON body: %w", err)
		}
		return bytes.NewReader(jsonBytes), "application/json", nil

	case "form":
		formData, ok := body.Content.(map[string]any)
		if !ok {
			return nil, "", fmt.Errorf("form body must be map[string]any")
		}
		values := url.Values{}
		for key, value := range formData {
			values.Add(key, fmt.Sprintf("%v", value))
		}
		return strings.NewReader(values.Encode()), "application/x-www-form-urlencoded", nil

	case "text":
		text, ok := body.Content.(string)
		if !ok {
			return nil, "", fmt.Errorf("text body must be string")
		}
		return strings.NewReader(text), "text/plain", nil

	case "binary":
		// Decode base64
		// For now, simplified - in production would handle base64
		return strings.NewReader(body.ContentBase64), "application/octet-stream", nil

	default:
		return nil, "", fmt.Errorf("unsupported body type: %s", body.Type)
	}
}

func flattenHeaders(headers http.Header) map[string]string {
	flat := make(map[string]string)
	for key, values := range headers {
		if len(values) > 0 {
			flat[key] = values[0] // Take first value
		}
	}
	return flat
}
