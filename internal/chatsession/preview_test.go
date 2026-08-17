package chatsession

import "testing"

func TestListPreviewPrefersSummary(t *testing.T) {
	got := ListPreview(ChatSessionIndexEntry{
		ID: "chat-abc", Title: "chat-abc", Summary: "腾讯股价多少",
	})
	if got != "腾讯股价多少" {
		t.Fatalf("got %q", got)
	}
}

func TestListPreviewUsesTitleWhenNotID(t *testing.T) {
	got := ListPreview(ChatSessionIndexEntry{
		ID: "chat-abc", Title: "分析小米",
	})
	if got != "分析小米" {
		t.Fatalf("got %q", got)
	}
}
