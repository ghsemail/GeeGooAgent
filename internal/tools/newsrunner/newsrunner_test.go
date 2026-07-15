package newsrunner_test

import (
	"errors"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/tools/newsrunner"
)

func TestMarketNewsUnavailableWithoutScript(t *testing.T) {
	t.Setenv("GEEGOO_NEWS_DISABLE_GO", "1")
	_, err := newsrunner.MarketNews(t.Context(), newsrunner.Options{ProjectRoot: t.TempDir(), BundledOnly: true}, "US", 3)
	if !errors.Is(err, newsrunner.ErrUnavailable) {
		t.Fatalf("expected ErrUnavailable, got %v", err)
	}
}
