package llm

import (
	"errors"
	"strconv"
)

// HTTPError is returned when an OpenAI-compatible API responds with HTTP >= 400.
type HTTPError struct {
	StatusCode int
	Body       string
}

func (e *HTTPError) Error() string {
	if e == nil {
		return "LLM HTTP error"
	}
	return "LLM HTTP " + strconv.Itoa(e.StatusCode) + ": " + truncate(e.Body, 200)
}

// FailoverEligible reports whether the gateway should try the next fallback provider.
func FailoverEligible(err error) bool {
	var he *HTTPError
	if !errors.As(err, &he) || he == nil {
		return false
	}
	switch he.StatusCode {
	case 401, 403, 429:
		return true
	default:
		return he.StatusCode >= 500
	}
}
