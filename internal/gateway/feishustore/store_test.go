package feishustore

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSaveLoadList(t *testing.T) {
	dir := t.TempDir()
	c := &Creds{
		UserID: "u1", MCPToken: "mcp_x", AppID: "cli_1", AppSecret: "sec",
		Domain: "feishu", AllowedUsers: []string{"ou_a"}, Enabled: true,
	}
	if err := Save(dir, c); err != nil {
		t.Fatal(err)
	}
	got, err := Load(dir, "u1")
	if err != nil || got == nil || got.AppID != "cli_1" || got.MCPToken != "mcp_x" {
		t.Fatalf("got=%+v err=%v", got, err)
	}
	list, err := List(dir)
	if err != nil || len(list) != 1 {
		t.Fatalf("list=%v err=%v", list, err)
	}
	tok, err := ReloadToken(dir)
	if err != nil || tok == "" {
		t.Fatalf("reload token=%q err=%v", tok, err)
	}
	if filepath.Base(Path(dir, "u1")) != "u1.json" {
		t.Fatal(Path(dir, "u1"))
	}
	_ = os.Remove(Path(dir, "u1"))
}
