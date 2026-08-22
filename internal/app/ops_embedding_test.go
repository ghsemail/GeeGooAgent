package app

import (
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/clients/admin"
	"github.com/ghsemail/GeeGooAgent/internal/config"
)

func TestCatalogEmbeddingConfig_kinfra(t *testing.T) {
	got := catalogEmbeddingConfig(admin.ConfiguredModel{
		Name:        "kinfra-text-embedding-4b",
		DisplayName: "kinfra Embedding",
		Token:       "sk-emb",
		BaseURL:     "https://tokenhub.tencentmaas.com/v1",
	}, 0)
	if got.Model != "kinfra-text-embedding-4b" {
		t.Fatalf("model=%q", got.Model)
	}
	if got.Provider != config.DefaultEmbeddingProvider {
		t.Fatalf("provider=%q", got.Provider)
	}
	if got.Dimensions != config.DefaultEmbeddingDimensions {
		t.Fatalf("dim=%d", got.Dimensions)
	}
	if got.TokenKey != "sk-emb" {
		t.Fatalf("token=%q", got.TokenKey)
	}
}

func TestApplyCatalogEmbedding(t *testing.T) {
	cfg := &config.AppConfig{
		Embedding: config.EmbeddingConfig{Model: "local-model", Dimensions: 1536},
	}
	ok := applyCatalogEmbedding(cfg, admin.ConfiguredModel{
		Name:    "kinfra-text-embedding-4b",
		Kind:    "embedding",
		Token:   "sk-ops",
		BaseURL: "https://tokenhub.tencentmaas.com/v1",
	})
	if !ok {
		t.Fatal("expected apply")
	}
	if cfg.Embedding.Model != "kinfra-text-embedding-4b" || cfg.Embedding.TokenKey != "sk-ops" {
		t.Fatalf("overlay = %+v", cfg.Embedding)
	}
	if cfg.Embedding.Dimensions != 1536 {
		t.Fatalf("keep dim got %d", cfg.Embedding.Dimensions)
	}
}

func TestApplyCatalogEmbedding_requiresEmbeddingKind(t *testing.T) {
	cfg := &config.AppConfig{Embedding: config.EmbeddingConfig{Model: "local"}}
	if applyCatalogEmbedding(cfg, admin.ConfiguredModel{
		Name: "deepseek-v4", Token: "sk", BaseURL: "https://api.deepseek.com",
		Kind: "chat",
	}) {
		t.Fatal("chat model must not overlay embedding")
	}
}

func TestApplyCatalogEmbedding_requiresToken(t *testing.T) {
	cfg := &config.AppConfig{Embedding: config.EmbeddingConfig{Model: "local"}}
	if applyCatalogEmbedding(cfg, admin.ConfiguredModel{Name: "kinfra-text-embedding-4b"}) {
		t.Fatal("empty token should not overlay")
	}
	if cfg.Embedding.Model != "local" {
		t.Fatalf("local overlayed: %+v", cfg.Embedding)
	}
}
