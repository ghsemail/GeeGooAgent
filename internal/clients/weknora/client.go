package weknora

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/config"
)

// Client talks to WeKnora API with X-API-Key (never exposed to the browser).
type Client struct {
	apiURL     string
	apiKey     string
	kbID       string
	httpClient *http.Client
}

// New creates a WeKnora client. apiURL is the host (no /api/v1 suffix).
func New(apiURL, apiKey, kbID string, httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 20 * time.Second}
	}
	return &Client{
		apiURL:     strings.TrimRight(strings.TrimSpace(apiURL), "/"),
		apiKey:     strings.TrimSpace(apiKey),
		kbID:       strings.TrimSpace(kbID),
		httpClient: httpClient,
	}
}

// NewFromResolved builds a client from resolved config.
func NewFromResolved(cfg config.ResolvedWeKnora) *Client {
	return New(cfg.APIURL, cfg.APIKey, cfg.KBID, nil)
}

// Configured reports whether the client has a URL, key, and KB id.
func (c *Client) Configured() bool {
	return c != nil && c.apiURL != "" && c.apiKey != "" && c.kbID != ""
}

// APIURL returns the WeKnora API base.
func (c *Client) APIURL() string {
	if c == nil {
		return ""
	}
	return c.apiURL
}

// KBID returns the configured knowledge-base id.
func (c *Client) KBID() string {
	if c == nil {
		return ""
	}
	return c.kbID
}

// Health checks GET /health.
func (c *Client) Health(ctx context.Context) error {
	if c == nil {
		return fmt.Errorf("weknora client is nil")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.apiURL+"/health", nil)
	if err != nil {
		return err
	}
	if c.apiKey != "" {
		req.Header.Set("X-API-Key", c.apiKey)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 1<<16))
	if resp.StatusCode >= 400 {
		return fmt.Errorf("weknora health status %d", resp.StatusCode)
	}
	return nil
}

// GetKnowledgeBase loads the configured KB.
func (c *Client) GetKnowledgeBase(ctx context.Context) (KnowledgeBase, error) {
	var kb KnowledgeBase
	raw, err := c.getJSON(ctx, "/api/v1/knowledge-bases/"+url.PathEscape(c.kbID), nil)
	if err != nil {
		return kb, err
	}
	obj := firstObject(raw)
	kb.ID = strField(obj, "id", "knowledge_base_id")
	if kb.ID == "" {
		kb.ID = c.kbID
	}
	kb.Name = strField(obj, "name")
	kb.Description = strField(obj, "description")
	kb.EmbeddingModelID = strField(obj, "embedding_model_id", "embeddingModelId")
	kb.ChatModelID = strField(obj, "summary_model_id", "chat_model_id", "llm_model_id", "summaryModelId")
	return kb, nil
}

// GetModel loads a model by id.
func (c *Client) GetModel(ctx context.Context, id string) (Model, error) {
	var m Model
	id = strings.TrimSpace(id)
	if id == "" {
		return m, fmt.Errorf("empty model id")
	}
	raw, err := c.getJSON(ctx, "/api/v1/models/"+url.PathEscape(id), nil)
	if err != nil {
		return m, err
	}
	obj := firstObject(raw)
	m.ID = strField(obj, "id")
	m.Name = strField(obj, "name", "display_name", "modelName")
	m.Type = strField(obj, "type")
	return m, nil
}

// ListDocuments lists knowledge files. When FilterFolder is true, folder_path is sent
// (empty string means KB root).
func (c *Client) ListDocuments(ctx context.Context, opts ListDocumentsOpts) ([]Document, error) {
	page := opts.Page
	if page <= 0 {
		page = 1
	}
	pageSize := opts.PageSize
	if pageSize <= 0 {
		pageSize = 100
	}
	var all []Document
	for {
		q := url.Values{}
		q.Set("page", strconv.Itoa(page))
		q.Set("page_size", strconv.Itoa(pageSize))
		if opts.FilterFolder {
			q.Set("folder_path", opts.FolderPath)
		}
		raw, err := c.getJSON(ctx, "/api/v1/knowledge-bases/"+url.PathEscape(c.kbID)+"/knowledge", q)
		if err != nil {
			return nil, err
		}
		items := firstArray(raw)
		for _, item := range items {
			all = append(all, documentFromMap(item))
		}
		total := intField(raw, "total")
		if len(items) < pageSize || (total > 0 && len(all) >= total) || len(items) == 0 {
			break
		}
		page++
		if page > 50 {
			break
		}
	}
	return all, nil
}

// Folders returns the WeKnora folder tree, falling back to aggregating documents.
func (c *Client) Folders(ctx context.Context) (FolderTree, error) {
	raw, err := c.getJSON(ctx, "/api/v1/knowledge-bases/"+url.PathEscape(c.kbID)+"/knowledge/folders", nil)
	if err == nil {
		if tree, ok := folderTreeFromMap(firstObject(raw)); ok {
			return tree, nil
		}
	}
	docs, listErr := c.ListDocuments(ctx, ListDocumentsOpts{})
	if listErr != nil {
		if err != nil {
			return FolderTree{}, err
		}
		return FolderTree{}, listErr
	}
	return BuildFolderTree(docs), nil
}

// Search runs hybrid knowledge search (no LLM summary).
func (c *Client) Search(ctx context.Context, query, folderPath string, topK int) ([]SearchHit, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, fmt.Errorf("query is required")
	}
	if topK <= 0 {
		topK = 8
	}
	body := map[string]any{
		"query":             query,
		"knowledge_base_id": c.kbID,
		"top_k":             topK,
	}
	folderPath = strings.TrimSpace(folderPath)
	if folderPath != "" {
		docs, err := c.ListDocuments(ctx, ListDocumentsOpts{FolderPath: folderPath, FilterFolder: true, PageSize: 100})
		if err != nil {
			return nil, err
		}
		ids := make([]string, 0, len(docs))
		for _, d := range docs {
			if d.ID != "" {
				ids = append(ids, d.ID)
			}
		}
		if len(ids) == 0 {
			return nil, nil
		}
		body["knowledge_ids"] = ids
	}
	raw, err := c.postJSON(ctx, "/api/v1/knowledge-search", body)
	if err != nil {
		return nil, err
	}
	items := firstArray(raw)
	out := make([]SearchHit, 0, len(items))
	for _, item := range items {
		hit := SearchHit{
			Content:  strField(item, "content"),
			Filename: strField(item, "knowledge_filename", "file_name", "filename"),
			Title:    strField(item, "knowledge_title", "title"),
			Folder:   strField(item, "folder_path"),
			Score:    floatField(item, "score"),
		}
		if meta, ok := item["metadata"].(map[string]any); ok && hit.Folder == "" {
			hit.Folder = strField(meta, "folder_path")
		}
		if hit.Content != "" || hit.Filename != "" {
			out = append(out, hit)
		}
	}
	return out, nil
}

func (c *Client) getJSON(ctx context.Context, path string, query url.Values) (map[string]any, error) {
	u := c.apiURL + path
	if query != nil && len(query) > 0 {
		u += "?" + query.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	return c.doJSON(req)
}

func (c *Client) postJSON(ctx context.Context, path string, body map[string]any) (map[string]any, error) {
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.apiURL+path, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	return c.doJSON(req)
}

func (c *Client) doJSON(req *http.Request) (map[string]any, error) {
	if !c.Configured() {
		return nil, fmt.Errorf("weknora is not configured")
	}
	req.Header.Set("X-API-Key", c.apiKey)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("weknora %s %s: status %d: %s", req.Method, req.URL.Path, resp.StatusCode, shorten(string(raw), 240))
	}
	if len(bytes.TrimSpace(raw)) == 0 {
		return map[string]any{}, nil
	}
	var envelope map[string]any
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return nil, fmt.Errorf("decode weknora json: %w", err)
	}
	return envelope, nil
}

func firstObject(raw map[string]any) map[string]any {
	if raw == nil {
		return map[string]any{}
	}
	if data, ok := raw["data"].(map[string]any); ok {
		return data
	}
	return raw
}

func firstArray(raw map[string]any) []map[string]any {
	if raw == nil {
		return nil
	}
	switch data := raw["data"].(type) {
	case []any:
		return mapsFromAny(data)
	case map[string]any:
		if items, ok := data["items"].([]any); ok {
			return mapsFromAny(items)
		}
		if items, ok := data["data"].([]any); ok {
			return mapsFromAny(items)
		}
	}
	if items, ok := raw["items"].([]any); ok {
		return mapsFromAny(items)
	}
	return nil
}

func mapsFromAny(in []any) []map[string]any {
	out := make([]map[string]any, 0, len(in))
	for _, item := range in {
		if m, ok := item.(map[string]any); ok {
			out = append(out, m)
		}
	}
	return out
}

func documentFromMap(m map[string]any) Document {
	return Document{
		ID:          strField(m, "id"),
		FileName:    strField(m, "file_name", "filename", "title"),
		Title:       strField(m, "title"),
		FolderPath:  strField(m, "folder_path"),
		FileSize:    int64(intField(m, "file_size")),
		ParseStatus: strField(m, "parse_status"),
		UpdatedAt:   strField(m, "updated_at"),
	}
}

func folderTreeFromMap(m map[string]any) (FolderTree, bool) {
	if m == nil {
		return FolderTree{}, false
	}
	foldersRaw, ok := m["folders"]
	if !ok {
		return FolderTree{}, false
	}
	tree := FolderTree{
		RootDocumentCount:  intField(m, "root_document_count"),
		TotalDocumentCount: intField(m, "total_document_count"),
	}
	switch v := foldersRaw.(type) {
	case []any:
		tree.Folders = folderNodesFromAny(v)
	}
	return tree, true
}

func folderNodesFromAny(in []any) []FolderNode {
	out := make([]FolderNode, 0, len(in))
	for _, item := range in {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		node := FolderNode{
			Path:          strField(m, "path"),
			Name:          strField(m, "name"),
			DocumentCount: intField(m, "document_count"),
			TotalCount:    intField(m, "total_count"),
		}
		if kids, ok := m["children"].([]any); ok {
			node.Children = folderNodesFromAny(kids)
		}
		out = append(out, node)
	}
	return out
}

func strField(m map[string]any, keys ...string) string {
	for _, k := range keys {
		switch v := m[k].(type) {
		case string:
			if strings.TrimSpace(v) != "" {
				return v
			}
		}
	}
	return ""
}

func intField(m map[string]any, key string) int {
	switch v := m[key].(type) {
	case float64:
		return int(v)
	case int:
		return v
	case json.Number:
		n, _ := v.Int64()
		return int(n)
	default:
		return 0
	}
}

func floatField(m map[string]any, key string) float64 {
	switch v := m[key].(type) {
	case float64:
		return v
	case int:
		return float64(v)
	case json.Number:
		n, _ := v.Float64()
		return n
	default:
		return 0
	}
}

func shorten(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	return s[:n-3] + "..."
}
