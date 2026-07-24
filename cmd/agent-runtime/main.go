package main

import (
	"context"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/ghsemail/GeeGooAgent/internal/app"
	"github.com/ghsemail/GeeGooAgent/internal/config"
	"github.com/ghsemail/GeeGooAgent/internal/httpserver"
	"github.com/ghsemail/GeeGooAgent/internal/runtimeapi"
)

func main() {
	fs := flag.NewFlagSet("agent-runtime", flag.ExitOnError)
	configPath := fs.String("config", "", "path to config.json (default GEEGOO_CONFIG)")
	_ = fs.Parse(os.Args[1:])

	rt := config.LoadRuntime()
	if *configPath != "" {
		rt.ConfigPath = *configPath
	}

	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
	slog.Info("starting agent-runtime",
		"port", rt.Port,
		"geegoo_bot_mcp", os.Getenv("GEEGOO_BOT_MCP_URL"),
		"insecure_auth", rt.AllowInsecure,
	)

	application, err := app.LoadFromConfigPath(rt.ConfigPath, false)
	if err != nil {
		slog.Error("config load failed", "error", err)
		os.Exit(1)
	}
	if application.Gateway == nil {
		slog.Warn("LLM not configured — /v1/chat/completions will return 503")
	}

	handler := httpserver.NewProtectedHandler(rt.ServiceName, rt.APIKey, rt.AllowInsecure, func(mux *http.ServeMux) {
		runtimeapi.NewHandler(application).Register(mux)
	})
	if len(rt.CORSOrigins) > 0 {
		handler = httpserver.CORS(rt.CORSOrigins, handler)
		slog.Info("CORS enabled", "origins", rt.CORSOrigins)
	}
	srv := httpserver.New(httpserver.Options{
		Name: rt.ServiceName,
		Port: rt.Port,
	}, handler)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := httpserver.Run(ctx, srv); err != nil {
		slog.Error("agent-runtime stopped", "error", err)
		os.Exit(1)
	}
}
