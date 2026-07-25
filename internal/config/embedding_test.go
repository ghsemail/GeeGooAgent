package config

import "testing"

func TestResolvedEmbeddingFromConfig(t *testing.T) {
	cfg := &AppConfig{
		Embedding: EmbeddingConfig{
			Provider:   "tencent-maas",
			TokenKey:   "sk-test",
			Model:      "kinfra-text-embedding-4b",
			BaseURL:    "https://tokenhub.tencentmaas.com/v1",
			Dimensions: 2560,
		},
	}
	got := cfg.ResolvedEmbedding()
	if !got.Configured || got.Model != "kinfra-text-embedding-4b" || got.Dimensions != 2560 {
		t.Fatalf("ResolvedEmbedding() = %+v", got)
	}
}

func TestResolvedEmbeddingDefaults(t *testing.T) {
	got := (&AppConfig{}).ResolvedEmbedding()
	if got.Model != DefaultEmbeddingModel || got.Dimensions != DefaultEmbeddingDimensions {
		t.Fatalf("defaults = %+v", got)
	}
	if got.Configured {
		t.Fatal("expected not configured without token")
	}
}
