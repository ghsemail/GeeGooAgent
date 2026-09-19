package tools

import "github.com/ghsemail/GeeGooAgent/internal/clients/weknora"

// DecorateKnowledgeTree adds display_path / display_name for legacy folder aliases.
func DecorateKnowledgeTree(tree weknora.FolderTree) map[string]any {
	return map[string]any{
		"root_document_count":  tree.RootDocumentCount,
		"total_document_count": tree.TotalDocumentCount,
		"folders":              decorateFolderNodes(tree.Folders),
	}
}

func decorateFolderNodes(nodes []weknora.FolderNode) []map[string]any {
	out := make([]map[string]any, 0, len(nodes))
	for _, node := range nodes {
		display := KnowledgeFolderDisplayName(node.Path)
		displayName := KnowledgeFolderDisplayName(node.Name)
		if displayName == node.Name {
			_, leaf := splitFolderLeaf(display)
			displayName = leaf
			if displayName == "" {
				displayName = display
			}
		}
		out = append(out, map[string]any{
			"path":           node.Path,
			"name":           node.Name,
			"display_path":   display,
			"display_name":   displayName,
			"document_count": node.DocumentCount,
			"total_count":    node.TotalCount,
			"children":       decorateFolderNodes(node.Children),
		})
	}
	return out
}

func splitFolderLeaf(path string) (parent, leaf string) {
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' {
			return path[:i], path[i+1:]
		}
	}
	return "", path
}
