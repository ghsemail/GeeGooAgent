package config

import (
	"strings"
	"testing"
)

func TestApplyProfileOverrides(t *testing.T) {
	t.Setenv("GEEGOO_PROFILE", "work")
	cfg := &AppConfig{
		OutputDir:    "/data/default",
		UserMCPToken: "root-token",
		Profiles: map[string]ProfileConfig{
			"work": {
				OutputDir:    "/data/work",
				MCPToken:     "work-token",
				ChatToolsets: []string{"market"},
			},
		},
	}
	applyProfile(cfg)
	if cfg.ResolvedProfile != "work" {
		t.Fatalf("profile=%q", cfg.ResolvedProfile)
	}
	if cfg.OutputDir != "/data/work" || cfg.UserMCPToken != "work-token" {
		t.Fatalf("cfg=%+v", cfg)
	}
	if len(cfg.ChatToolsets) != 1 || cfg.ChatToolsets[0] != "market" {
		t.Fatalf("toolsets=%v", cfg.ChatToolsets)
	}
}

func TestApplyProfileDefaultWithoutMap(t *testing.T) {
	cfg := &AppConfig{OutputDir: "/data"}
	applyProfile(cfg)
	if cfg.ResolvedProfile != "default" || cfg.OutputDir != "/data" {
		t.Fatalf("cfg=%+v", cfg)
	}
}

func TestProfileSummaryWithOverrides(t *testing.T) {
	t.Setenv("GEEGOO_PROFILE", "work")
	cfg := &AppConfig{
		OutputDir: "/data/root",
		Profiles: map[string]ProfileConfig{
			"work": {
				OutputDir:    "/data/work",
				ChatToolsets: []string{"market"},
			},
		},
	}
	applyProfile(cfg)
	summary := cfg.ProfileSummary()
	if !strings.Contains(summary, "work via GEEGOO_PROFILE") {
		t.Fatalf("summary=%q", summary)
	}
	if !strings.Contains(summary, "output_dir=/data/work") || !strings.Contains(summary, "chat_toolsets=market") {
		t.Fatalf("summary=%q", summary)
	}
	if !cfg.ProfileOverridesApplied() {
		t.Fatal("expected overrides applied")
	}
}

func TestProfileSummaryUndefinedProfile(t *testing.T) {
	cfg := &AppConfig{
		ActiveProfile: "missing",
		Profiles: map[string]ProfileConfig{
			"work": {OutputDir: "/data/work"},
		},
	}
	applyProfile(cfg)
	if cfg.ProfileOverridesApplied() {
		t.Fatal("expected no overrides")
	}
	if !strings.Contains(cfg.ProfileSummary(), "no overrides") {
		t.Fatalf("summary=%q", cfg.ProfileSummary())
	}
}
