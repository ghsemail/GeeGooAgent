package signalclient

import (
	"context"
	"errors"
	"net"
	"strings"
	"time"
)

const (
	defaultFetchAttempts = 3
	defaultRetryBase     = 400 * time.Millisecond
)

// IsRetryableFetchError reports whether a lab-api fetch may succeed on a new attempt.
func IsRetryableFetchError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.Canceled) {
		return false
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	var api *APIError
	if errors.As(err, &api) {
		switch {
		case api.StatusCode == 429:
			return true
		case api.StatusCode >= 500:
			return true
		case api.StatusCode >= 400 && api.StatusCode < 500:
			return false
		}
	}
	msg := strings.ToLower(err.Error())
	if strings.Contains(msg, "unsupported frequency") || strings.Contains(msg, "invalid frequency") {
		return false
	}
	if strings.Contains(msg, "poll timeout") {
		return true
	}
	if strings.Contains(msg, "completed without result") {
		return true
	}
	var netErr net.Error
	if errors.As(err, &netErr) && (netErr.Timeout() || netErr.Temporary()) {
		return true
	}
	// Transient worker failures may succeed on a fresh job.
	if strings.Contains(msg, "lab-api: job") && strings.Contains(msg, "failed") {
		return !strings.Contains(msg, "unsupported")
	}
	return false
}

func (c *HTTPClient) fetchAttempts() int {
	if c != nil && c.FetchAttempts > 0 {
		return c.FetchAttempts
	}
	return defaultFetchAttempts
}

func (c *HTTPClient) retryBase() time.Duration {
	if c != nil && c.RetryBase > 0 {
		return c.RetryBase
	}
	return defaultRetryBase
}

func sleepFetchBackoff(ctx context.Context, attempt int, base time.Duration) error {
	if base <= 0 {
		base = defaultRetryBase
	}
	d := time.Duration(attempt+1) * base
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
