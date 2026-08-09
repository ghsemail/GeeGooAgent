package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/app"
	"github.com/ghsemail/GeeGooAgent/internal/config"
	"github.com/ghsemail/GeeGooAgent/internal/gateway"
	"github.com/ghsemail/GeeGooAgent/internal/gateway/feishustore"
	"github.com/ghsemail/GeeGooAgent/internal/gateway/multitenant"
	"github.com/ghsemail/GeeGooAgent/internal/gateway/platforms/feishu"
)

func runGateway(args []string) {
	_ = config.LoadGeeGooDotEnv()
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: geegoo gateway <run|status|setup> [flags]")
		os.Exit(2)
	}
	switch args[0] {
	case "run":
		runGatewayRun(args[1:])
	case "status":
		runGatewayStatus(args[1:])
	case "setup":
		runGatewaySetup(args[1:])
	default:
		fmt.Fprintf(os.Stderr, "geegoo gateway: unknown subcommand %q (try: run, status, setup)\n", args[0])
		os.Exit(2)
	}
}

func runGatewayRun(args []string) {
	fs := flag.NewFlagSet("gateway run", flag.ExitOnError)
	configPath := fs.String("config", config.DefaultPath(), "path to config.json")
	dryRun := fs.Bool("dry-run", false, "skip sending replies to Feishu")
	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}
	application, err := app.LoadFromConfigPath(*configPath, *dryRun)
	if err != nil {
		fmt.Fprintf(os.Stderr, "gateway: %v\n", err)
		os.Exit(2)
	}
	defer application.Close()

	workspace, err := application.Config.ResolveOutputDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "gateway: %v\n", err)
		os.Exit(2)
	}
	sessions, err := gateway.NewSessionMap(workspace)
	if err != nil {
		fmt.Fprintf(os.Stderr, "gateway: session map: %v\n", err)
		os.Exit(2)
	}

	// Prefer per-user store; optionally claim host .env into FEISHU_OWNER_USER_ID once.
	if owner := strings.TrimSpace(os.Getenv("FEISHU_OWNER_USER_ID")); owner != "" {
		_, _ = migrateHostEnvToUser(workspace, owner)
	}

	runner := multitenant.NewRunner(application, sessions, workspace, *dryRun || application.Config.DryRun)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	fmt.Println("geegoo gateway started (multi-tenant feishu); Ctrl+C to stop.")
	if err := runner.Start(ctx); err != nil && err != context.Canceled {
		fmt.Fprintf(os.Stderr, "gateway: %v\n", err)
		os.Exit(1)
	}
	slog.Info("gateway: stopped")
}

func migrateHostEnvToUser(outputDir, userID string) (*feishustore.Creds, error) {
	existing, err := feishustore.Load(outputDir, userID)
	if err != nil {
		return nil, err
	}
	if existing != nil && existing.Configured() {
		return existing, nil
	}
	_ = config.LoadGeeGooDotEnv()
	envCfg := feishu.LoadConfigFromEnv(os.Getenv)
	if strings.TrimSpace(envCfg.AppID) == "" || strings.TrimSpace(envCfg.AppSecret) == "" {
		return nil, nil
	}
	list, _ := feishustore.List(outputDir)
	for _, c := range list {
		if c.AppID == envCfg.AppID {
			return nil, nil
		}
	}
	doc := &feishustore.Creds{
		UserID:       userID,
		AppID:        envCfg.AppID,
		AppSecret:    envCfg.AppSecret,
		Domain:       envCfg.Domain,
		BotName:      envCfg.BotName,
		BotOpenID:    envCfg.BotOpenID,
		AllowedUsers: append([]string(nil), envCfg.AllowedUsers...),
		HomeChannel:  envCfg.HomeChannel,
		GroupPolicy:  envCfg.GroupPolicy,
		Enabled:      true,
	}
	if err := feishustore.Save(outputDir, doc); err != nil {
		return nil, err
	}
	fmt.Printf("gateway: migrated host Feishu env → user %s\n", userID)
	return doc, nil
}

func runGatewayStatus(args []string) {
	fs := flag.NewFlagSet("gateway status", flag.ExitOnError)
	_ = fs.String("config", config.DefaultPath(), "path to config.json")
	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}
	fsCfg := feishu.LoadConfigFromEnv(os.Getenv)
	ad := feishu.NewAdapter(fsCfg)
	st := ad.Status()
	fmt.Printf("platform: %s\n", st.Platform)
	fmt.Printf("configured: %v\n", st.Configured)
	fmt.Printf("connected: %v (only meaningful while `gateway run` is active)\n", st.Connected)
	fmt.Printf("detail: %s\n", st.Detail)
	fmt.Printf("env_file: %s\n", config.EnvFilePath())
	if len(fsCfg.AllowedUsers) == 0 {
		fmt.Println("allowed_users: (empty — all users accepted unless FEISHU_ALLOW_ALL_USERS is used intentionally)")
	} else {
		fmt.Printf("allowed_users: %d configured\n", len(fsCfg.AllowedUsers))
	}
	if fsCfg.HomeChannel != "" {
		fmt.Printf("home_channel: %s\n", fsCfg.HomeChannel)
	}
	if !st.Configured {
		os.Exit(1)
	}
}

func runGatewaySetup(args []string) {
	fs := flag.NewFlagSet("gateway setup", flag.ExitOnError)
	domain := fs.String("domain", "feishu", "feishu (China) or lark (International)")
	manual := fs.Bool("manual", false, "enter App ID / App Secret manually instead of QR scan")
	timeout := fs.Duration("timeout", 10*time.Minute, "QR scan wait timeout")
	noQR := fs.Bool("no-qr", false, "do not render ASCII QR (print URL only)")
	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}
	*domain = strings.ToLower(strings.TrimSpace(*domain))
	if *domain != "feishu" && *domain != "lark" {
		fmt.Fprintf(os.Stderr, "gateway setup: invalid --domain %q (use feishu or lark)\n", *domain)
		os.Exit(2)
	}

	existing := feishu.LoadConfigFromEnv(os.Getenv)
	if existing.AppID != "" && existing.AppSecret != "" {
		fmt.Printf("Feishu already configured (app_id=%s).\n", existing.AppID)
		if !promptYesNo("Reconfigure Feishu / Lark?", false) {
			return
		}
	}

	var creds *feishu.Credentials
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if !*manual {
		fmt.Println("=== Feishu / Lark setup (QR scan-to-create) ===")
		fmt.Println("Scan with the Feishu/Lark mobile app to create a bot automatically.")
		var err error
		creds, err = feishu.QRRegister(ctx, feishu.RegisterOptions{
			Domain:       *domain,
			Timeout:      *timeout,
			Stdout:       os.Stdout,
			SkipQRRender: *noQR,
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "QR registration failed: %v\n", err)
			if !promptYesNo("Fall back to manual App ID / App Secret?", true) {
				os.Exit(1)
			}
			*manual = true
		}
	}

	if *manual {
		fmt.Println("=== Feishu / Lark setup (manual) ===")
		fmt.Println("Create an app at https://open.feishu.cn/ (or https://open.larksuite.com/),")
		fmt.Println("enable Bot capability, then paste credentials.")
		appID := promptLine("App ID")
		appSecret := promptLine("App Secret")
		if appID == "" || appSecret == "" {
			fmt.Fprintln(os.Stderr, "App ID and App Secret are required")
			os.Exit(2)
		}
		dom := *domain
		if d := promptLine(fmt.Sprintf("Domain [%s]", dom)); d != "" {
			dom = strings.ToLower(d)
		}
		creds = &feishu.Credentials{AppID: appID, AppSecret: appSecret, Domain: dom}
		if name, oid, err := feishu.ProbeBot(ctx, appID, appSecret, dom); err == nil {
			creds.BotName = name
			creds.BotOpenID = oid
			fmt.Printf("Credentials verified — bot: %s\n", name)
		} else {
			fmt.Printf("Could not verify bot yet (%v); credentials will still be saved.\n", err)
		}
	}

	if creds == nil {
		fmt.Fprintln(os.Stderr, "setup cancelled")
		os.Exit(1)
	}

	extra := map[string]string{}
	if creds.BotOpenID != "" {
		extra["FEISHU_BOT_OPEN_ID"] = creds.BotOpenID
	}
	if creds.BotName != "" {
		extra["FEISHU_BOT_NAME"] = creds.BotName
	}
	// Prefer pairing the scanning user onto the allowlist when available.
	if creds.OpenID != "" {
		extra["FEISHU_ALLOWED_USERS"] = creds.OpenID
	}

	if err := config.SaveFeishuEnv(creds.AppID, creds.AppSecret, creds.Domain, extra); err != nil {
		fmt.Fprintf(os.Stderr, "save env: %v\n", err)
		os.Exit(1)
	}

	fmt.Println()
	fmt.Println("Feishu / Lark configured.")
	fmt.Printf("  App ID:   %s\n", creds.AppID)
	fmt.Printf("  Domain:   %s\n", creds.Domain)
	if creds.BotName != "" {
		fmt.Printf("  Bot:      %s\n", creds.BotName)
	}
	if creds.BotOpenID != "" {
		fmt.Printf("  Bot OID:  %s\n", creds.BotOpenID)
	}
	fmt.Printf("  Saved to: %s\n", config.EnvFilePath())
	fmt.Println()
	fmt.Println("Next: ensure the Feishu app uses 长连接 (WebSocket) and subscribes to im.message.receive_v1,")
	fmt.Println("then run:  geegoo gateway run")
}

func promptLine(label string) string {
	fmt.Printf("%s: ", label)
	sc := bufio.NewScanner(os.Stdin)
	if !sc.Scan() {
		return ""
	}
	return strings.TrimSpace(sc.Text())
}

func promptYesNo(question string, def bool) bool {
	hint := "y/N"
	if def {
		hint = "Y/n"
	}
	fmt.Printf("%s [%s]: ", question, hint)
	sc := bufio.NewScanner(os.Stdin)
	if !sc.Scan() {
		return def
	}
	s := strings.ToLower(strings.TrimSpace(sc.Text()))
	if s == "" {
		return def
	}
	return s == "y" || s == "yes"
}
