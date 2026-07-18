package config

import "testing"

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
