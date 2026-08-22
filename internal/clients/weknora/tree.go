package weknora

import "strings"

type folderAcc struct {
	direct int
	total  int
}

// BuildFolderTree aggregates documents by folder_path into a tree.
func BuildFolderTree(docs []Document) FolderTree {
	counts := map[string]*folderAcc{}
	rootDirect := 0
	for _, doc := range docs {
		path := strings.Trim(strings.ReplaceAll(doc.FolderPath, "\\", "/"), "/")
		if path == "" {
			rootDirect++
			continue
		}
		parts := strings.Split(path, "/")
		for i := range parts {
			p := strings.Join(parts[:i+1], "/")
			if counts[p] == nil {
				counts[p] = &folderAcc{}
			}
			counts[p].total++
			if i == len(parts)-1 {
				counts[p].direct++
			}
		}
	}
	children := map[string][]string{}
	var tops []string
	seenTop := map[string]struct{}{}
	for path := range counts {
		parent, name := splitFolder(path)
		_ = name
		if parent == "" {
			if _, ok := seenTop[path]; !ok {
				seenTop[path] = struct{}{}
				tops = append(tops, path)
			}
			continue
		}
		children[parent] = append(children[parent], path)
	}
	sortPaths(tops)
	nodes := make([]FolderNode, 0, len(tops))
	for _, p := range tops {
		nodes = append(nodes, buildNode(p, counts, children))
	}
	return FolderTree{
		RootDocumentCount:  rootDirect,
		TotalDocumentCount: len(docs),
		Folders:            nodes,
	}
}

func buildNode(path string, counts map[string]*folderAcc, children map[string][]string) FolderNode {
	_, name := splitFolder(path)
	c := counts[path]
	direct, total := 0, 0
	if c != nil {
		direct, total = c.direct, c.total
	}
	kids := append([]string(nil), children[path]...)
	sortPaths(kids)
	outKids := make([]FolderNode, 0, len(kids))
	for _, child := range kids {
		outKids = append(outKids, buildNode(child, counts, children))
	}
	return FolderNode{
		Path:          path,
		Name:          name,
		DocumentCount: direct,
		TotalCount:    total,
		Children:      outKids,
	}
}

func splitFolder(path string) (parent, name string) {
	path = strings.Trim(path, "/")
	if i := strings.LastIndex(path, "/"); i >= 0 {
		return path[:i], path[i+1:]
	}
	return "", path
}

func sortPaths(paths []string) {
	if len(paths) < 2 {
		return
	}
	for i := 0; i < len(paths); i++ {
		for j := i + 1; j < len(paths); j++ {
			if paths[j] < paths[i] {
				paths[i], paths[j] = paths[j], paths[i]
			}
		}
	}
}
