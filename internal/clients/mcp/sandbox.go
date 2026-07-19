package mcp

import (
	"fmt"
	"strings"
)

// ShouldAnalyzeFallback reports whether analyze-api failure should retry via mcp-api.
// JSON business errors (e.g. code 101) must not fall back — they would surface as misleading HTTP 502.
func ShouldAnalyzeFallback(err error) bool {
	if err == nil {
		return false
	}
	ce, ok := err.(*ClientError)
	if !ok {
		return true
	}
	if ce.HTTPStatus >= 500 {
		return true
	}
	if ce.APICode != nil {
		return false
	}
	msg := ce.Message
	return strings.Contains(msg, "transport error") ||
		strings.Contains(msg, "request failed after retries") ||
		strings.Contains(msg, "invalid JSON")
}

// ClientError is an HTTP or GeeGoo API business error.
type ClientError struct {
	Message    string
	APICode    *int
	HTTPStatus int
}

func (e *ClientError) Error() string { return e.Message }

func newClientError(msg string, code *int, httpStatus int) *ClientError {
	return &ClientError{Message: msg, APICode: code, HTTPStatus: httpStatus}
}

// SandboxError indicates a network policy violation.
type SandboxError struct {
	Message string
}

func (e *SandboxError) Error() string { return e.Message }

// NetworkPolicy enforces HTTP host allowlist.
type NetworkPolicy struct {
	allowed map[string]struct{}
}

func NewNetworkPolicy(hosts []string) *NetworkPolicy {
	m := make(map[string]struct{}, len(hosts))
	for _, h := range hosts {
		m[normalizeHost(h)] = struct{}{}
	}
	return &NetworkPolicy{allowed: m}
}

func (p *NetworkPolicy) AssertHostAllowed(host string) error {
	h := normalizeHost(host)
	if h == "" {
		return &SandboxError{Message: "invalid url: empty host"}
	}
	if _, ok := p.allowed[h]; !ok {
		return &SandboxError{Message: fmt.Sprintf("host not in allowlist: %s", h)}
	}
	return nil
}

func normalizeHost(host string) string {
	// lower-case ASCII hostnames
	b := []byte(host)
	for i, c := range b {
		if c >= 'A' && c <= 'Z' {
			b[i] = c + ('a' - 'A')
		}
	}
	return string(b)
}
