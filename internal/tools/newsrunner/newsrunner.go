package newsrunner

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// ErrUnavailable means no Python or bundled fetch_news.py script.
var ErrUnavailable = errors.New("news script runner unavailable")

// Options configures script resolution and execution.
type Options struct {
	ProjectRoot string
	Timeout     time.Duration
	// BundledOnly limits script lookup to ProjectRoot/skills/bundled (for tests).
	BundledOnly bool
}

// MarketNews runs finance-news for US/CN/HK/ALL market headlines.
func MarketNews(ctx context.Context, opts Options, market string, limit int) (string, error) {
	market = strings.ToUpper(strings.TrimSpace(market))
	if market == "" {
		market = "US"
	}
	if limit <= 0 {
		limit = 8
	}
	return runWithFallback(ctx, opts, []string{"--type", market, "--limit", fmt.Sprintf("%d", limit)}, func(c context.Context) (string, error) {
		return MarketNewsGo(c, market, limit)
	})
}

// StockNews runs finance-news for a single ticker.
func StockNews(ctx context.Context, opts Options, code string, limit int) (string, error) {
	code = strings.TrimSpace(code)
	if code == "" {
		return "", fmt.Errorf("fetch_stock_news: code required")
	}
	if limit <= 0 {
		limit = 8
	}
	return runWithFallback(ctx, opts, []string{"--stock", code, "--limit", fmt.Sprintf("%d", limit)}, func(c context.Context) (string, error) {
		return StockNewsGo(c, code, limit)
	})
}

func runScript(ctx context.Context, opts Options, args []string) (string, error) {
	script, err := resolveScript(opts.ProjectRoot, opts.BundledOnly)
	if err != nil {
		return "", err
	}
	python, err := resolvePython()
	if err != nil {
		return "", ErrUnavailable
	}
	timeout := opts.Timeout
	if timeout <= 0 {
		timeout = 90 * time.Second
	}
	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(runCtx, python, append([]string{script}, args...)...)
	if filepath.Base(python) == "py" || strings.EqualFold(filepath.Base(python), "py.exe") {
		cmd = exec.CommandContext(runCtx, python, append([]string{"-3", script}, args...)...)
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
		return "", fmt.Errorf("fetch_news.py: %s", msg)
	}
	return strings.TrimSpace(stdout.String()), nil
}

func resolveScript(projectRoot string, bundledOnly bool) (string, error) {
	candidates := []string{}
	if root := strings.TrimSpace(projectRoot); root != "" {
		candidates = append(candidates, filepath.Join(root, "skills", "bundled", "finance-news", "scripts", "fetch_news.py"))
	}
	if !bundledOnly {
		if home, err := os.UserHomeDir(); err == nil {
			candidates = append(candidates,
				filepath.Join(home, ".cursor", "skills", "finance-news", "scripts", "fetch_news.py"),
				filepath.Join(home, ".cursor", "skills", "geegoo", "skills", "finance-news", "scripts", "fetch_news.py"),
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
		if name == "py" {
			return path, nil
		}
		return path, nil
	}
	return "", ErrUnavailable
}
