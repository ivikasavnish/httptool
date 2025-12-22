package parser

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/vikasavnish/httptool/pkg/ir"
)

// CurlParser converts curl commands to IR
type CurlParser struct{}

// NewCurlParser creates a new curl parser
func NewCurlParser() *CurlParser {
	return &CurlParser{}
}

// Parse converts a curl command string to IR
func (p *CurlParser) Parse(curlCmd string) (*ir.IR, error) {
	tokens, err := tokenize(curlCmd)
	if err != nil {
		return nil, fmt.Errorf("tokenization failed: %w", err)
	}

	result := &ir.IR{
		Version: ir.Version,
		Metadata: &ir.Metadata{
			ID:        uuid.New().String(),
			Source:    "curl",
			CreatedAt: timePtr(time.Now()),
		},
		Request: ir.Request{
			Method:  "GET", // default
			Headers: make(map[string]string),
			Query:   make(map[string]any),
		},
		Transport:  ir.DefaultTransport(),
		Evaluation: ir.DefaultEvaluation(),
	}

	i := 0
	for i < len(tokens) {
		token := tokens[i]

		// Skip 'curl' command itself
		if token == "curl" {
			i++
			continue
		}

		// Handle flags
		if strings.HasPrefix(token, "-") {
			flag := token
			i++

			switch flag {
			case "-X", "--request":
				if i >= len(tokens) {
					return nil, fmt.Errorf("missing value for %s", flag)
				}
				result.Request.Method = strings.ToUpper(tokens[i])
				i++

			case "-H", "--header":
				if i >= len(tokens) {
					return nil, fmt.Errorf("missing value for %s", flag)
				}
				if err := parseHeader(tokens[i], &result.Request); err != nil {
					return nil, err
				}
				i++

			case "-d", "--data", "--data-raw", "--data-binary", "--data-urlencode":
				if i >= len(tokens) {
					return nil, fmt.Errorf("missing value for %s", flag)
				}
				if result.Request.Method == "GET" {
					result.Request.Method = "POST"
				}
				if err := parseData(tokens[i], flag, &result.Request); err != nil {
					return nil, err
				}
				i++

			case "-b", "--cookie":
				if i >= len(tokens) {
					return nil, fmt.Errorf("missing value for %s", flag)
				}
				parseCookies(tokens[i], &result.Request)
				i++

			case "-u", "--user":
				if i >= len(tokens) {
					return nil, fmt.Errorf("missing value for %s", flag)
				}
				parseAuth(tokens[i], &result.Request)
				i++

			case "-A", "--user-agent":
				if i >= len(tokens) {
					return nil, fmt.Errorf("missing value for %s", flag)
				}
				result.Request.Headers["User-Agent"] = tokens[i]
				i++

			case "-e", "--referer":
				if i >= len(tokens) {
					return nil, fmt.Errorf("missing value for %s", flag)
				}
				result.Request.Headers["Referer"] = tokens[i]
				i++

			case "-k", "--insecure":
				result.Transport.TLSVerify = false

			case "-L", "--location":
				result.Transport.FollowRedirects = true

			case "--max-redirs":
				if i >= len(tokens) {
					return nil, fmt.Errorf("missing value for %s", flag)
				}
				fmt.Sscanf(tokens[i], "%d", &result.Transport.MaxRedirects)
				i++

			case "-x", "--proxy":
				if i >= len(tokens) {
					return nil, fmt.Errorf("missing value for %s", flag)
				}
				result.Transport.Proxy = tokens[i]
				i++

			case "-m", "--max-time":
				if i >= len(tokens) {
					return nil, fmt.Errorf("missing value for %s", flag)
				}
				var seconds float64
				fmt.Sscanf(tokens[i], "%f", &seconds)
				result.Transport.TimeoutMs = int(seconds * 1000)
				i++

			case "--connect-timeout":
				if i >= len(tokens) {
					return nil, fmt.Errorf("missing value for %s", flag)
				}
				// Note: connect timeout separate from request timeout
				// For now, map to overall timeout
				var seconds float64
				fmt.Sscanf(tokens[i], "%f", &seconds)
				if result.Transport.TimeoutMs == 30000 { // if still default
					result.Transport.TimeoutMs = int(seconds * 1000)
				}
				i++

			case "-G", "--get":
				result.Request.Method = "GET"

			case "-I", "--head":
				result.Request.Method = "HEAD"

			case "--compressed":
				result.Request.Headers["Accept-Encoding"] = "gzip, deflate, br"

			default:
				// Ignore unknown flags for now
				// Could log warning in production
				if i < len(tokens) && !strings.HasPrefix(tokens[i], "-") {
					i++ // skip value if present
				}
			}
		} else {
			// Assume it's the URL
			if result.Request.URL == "" {
				result.Request.URL = token
			}
			i++
		}
	}

	// Validate URL is present
	if result.Request.URL == "" {
		return nil, fmt.Errorf("no URL found in curl command")
	}

	// Parse query parameters from URL
	if err := extractQueryParams(&result.Request); err != nil {
		return nil, err
	}

	return result, nil
}

func parseHeader(header string, req *ir.Request) error {
	parts := strings.SplitN(header, ":", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid header format: %s", header)
	}

	key := strings.TrimSpace(parts[0])
	value := strings.TrimSpace(parts[1])

	// Handle special headers
	switch strings.ToLower(key) {
	case "cookie":
		parseCookies(value, req)
	case "authorization":
		parseAuthorizationHeader(value, req)
	default:
		req.Headers[key] = value
	}

	return nil
}

func parseData(data string, flag string, req *ir.Request) error {
	// Try to parse as JSON first
	var jsonData any
	if err := json.Unmarshal([]byte(data), &jsonData); err == nil {
		req.Body = &ir.Body{
			Type:    "json",
			Content: jsonData,
		}
		if req.Headers["Content-Type"] == "" {
			req.Headers["Content-Type"] = "application/json"
		}
		return nil
	}

	// Check if it's URL-encoded form data
	if strings.Contains(data, "=") && !strings.Contains(data, "{") {
		formData := make(map[string]string)
		pairs := strings.Split(data, "&")
		for _, pair := range pairs {
			kv := strings.SplitN(pair, "=", 2)
			if len(kv) == 2 {
				key, _ := url.QueryUnescape(kv[0])
				val, _ := url.QueryUnescape(kv[1])
				formData[key] = val
			}
		}
		req.Body = &ir.Body{
			Type:    "form",
			Content: formData,
		}
		if req.Headers["Content-Type"] == "" {
			req.Headers["Content-Type"] = "application/x-www-form-urlencoded"
		}
		return nil
	}

	// Binary data
	if flag == "--data-binary" {
		req.Body = &ir.Body{
			Type:          "binary",
			ContentBase64: base64.StdEncoding.EncodeToString([]byte(data)),
		}
		return nil
	}

	// Default to text
	req.Body = &ir.Body{
		Type:    "text",
		Content: data,
	}
	return nil
}

func parseCookies(cookieStr string, req *ir.Request) {
	if req.Cookies == nil {
		req.Cookies = make(map[string]string)
	}

	pairs := strings.Split(cookieStr, ";")
	for _, pair := range pairs {
		kv := strings.SplitN(strings.TrimSpace(pair), "=", 2)
		if len(kv) == 2 {
			req.Cookies[kv[0]] = kv[1]
		}
	}
}

func parseAuth(userpass string, req *ir.Request) {
	parts := strings.SplitN(userpass, ":", 2)
	password := ""
	if len(parts) == 2 {
		password = parts[1]
	}

	req.Auth = &ir.Auth{
		Type:     "basic",
		Username: parts[0],
		Password: password,
	}
}

func parseAuthorizationHeader(value string, req *ir.Request) {
	if strings.HasPrefix(value, "Bearer ") {
		req.Auth = &ir.Auth{
			Type:  "bearer",
			Token: strings.TrimPrefix(value, "Bearer "),
		}
	} else if strings.HasPrefix(value, "Basic ") {
		// Could decode basic auth, but keep as-is for now
		req.Headers["Authorization"] = value
	} else {
		req.Headers["Authorization"] = value
	}
}

func extractQueryParams(req *ir.Request) error {
	parsedURL, err := url.Parse(req.URL)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	if len(parsedURL.Query()) > 0 {
		for key, values := range parsedURL.Query() {
			if len(values) == 1 {
				req.Query[key] = values[0]
			} else {
				req.Query[key] = values
			}
		}

		// Remove query from URL
		parsedURL.RawQuery = ""
		req.URL = parsedURL.String()
	}

	return nil
}

func timePtr(t time.Time) *time.Time {
	return &t
}
