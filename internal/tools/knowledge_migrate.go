package tools

import (
	"context"
	"fmt"

	"github.com/ghsemail/GeeGooAgent/internal/clients/weknora"
)

var legacyKnowledgeFolderPairs = []struct {
	from string
	to   string
}{
	{LegacyStrategyMaterialsFolder, StrategyMaterialsFolder},
	{LegacyStrategyArchiveFolder, StrategyArchiveFolder},
}

// MigrateLegacyKnowledgeFolders moves documents from legacy folder paths to canonical names.
func MigrateLegacyKnowledgeFolders(ctx context.Context, client *weknora.Client) (int, error) {
	if client == nil || !client.Configured() {
		return 0, fmt.Errorf("weknora is not configured")
	}
	moved := 0
	for _, pair := range legacyKnowledgeFolderPairs {
		if pair.from == pair.to {
			continue
		}
		docs, err := client.ListDocuments(ctx, weknora.ListDocumentsOpts{
			FolderPath:   pair.from,
			FilterFolder: true,
			PageSize:     500,
		})
		if err != nil {
			return moved, fmt.Errorf("list %s: %w", pair.from, err)
		}
		if len(docs) == 0 {
			continue
		}
		ids := make([]string, 0, len(docs))
		for _, doc := range docs {
			if doc.ID != "" {
				ids = append(ids, doc.ID)
			}
		}
		if len(ids) == 0 {
			continue
		}
		if err := client.MoveKnowledgeToFolder(ctx, ids, pair.to); err != nil {
			return moved, fmt.Errorf("move %s → %s: %w", pair.from, pair.to, err)
		}
		moved += len(ids)
	}
	return moved, nil
}
