package config_test

import (
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/config"
)

func TestResolvedDataNodesDefault(t *testing.T) {
	cfg := &config.AppConfig{DataBaseURL: "http://data.local:3300"}
	nodes := cfg.ResolvedDataNodes()
	if len(nodes) != 1 {
		t.Fatalf("len=%d", len(nodes))
	}
	if nodes[0].ID != "default" || nodes[0].BaseURL != "http://data.local:3300" {
		t.Fatalf("node=%+v", nodes[0])
	}
}

func TestResolvedDataNodesExplicit(t *testing.T) {
	cfg := &config.AppConfig{
		DataNodes: []config.DataNodeConfig{
			{ID: "cn", Label: "A股", BaseURL: "http://cn:3300", Regions: []string{"CN"}},
			{ID: "us", Label: "港美", BaseURL: "http://us:3300/", Regions: []string{"US", "HK"}},
		},
	}
	nodes := cfg.ResolvedDataNodes()
	if len(nodes) != 2 {
		t.Fatalf("len=%d", len(nodes))
	}
	if nodes[1].BaseURL != "http://us:3300" {
		t.Fatalf("base=%q", nodes[1].BaseURL)
	}
}

func TestDataNodeByID(t *testing.T) {
	cfg := &config.AppConfig{
		DataNodes: []config.DataNodeConfig{{ID: "cn", BaseURL: "http://cn:3300"}},
	}
	if _, ok := cfg.DataNodeByID("missing"); ok {
		t.Fatal("expected missing")
	}
	n, ok := cfg.DataNodeByID("cn")
	if !ok || n.BaseURL != "http://cn:3300" {
		t.Fatalf("node=%+v ok=%v", n, ok)
	}
}
