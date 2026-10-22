package app

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/clients/admin"
	"github.com/ghsemail/GeeGooAgent/internal/llm"
)

func (a *App) signalDiagnoseComposeFromEnv() llm.Provider {
	id := strings.TrimSpace(os.Getenv("SIGNAL_DIAGNOSE_COMPOSE_MODEL"))
	if id == "" || a == nil || a.Config == nil {
		return nil
	}
	targets := a.adminQueryTargets()
	if len(targets) == 0 {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	doc, _, err := admin.QueryModelFromTargets(ctx, targets, id, false)
	if err != nil {
		fmt.Fprintf(os.Stderr, "警告: SIGNAL_DIAGNOSE_COMPOSE_MODEL=%q: %v\n", id, err)
		return nil
	}
	p, err := a.composeProviderFromCatalogDoc(doc)
	if err != nil {
		fmt.Fprintf(os.Stderr, "警告: compose 模型 %q 构建失败: %v\n", id, err)
		return nil
	}
	return p
}
