package weknora

import "testing"

func TestBuildFolderTree(t *testing.T) {
	t.Parallel()
	tree := BuildFolderTree([]Document{
		{FileName: "root.txt", FolderPath: ""},
		{FileName: "macd.pdf", FolderPath: "策略"},
		{FileName: "a.md", FolderPath: "策略/子目录"},
		{FileName: "b.md", FolderPath: "策略/子目录"},
	})
	if tree.RootDocumentCount != 1 {
		t.Fatalf("root=%d", tree.RootDocumentCount)
	}
	if tree.TotalDocumentCount != 4 {
		t.Fatalf("total=%d", tree.TotalDocumentCount)
	}
	if len(tree.Folders) != 1 || tree.Folders[0].Path != "策略" {
		t.Fatalf("folders=%+v", tree.Folders)
	}
	if tree.Folders[0].DocumentCount != 1 || tree.Folders[0].TotalCount != 3 {
		t.Fatalf("策略 counts=%+v", tree.Folders[0])
	}
	if len(tree.Folders[0].Children) != 1 || tree.Folders[0].Children[0].Path != "策略/子目录" {
		t.Fatalf("children=%+v", tree.Folders[0].Children)
	}
	if tree.Folders[0].Children[0].DocumentCount != 2 {
		t.Fatalf("subdir count=%d", tree.Folders[0].Children[0].DocumentCount)
	}
}
