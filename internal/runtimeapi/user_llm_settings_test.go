package runtimeapi

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/config"
)

func TestUserLLMSettingsMergeAndPersist(t *testing.T) {
	dir := t.TempDir()
	on := true
	us := &userLLMSettings{
		Thinking:    "on",
		UseOpsModel: &on,
		Pinned:      []string{"geegoo:test-model"},
	}
	us.applyRequest(dashboardSettingsRequest{
		Provider: "deepseek",
		Model:    "deepseek-chat",
	})
	if err := saveUserLLMSettings(dir, "user-1", us); err != nil {
		t.Fatal(err)
	}
	loaded, err := loadUserLLMSettings(dir, "user-1")
	if err != nil || loaded == nil {
		t.Fatalf("load: %v %+v", err, loaded)
	}
	base := config.LLMConfig{Provider: "geegoo", Model: "base", Temperature: 0.5}
	merged := loaded.mergeInto(base)
	if merged.Provider != "deepseek" || merged.Model != "deepseek-chat" {
		t.Fatalf("merge: %+v", merged)
	}
	if merged.Thinking == nil || !*merged.Thinking {
		t.Fatalf("thinking: %+v", merged.Thinking)
	}
	path := userLLMSettingsPath(dir, "user/2")
	if filepath.Base(path) != "user_2.json" {
		t.Fatalf("path=%s", path)
	}
	_ = os.Remove(path)
}

func TestUserLLMSettingsGatewayOverlay(t *testing.T) {
	dir := t.TempDir()
	us := &userLLMSettings{
		CatalogModelID: "global-model",
		Gateways: map[string]gatewayLLMSettings{
			"web":         {CatalogModelID: "web-model"},
			"trading_app": {CatalogModelID: "app-model"},
		},
	}
	base := config.LLMConfig{Provider: "geegoo", Model: "base"}
	web := us.mergeInto(base)
	web = us.Gateways["web"].mergeInto(web)
	if web.CatalogModelID != "web-model" {
		t.Fatalf("web catalog=%q", web.CatalogModelID)
	}
	app := us.mergeInto(base)
	app = us.Gateways["trading_app"].mergeInto(app)
	if app.CatalogModelID != "app-model" {
		t.Fatalf("app catalog=%q", app.CatalogModelID)
	}
	us.applyRequest(dashboardSettingsRequest{
		Gateway:        "trading_app",
		CatalogModelID: "new-app-model",
	})
	if us.Gateways["trading_app"].CatalogModelID != "new-app-model" {
		t.Fatalf("gateways=%+v", us.Gateways)
	}
	if us.CatalogModelID != "global-model" {
		t.Fatalf("global should be unchanged: %q", us.CatalogModelID)
	}
	_ = dir
}

func TestPinnedFromUserSettings(t *testing.T) {
	us := &userLLMSettings{Pinned: []string{"a:m1", "b:m2"}}
	got := pinnedFromUserSettings(us, "x", "y")
	if len(got) != 2 {
		t.Fatalf("got=%+v", got)
	}
}
