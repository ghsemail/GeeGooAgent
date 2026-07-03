package workflow

import "fmt"

// ErrorKind classifies workflow step failures.
type ErrorKind string

const (
	// ErrorRecoverable: transient (network, API 5xx, timeout). Retry-worthy.
	ErrorRecoverable ErrorKind = "recoverable"
	// ErrorTerminal: contract/auth/data-shape failure. Do not retry.
	ErrorTerminal ErrorKind = "terminal"
)

// StepError is a classified workflow error.
type StepError struct {
	Kind    ErrorKind
	Tool    string
	Message string
	Cause   error
}

func (e *StepError) Error() string {
	return fmt.Sprintf("[%s] %s: %s", e.Kind, e.Tool, e.Message)
}

func (e *StepError) Unwrap() error { return e.Cause }

// NewRecoverable wraps a transient error.
func NewRecoverable(tool string, err error) *StepError {
	return &StepError{Kind: ErrorRecoverable, Tool: tool, Message: err.Error(), Cause: err}
}

// NewTerminal wraps a hard failure.
func NewTerminal(tool string, err error) *StepError {
	return &StepError{Kind: ErrorTerminal, Tool: tool, Message: err.Error(), Cause: err}
}

// ClassifyForTest exposes classifyError for tests.
func ClassifyForTest(tool, msg string) ErrorKind { return classifyError(tool, msg) }

// classifyError best-effort classifies a tool error string into a kind.
// Used when a tool returns StatusError without an explicit StepError.
func classifyError(tool, msg string) ErrorKind {
	low := msg
	if len(low) > 200 {
		low = low[:200]
	}
	low = lower(low)
	// Transient signals.
	for _, hint := range []string{"timeout", "timed out", "connection refused", "eof", "broken pipe", "502", "503", "504", "temporarily", "retry"} {
		if contains(low, hint) {
			return ErrorRecoverable
		}
	}
	// Terminal signals.
	for _, hint := range []string{"401", "403", "unauthorized", "forbidden", "invalid token", "missing mcp_token", "400", "missing required", "data contract"} {
		if contains(low, hint) {
			return ErrorTerminal
		}
	}
	// Default: recoverable for safety (supervisor will catch persistent failures).
	return ErrorRecoverable
}

func lower(s string) string {
	out := make([]byte, len(s))
	for i := range s {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		out[i] = c
	}
	return string(out)
}

func contains(s, sub string) bool {
	return indexOf(s, sub) >= 0
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
