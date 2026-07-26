package searchrunner

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/search"
)

// ErrUnavailable means no Python or bundled web_search.py script.
var ErrUnavailable = errors.New("web search script runner unavailable")

// Options configures script resolution and execution.
type Options struct {
	ProjectRoot string
	Timeout     time.Duration
	BundledOnly bool
}

type scriptPayload struct {
	Query   string `json:"query"`
	Count   int    `json:"count"`
	Results []struct {
		Title   string `json:"title"`
		URL     string `json:"url"`
		Snippet string `json:"snippet"`
	} `json:"results"`
}

// WebSearch runs duckduckgo-search skill script, falling back to Go DuckDuckGo.
func WebSearch(ctx context.Context, opts Options, query string, maxResults int) ([]search.Hit, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, fmt.Errorf("query required")
	}
	if maxResults <= 0 {
		maxResults = 5
	}
	hits, err := runScript(ctx, opts, query, maxResults)
	if err == nil {
		return hits, nil
	}
	if !errors.Is(err, ErrUnavailable) {
		return nil, err
	}
	return search.DuckDuckGo(ctx, query, maxResults)
}

func runScript(ctx context.Context, opts Options, query string, maxResults int) ([]search.Hit, error) {
	script, err := resolveScript(opts.ProjectRoot, opts.BundledOnly)
	if err != nil {
		return nil, err
	}
	python, err := resolvePython()
	if err != nil {
		return nil, ErrUnavailable
	}
	timeout := opts.Timeout
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	args := []string{script, "--query", query, "--limit", fmt.Sprintf("%d", maxResults), "--json"}
	var cmd *exec.Cmd
	if filepath.Base(python) == "py" || strings.EqualFold(filepath.Base(python), "py.exe") {
		cmd = exec.CommandContext(runCtx, python, append([]string{"-3"}, args...)...)
	} else {
		cmd = exec.CommandContext(runCtx, python, args...)
	}
	cmd.Dir = filepath.Dir(script)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return nil, fmt.Errorf("web_search.py: %s", msg)
	}
	var payload scriptPayload
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		return nil, fmt.Errorf("web_search.py: invalid JSON: %w", err)
	}
	out := make([]search.Hit, 0, len(payload.Results))
	for _, r := range payload.Results {
		title := strings.TrimSpace(r.Title)
		if title == "" {
			continue
		}
		out = append(out, search.Hit{
			Title:   title,
			URL:     strings.TrimSpace(r.URL),
			Snippet: strings.TrimSpace(r.Snippet),
		})
	}
	return out, nil
}

func resolveScript(projectRoot string, bundledOnly bool) (string, error) {
	candidates := []string{}
	if root := strings.TrimSpace(projectRoot); root != "" {
		candidates = append(candidates, filepath.Join(root, "skills", "bundled", "duckduckgo-search", "scripts", "web_search.py"))
	}
	if !bundledOnly {
		if home, err := os.UserHomeDir(); err == nil {
			candidates = append(candidates,
				filepath.Join(home, ".cursor", "skills", "duckduckgo-search", "scripts", "web_search.py"),
				filepath.Join(home, ".cursor", "skills", "free-stock-global-quotes-news", "scripts", "web_search.py"),
			)
		}
	}
	for _, path := range candidates {
		if st, err := os.Stat(path); err == nil && !st.IsDir() {
			return path, nil
		}
	}
	return "", ErrUnavailable
}

func resolvePython() (string, error) {
	if v := strings.TrimSpace(os.Getenv("GEEGOO_PYTHON")); v != "" {
		if _, err := exec.LookPath(v); err == nil {
			return v, nil
		}
	}
	names := []string{"python3", "python"}
	if runtime.GOOS == "windows" {
		names = append([]string{"py"}, names...)
	}
	for _, name := range names {
		path, err := exec.LookPath(name)
		if err != nil {
			continue
		}
		return path, nil
	}
	return "", ErrUnavailable
}
