package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Client talks to GeeGooBot mcp-api (:3120) with Bearer + mcp_token body.
type Client struct {
	baseURL    string
	apiKey     string
	network    *NetworkPolicy
	httpClient *http.Client
	maxRetries int
	retryWait  time.Duration
	sleep      func(time.Duration)
}

// Options configures MCP client behavior.
type Options struct {
	Timeout           time.Duration
	MaxRetries        int
	RetryWait         time.Duration
	HTTPClient        *http.Client
	Sleep             func(time.Duration)
	AllowedHosts      []string
	NetworkPolicy     *NetworkPolicy
}

// NewClient creates an MCP HTTP client.
func NewClient(baseURL, apiKey string, opts Options) *Client {
	if opts.Timeout == 0 {
		opts.Timeout = 60 * time.Second
	}
	if opts.MaxRetries == 0 {
		opts.MaxRetries = 3
	}
	if opts.RetryWait == 0 {
		opts.RetryWait = 5 * time.Second
	}
	if opts.Sleep == nil {
		opts.Sleep = time.Sleep
	}
	policy := opts.NetworkPolicy
	if policy == nil {
		policy = NewNetworkPolicy(opts.AllowedHosts)
	}
	httpClient := opts.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: opts.Timeout}
	}
	return &Client{
		baseURL:    strings.TrimRight(baseURL, "/"),
		apiKey:     apiKey,
		network:    policy,
		httpClient: httpClient,
		maxRetries: opts.MaxRetries,
		retryWait:  opts.RetryWait,
		sleep:      opts.Sleep,
	}
}

// Post sends a standard MCP request expecting {"code":100,...}.
func (c *Client) Post(ctx context.Context, path string, body map[string]any) (map[string]any, error) {
	raw, err := c.doJSON(ctx, path, body)
	if err != nil {
		return nil, err
	}
	var envelope map[string]any
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return nil, newClientError(fmt.Sprintf("invalid JSON for %s", path), nil, 0)
	}
	if code, ok := envelope["code"].(float64); ok {
		if int(code) != 100 {
			msg, _ := envelope["message"].(string)
			if msg == "" {
				msg = fmt.Sprintf("api error code %d", int(code))
			}
			ic := int(code)
			return nil, newClientError(msg, &ic, 0)
		}
	}
	return envelope, nil
}

// PostDirect handles MCP Common endpoints (bare object/array or code envelope).
func (c *Client) PostDirect(ctx context.Context, path string, body map[string]any) (any, error) {
	raw, err := c.doJSON(ctx, path, body)
	if err != nil {
		return nil, err
	}
	var data any
	if err := json.Unmarshal(raw, &data); err != nil {
		return nil, newClientError(fmt.Sprintf("invalid JSON for %s", path), nil, 0)
	}
	if m, ok := data.(map[string]any); ok {
		if code, has := m["code"].(float64); has {
			if int(code) != 100 {
				msg, _ := m["message"].(string)
				if msg == "" {
					msg = fmt.Sprintf("api error code %d", int(code))
				}
				ic := int(code)
				return nil, newClientError(msg, &ic, 0)
			}
		}
		if errVal, has := m["error"]; has && m["code"] == nil {
			return nil, newClientError(fmt.Sprintf("%v", errVal), nil, 401)
		}
	}
	return data, nil
}

func (c *Client) doJSON(ctx context.Context, path string, body map[string]any) ([]byte, error) {
	fullURL := c.baseURL + path
	host, err := hostFromURL(fullURL)
	if err != nil {
		return nil, err
	}
	if err := c.network.AssertHostAllowed(host); err != nil {
		return nil, err
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	var lastErr error
	for attempt := 0; attempt < c.maxRetries; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, fullURL, bytes.NewReader(payload))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
		req.Header.Set("Content-Type", "application/json")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = newClientError(fmt.Sprintf("transport error: %v", err), nil, 0)
			if attempt < c.maxRetries-1 {
				c.sleep(c.retryWait)
			}
			continue
		}

		raw, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr != nil {
			lastErr = readErr
			if attempt < c.maxRetries-1 {
				c.sleep(c.retryWait)
			}
			continue
		}

		if resp.StatusCode >= 500 {
			lastErr = newClientError(fmt.Sprintf("server error %d for %s", resp.StatusCode, path), nil, resp.StatusCode)
			if attempt < c.maxRetries-1 {
				c.sleep(c.retryWait)
			}
			continue
		}

		if resp.StatusCode >= 400 {
			return nil, clientErrorFromHTTP(path, resp.StatusCode, raw)
		}

		return raw, nil
	}
	return nil, newClientError(fmt.Sprintf("request failed after retries: %v", lastErr), nil, 0)
}

func clientErrorFromHTTP(path string, status int, raw []byte) error {
	var data map[string]any
	if json.Unmarshal(raw, &data) == nil {
		if errMsg, ok := data["error"].(string); ok && strings.TrimSpace(errMsg) != "" {
			return newClientError(errMsg, nil, status)
		}
		msg, _ := data["message"].(string)
		var codePtr *int
		if code, ok := data["code"].(float64); ok {
			ic := int(code)
			codePtr = &ic
		}
		if strings.TrimSpace(msg) != "" {
			return newClientError(msg, codePtr, status)
		}
		if codePtr != nil {
			return newClientError(fmt.Sprintf("api error code %d", *codePtr), codePtr, status)
		}
	}
	snippet := strings.TrimSpace(string(raw))
	if snippet == "" {
		snippet = fmt.Sprintf("HTTP %d", status)
	}
	return newClientError(fmt.Sprintf("HTTP %d for %s: %s", status, path, truncate(snippet, 120)), nil, status)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

func hostFromURL(raw string) (string, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return "", &SandboxError{Message: fmt.Sprintf("invalid url: %s", raw)}
	}
	if u.Hostname() == "" {
		return "", &SandboxError{Message: fmt.Sprintf("invalid url: %s", raw)}
	}
	return u.Hostname(), nil
}
