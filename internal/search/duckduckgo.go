package search

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

// Hit is one web search result.
type Hit struct {
	Title   string
	URL     string
	Snippet string
}

var (
	ddgResultLink = regexp.MustCompile(`<a[^>]*class="result__a"[^>]*href="([^"]+)"[^>]*>([\s\S]*?)</a>`)
	ddgSnippet    = regexp.MustCompile(`<a[^>]*class="result__snippet"[^>]*>([\s\S]*?)</a>`)
	tagStrip      = regexp.MustCompile(`<[^>]+>`)
)

// DuckDuckGo searches the public HTML endpoint (no API key).
func DuckDuckGo(ctx context.Context, query string, maxResults int) ([]Hit, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, fmt.Errorf("query required")
	}
	if maxResults <= 0 {
		maxResults = 5
	}
	if maxResults > 10 {
		maxResults = 10
	}

	form := url.Values{"q": {query}}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://html.duckduckgo.com/html/", strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "GeeGooAgent/1.0 (+https://github.com/ghsemail/GeeGooAgent)")

	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("duckduckgo HTTP %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return nil, err
	}
	html := string(body)
	links := ddgResultLink.FindAllStringSubmatch(html, maxResults)
	snippets := ddgSnippet.FindAllStringSubmatch(html, maxResults)
	if len(links) == 0 {
		return nil, nil
	}
	out := make([]Hit, 0, len(links))
	for i, m := range links {
		if len(m) < 3 {
			continue
		}
		hit := Hit{
			URL:   decodeDDGRedirect(m[1]),
			Title: cleanHTML(m[2]),
		}
		if i < len(snippets) && len(snippets[i]) > 1 {
			hit.Snippet = cleanHTML(snippets[i][1])
		}
		if hit.Title != "" {
			out = append(out, hit)
		}
	}
	return out, nil
}

func decodeDDGRedirect(raw string) string {
	raw = strings.TrimSpace(raw)
	if strings.Contains(raw, "uddg=") {
		if u, err := url.Parse(raw); err == nil {
			if v := u.Query().Get("uddg"); v != "" {
				return v
			}
		}
	}
	return raw
}

func cleanHTML(s string) string {
	s = tagStrip.ReplaceAllString(s, "")
	s = strings.ReplaceAll(s, "&amp;", "&")
	s = strings.ReplaceAll(s, "&quot;", "\"")
	s = strings.ReplaceAll(s, "&#39;", "'")
	return strings.TrimSpace(s)
}
