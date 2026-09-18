package weknora

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

const (
	ManualStatusPublish = "publish"
	ChannelGeeGooAgent  = "geegoo-agent"
)

// CreateManualKnowledge creates a manual Markdown knowledge entry.
func (c *Client) CreateManualKnowledge(ctx context.Context, title, content string) (Document, error) {
	title = strings.TrimSpace(title)
	content = strings.TrimSpace(content)
	if title == "" || content == "" {
		return Document{}, fmt.Errorf("title and content are required")
	}
	raw, err := c.postJSON(ctx, "/api/v1/knowledge-bases/"+url.PathEscape(c.kbID)+"/knowledge/manual", map[string]any{
		"title":   title,
		"content": content,
		"status":  ManualStatusPublish,
		"channel": ChannelGeeGooAgent,
	})
	if err != nil {
		return Document{}, err
	}
	doc := documentFromMap(firstObject(raw))
	if doc.ID == "" {
		return Document{}, fmt.Errorf("weknora create manual: missing id")
	}
	if doc.Title == "" {
		doc.Title = title
	}
	return doc, nil
}

// UpdateManualKnowledge updates a manual Markdown knowledge entry.
func (c *Client) UpdateManualKnowledge(ctx context.Context, id, title, content string) (Document, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return Document{}, fmt.Errorf("knowledge id is required")
	}
	body := map[string]any{
		"status": ManualStatusPublish,
	}
	if t := strings.TrimSpace(title); t != "" {
		body["title"] = t
	}
	if ctt := strings.TrimSpace(content); ctt != "" {
		body["content"] = ctt
	}
	if len(body) <= 1 {
		return Document{}, fmt.Errorf("title or content required")
	}
	raw, err := c.putJSON(ctx, "/api/v1/knowledge/manual/"+url.PathEscape(id), body)
	if err != nil {
		return Document{}, err
	}
	doc := documentFromMap(firstObject(raw))
	if doc.ID == "" {
		doc.ID = id
	}
	return doc, nil
}

// GetKnowledge loads one knowledge record by id.
func (c *Client) GetKnowledge(ctx context.Context, id string) (Document, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return Document{}, fmt.Errorf("knowledge id is required")
	}
	raw, err := c.getJSON(ctx, "/api/v1/knowledge/"+url.PathEscape(id), nil)
	if err != nil {
		return Document{}, err
	}
	return documentFromMap(firstObject(raw)), nil
}

// MoveKnowledgeToFolder assigns folder_path for knowledge entries (no reparse).
func (c *Client) MoveKnowledgeToFolder(ctx context.Context, ids []string, folderPath string) error {
	clean := make([]string, 0, len(ids))
	for _, id := range ids {
		if id = strings.TrimSpace(id); id != "" {
			clean = append(clean, id)
		}
	}
	if len(clean) == 0 {
		return fmt.Errorf("knowledge ids required")
	}
	_, err := c.postJSON(ctx, "/api/v1/knowledge/folder", map[string]any{
		"kb_id":         c.kbID,
		"knowledge_ids": clean,
		"folder_path":   strings.TrimSpace(folderPath),
	})
	return err
}

// FindKnowledgeByTitle finds a document under folderPath with exact title.
func (c *Client) FindKnowledgeByTitle(ctx context.Context, folderPath, title string) (*Document, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return nil, fmt.Errorf("title is required")
	}
	docs, err := c.ListDocuments(ctx, ListDocumentsOpts{
		FolderPath:   folderPath,
		FilterFolder: folderPath != "",
		PageSize:     200,
	})
	if err != nil {
		return nil, err
	}
	for _, doc := range docs {
		if strings.TrimSpace(doc.Title) == title || strings.TrimSpace(doc.FileName) == title {
			copy := doc
			return &copy, nil
		}
	}
	return nil, nil
}

// UpsertManualKnowledge creates or updates a manual doc and ensures folder_path.
func (c *Client) UpsertManualKnowledge(ctx context.Context, folderPath, title, content string) (Document, error) {
	folderPath = strings.TrimSpace(folderPath)
	title = strings.TrimSpace(title)
	content = strings.TrimSpace(content)
	if title == "" || content == "" {
		return Document{}, fmt.Errorf("title and content are required")
	}
	existing, err := c.FindKnowledgeByTitle(ctx, folderPath, title)
	if err != nil {
		return Document{}, err
	}
	var doc Document
	if existing != nil && existing.ID != "" {
		doc, err = c.UpdateManualKnowledge(ctx, existing.ID, title, content)
	} else {
		doc, err = c.CreateManualKnowledge(ctx, title, content)
	}
	if err != nil {
		return Document{}, err
	}
	if folderPath != "" && doc.ID != "" {
		if err := c.MoveKnowledgeToFolder(ctx, []string{doc.ID}, folderPath); err != nil {
			return Document{}, fmt.Errorf("move to folder %q: %w", folderPath, err)
		}
		doc.FolderPath = folderPath
	}
	return doc, nil
}

func (c *Client) putJSON(ctx context.Context, path string, body map[string]any) (map[string]any, error) {
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, c.apiURL+path, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	return c.doJSON(req)
}
