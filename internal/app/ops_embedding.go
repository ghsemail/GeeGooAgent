package app

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/clients/admin"
	"github.com/ghsemail/GeeGooAgent/internal/config"
	"github.com/ghsemail/GeeGooAgent/internal/llm"
	"github.com/ghsemail/GeeGooAgent/internal/memory/semantic"
)

const opsEmbeddingRefreshEvery = 60 * time.Second

// catalogEmbeddingConfig maps an ops catalog embedding row onto Agent embedding settings.
func catalogEmbeddingConfig(doc admin.ConfiguredModel, keepDim int) config.EmbeddingConfig {
	name := strings.TrimSpace(doc.Name)
	if name == "" {
		name = strings.TrimSpace(doc.DisplayName)
	}
	dim := keepDim
	if dim <= 0 {
		dim = config.DefaultEmbeddingDimensions
	}
	return config.EmbeddingConfig{
		Provider:   catalogEmbeddingProvider(doc),
		TokenKey:   strings.TrimSpace(doc.Token),
		Model:      name,
		BaseURL:    strings.TrimRight(strings.TrimSpace(doc.BaseURL), "/"),
		Dimensions: dim,
	}
}

func catalogEmbeddingProvider(doc admin.ConfiguredModel) string {
	if p := strings.TrimSpace(doc.Provider); p != "" {
		return p
	}
	hay := strings.ToLower(doc.BaseURL + " " + doc.Name + " " + doc.DisplayName)
	if strings.Contains(hay, "tencentmaas") || strings.Contains(hay, "tokenhub") || strings.Contains(hay, "kinfra") {
		return config.DefaultEmbeddingProvider
	}
	return llm.InferProviderFromNames(doc.DisplayName, doc.Name)
}

func catalogModelIsEmbedding(doc admin.ConfiguredModel) bool {
	switch strings.ToLower(strings.TrimSpace(doc.Kind)) {
	case "embedding", "embed", "embeddings":
		return true
	}
	hay := strings.ToLower(doc.Name + " " + doc.DisplayName)
	return strings.Contains(hay, "embedding") || strings.Contains(hay, "kinfra-text-embedding")
}

func applyCatalogEmbedding(cfg *config.AppConfig, doc admin.ConfiguredModel) bool {
	if cfg == nil {
		return false
	}
	if !catalogModelIsEmbedding(doc) {
		return false
	}
	next := catalogEmbeddingConfig(doc, cfg.Embedding.Dimensions)
	if strings.TrimSpace(next.TokenKey) == "" || strings.TrimSpace(next.Model) == "" {
		return false
	}
	cfg.Embedding = next
	return true
}

func (a *App) catalogQueryTargets() []admin.QueryTarget {
	if a == nil || a.Config == nil {
		return nil
	}
	out := make([]admin.QueryTarget, 0, len(a.Config.AdminModelQueryTargets()))
	for _, t := range a.Config.AdminModelQueryTargets() {
		out = append(out, admin.QueryTarget{BaseURL: t.BaseURL, Bearer: t.Bearer})
	}
	return out
}

func (a *App) RefreshOpsEmbedding(force bool) {
	if a == nil || a.Config == nil {
		return
	}
	a.embeddingMu.Lock()
	defer a.embeddingMu.Unlock()
	if !force && !a.embeddingRefreshedAt.IsZero() && time.Since(a.embeddingRefreshedAt) < opsEmbeddingRefreshEvery {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	view, err := admin.QueryModelRuntimeConfigFromTargets(ctx, a.catalogQueryTargets())
	a.embeddingRefreshedAt = time.Now()
	if err != nil {
		fmt.Fprintf(os.Stderr, "警告: 拉取运营 Embedding 失败: %v\n", err)
		a.embeddingSource = "unset"
		return
	}
	if view.EmbeddingModel == nil || strings.TrimSpace(view.EmbeddingModelID) == "" {
		a.embeddingSource = "unset"
		fmt.Fprintf(os.Stderr, "提示: 运营台未配置 Embedding，请到 Monday → 模型管理选择 Embedding 并保存\n")
		return
	}
	if !catalogModelIsEmbedding(*view.EmbeddingModel) {
		a.embeddingSource = "unset"
		fmt.Fprintf(os.Stderr, "警告: 运营 embedding_model_id 不是 Embedding 类型，请检查模型管理\n")
		return
	}
	if !applyCatalogEmbedding(a.Config, *view.EmbeddingModel) {
		a.embeddingSource = "unset"
		fmt.Fprintf(os.Stderr, "警告: 运营 Embedding 缺少 token/模型名，请在模型管理中补全\n")
		return
	}
	a.embeddingSource = "ops"
	a.embeddingCatalogID = strings.TrimSpace(view.EmbeddingModelID)
	if a.Semantic != nil {
		a.Semantic.SetEmbedder(semantic.NewEmbedderFromResolved(a.Config.ResolvedEmbedding()))
	}
	fmt.Fprintf(os.Stderr, "embedding: model=%s base_url=%s from ops catalog\n",
		a.Config.Embedding.Model, a.Config.Embedding.BaseURL)
}

func (a *App) EmbeddingSource() string {
	if a == nil {
		return "unset"
	}
	if strings.TrimSpace(a.embeddingSource) == "" {
		return "unset"
	}
	return a.embeddingSource
}

func (a *App) EmbeddingCatalogID() string {
	if a == nil {
		return ""
	}
	return a.embeddingCatalogID
}
