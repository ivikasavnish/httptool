package executor

import (
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"sync"
)

// CookieJar manages cookies across requests
type CookieJar struct {
	jar *cookiejar.Jar
	mu  sync.RWMutex
}

// NewCookieJar creates a new cookie jar
func NewCookieJar() *CookieJar {
	jar, _ := cookiejar.New(nil)
	return &CookieJar{
		jar: jar,
	}
}

// SetCookies sets cookies for a URL
func (c *CookieJar) SetCookies(urlStr string, cookies []*http.Cookie) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	u, err := url.Parse(urlStr)
	if err != nil {
		return err
	}

	c.jar.SetCookies(u, cookies)
	return nil
}

// GetCookies gets cookies for a URL
func (c *CookieJar) GetCookies(urlStr string) ([]*http.Cookie, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	u, err := url.Parse(urlStr)
	if err != nil {
		return nil, err
	}

	return c.jar.Cookies(u), nil
}

// ExtractCookies extracts cookies from response headers
func ExtractCookies(headers http.Header, reqURL string) map[string]string {
	cookies := make(map[string]string)

	_, err := url.Parse(reqURL)
	if err != nil {
		return cookies
	}

	// Parse Set-Cookie headers
	rawCookies := headers["Set-Cookie"]
	for _, rawCookie := range rawCookies {
		parsed := (&http.Response{Header: http.Header{"Set-Cookie": {rawCookie}}}).Cookies()
		for _, cookie := range parsed {
			cookies[cookie.Name] = cookie.Value
		}
	}

	return cookies
}

// MergeCookies merges existing cookies with new ones
func MergeCookies(existing map[string]string, new map[string]string) map[string]string {
	merged := make(map[string]string)

	// Copy existing
	for k, v := range existing {
		merged[k] = v
	}

	// Overlay new (new cookies override existing)
	for k, v := range new {
		merged[k] = v
	}

	return merged
}
