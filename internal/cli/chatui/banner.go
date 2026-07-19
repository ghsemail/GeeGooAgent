package chatui

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/tools"
)

// BannerOptions feeds welcome banner content.
type BannerOptions struct {
	SessionID   string
	Provider    string
	Model       string
	Registry    *tools.Registry
	ToolNames   []string // chat allowlist; empty = all registered
	Thinking    bool
	DryRun      bool
	Workspace   string
	InstallDir  string
	ProjectRoot string
	APIHosts    map[string]string
	Revision    string
}

// ResolveRevision returns short git rev or "local".
func ResolveRevision(installDir string) string {
	if v := strings.TrimSpace(os.Getenv("GEEGOO_REVISION")); v != "" {
		if len(v) > 12 {
			return v[:12]
		}
		return v
	}
	if installDir == "" {
		return "local"
	}
	out, err := exec.Command("git", "-C", installDir, "rev-parse", "--short", "HEAD").Output()
	if err != nil {
		return "local"
	}
	return strings.TrimSpace(string(out))
}

// BuildPlainBanner matches Python build_plain_banner.
func BuildPlainBanner(opts BannerOptions) string {
	rev := opts.Revision
	if rev == "" {
		rev = ResolveRevision(opts.InstallDir)
	}
	var lines []string
	lines = append(lines, fmt.Sprintf("GeeGoo Agent · upstream %s", rev))
	lines = append(lines, fmt.Sprintf("Model: %s / %s", opts.Provider, opts.Model))
	lines = append(lines, fmt.Sprintf("Session: %s", opts.SessionID))
	if opts.Workspace != "" {
		lines = append(lines, fmt.Sprintf("CWD: %s", opts.Workspace))
	}
	think := "off"
	if opts.Thinking {
		think = "on"
	}
	dry := "off"
	if opts.DryRun {
		dry = "on"
	}
	lines = append(lines, fmt.Sprintf("Think: %s  Dry-run: %s", think, dry))
	lines = append(lines, "")
	lines = append(lines, "Available Tools:")
	for label, names := range groupTools(opts.Registry, opts.ToolNames) {
		if len(names) > 0 {
			lines = append(lines, fmt.Sprintf("  %s: %s", label, strings.Join(names, ", ")))
		}
	}
	if len(opts.APIHosts) > 0 {
		lines = append(lines, "")
		lines = append(lines, "APIs:")
		for k, v := range opts.APIHosts {
			lines = append(lines, fmt.Sprintf("  %s: %s", k, v))
		}
	}
	skills := scanSkills(opts.ProjectRoot)
	if len(skills) > 0 {
		lines = append(lines, "")
		lines = append(lines, "Skills:")
		for cat, names := range skills {
			lines = append(lines, fmt.Sprintf("  %s: %s", cat, strings.Join(names, ", ")))
		}
	}
	totalTools := len(toolNamesForBanner(opts))
	totalSkills := 0
	for _, v := range skills {
		totalSkills += len(v)
	}
	lines = append(lines, "")
	lines = append(lines, fmt.Sprintf("%d tools · %d skills · /help for commands", totalTools, totalSkills))
	lines = append(lines, "")
	lines = append(lines, "Welcome to GeeGoo Agent! Type your message or /help for commands.")
	lines = append(lines, "")
	for _, line := range welcomeTipLines() {
		lines = append(lines, "  ✦ "+line)
	}
	return strings.Join(lines, "\n") + "\n"
}

// welcomeTipLines returns onboarding hints shown under the welcome banner.
func welcomeTipLines() []string {
	return []string{
		"个股信号/走势：例如「分析 SpaceX 信号趋势」— 自动 search_code → 取 prompt 模板 → 技术信号分析",
		"/details collapsed 折叠思考与工具 · Space 切换折叠块 · Ctrl+X 多 live session",
		"/think on 开启 DeepSeek 推理 · /verbose on 展开过程 · /help 查看全部斜杠命令",
		"/toolsets 切换工具分组（market / bot_manager 等）· /tools 列出当前可用 Tool",
		"/recall 关键词 搜索历史会话（含 /exit 后的 closed 会话）· /dry-run on 跳过写 API",
		"查交易 Bot：list_smart_trades / list_dca_bots；不要用 get_report_bot_codes 列机器人",
		"查价先 search_code（如 SPCX、腾讯）再 get_current_price；拼写 SpaceX 勿写成 Xspace",
	}
}

// RenderWelcomeTips returns styled multi-line tips for the Hermes welcome panel.
func RenderWelcomeTips() string {
	var b strings.Builder
	b.WriteString(styleWhisper.Render("✦ Tips:"))
	b.WriteByte('\n')
	for _, tip := range welcomeTipLines() {
		b.WriteString(styleMeta.Render("  · "))
		b.WriteString(styleBody.Render(tip))
		b.WriteByte('\n')
	}
	return strings.TrimRight(b.String(), "\n")
}

func buildBannerLeft(opts BannerOptions) string {
	modelShort := opts.Model
	if i := strings.LastIndex(modelShort, "/"); i >= 0 {
		modelShort = modelShort[i+1:]
	}
	if len(modelShort) > 28 {
		modelShort = modelShort[:25] + "..."
	}
	think := "off"
	if opts.Thinking {
		think = "on"
	}
	dry := "off"
	if opts.DryRun {
		dry = "on"
	}
	var lines []string
	lines = append(lines,
		styleBrand.Render(modelShort)+styleMeta.Render(" · ")+styleMeta.Render(opts.Provider),
		styleWhisper.Render(fmt.Sprintf("think %s · dry-run %s", think, dry)),
	)
	if opts.Workspace != "" {
		cwd := opts.Workspace
		if len(cwd) > 36 {
			cwd = "…" + cwd[len(cwd)-35:]
		}
		lines = append(lines, styleWhisper.Render(cwd))
	}
	lines = append(lines, styleMeta.Render("Session: "+opts.SessionID))
	return strings.Join(lines, "\n")
}

func buildBannerRight(opts BannerOptions) string {
	lines := []string{styleTitle.Render("Available Tools")}
	groups := groupTools(opts.Registry, opts.ToolNames)
	order := []string{"perceive", "analyze", "decide", "act", "meta", "other"}
	shown := 0
	for _, label := range order {
		names := groups[label]
		if len(names) == 0 {
			continue
		}
		lines = append(lines, styleMeta.Render(label+":")+" "+styleBody.Render(truncateToolList(names, 44)))
		shown++
		if shown >= 7 {
			break
		}
	}
	if len(opts.APIHosts) > 0 {
		lines = append(lines, "")
		lines = append(lines, styleTitle.Render("APIs"))
		for k, v := range opts.APIHosts {
			lines = append(lines, styleMeta.Render(k)+" "+styleBody.Render("("+v+")"))
		}
	}
	skills := scanSkills(opts.ProjectRoot)
	totalSkills := 0
	for _, v := range skills {
		totalSkills += len(v)
	}
	lines = append(lines, "")
	lines = append(lines, styleTitle.Render("Available Skills"))
	if totalSkills == 0 {
		lines = append(lines, styleWhisper.Render("No skills in project"))
	} else {
		count := 0
		for cat, names := range skills {
			lines = append(lines, styleMeta.Render(cat+":")+" "+styleBody.Render(truncateToolList(names, 52)))
			count++
			if count >= 8 {
				break
			}
		}
	}
	totalTools := len(toolNamesForBanner(opts))
	lines = append(lines, "")
	lines = append(lines, styleWhisper.Render(fmt.Sprintf("%d tools · %d skills · /help for commands", totalTools, totalSkills)))
	return strings.Join(lines, "\n")
}

func groupTools(registry *tools.Registry, names []string) map[string][]string {
	out := map[string][]string{
		"perceive": {}, "analyze": {}, "decide": {}, "act": {}, "meta": {}, "other": {},
	}
	if len(names) == 0 && registry != nil {
		names = registry.ListNames()
	}
	for _, name := range names {
		label := toolCategory(name)
		out[label] = append(out[label], name)
	}
	return out
}

func toolCategory(name string) string {
	switch {
	case strings.HasPrefix(name, "search_"), name == "recall":
		return "perceive"
	case strings.HasPrefix(name, "get_"), strings.HasPrefix(name, "fetch_"), strings.Contains(name, "analysis"):
		return "analyze"
	case strings.HasPrefix(name, "generate_"), strings.HasPrefix(name, "loopback"):
		return "decide"
	case strings.HasPrefix(name, "create_"), strings.HasPrefix(name, "update_"), strings.HasPrefix(name, "delete_"):
		return "act"
	case strings.HasPrefix(name, "read_"), name == "write_execution_log":
		return "meta"
	default:
		return "other"
	}
}

func truncateToolList(names []string, maxLen int) string {
	if len(names) == 0 {
		return ""
	}
	parts := []string{}
	length := 0
	for _, name := range names {
		extra := len(name)
		if len(parts) > 0 {
			extra += 2
		}
		if length+extra > maxLen {
			parts = append(parts, "...")
			break
		}
		parts = append(parts, name)
		length += extra
	}
	return strings.Join(parts, ", ")
}

func scanSkills(projectRoot string) map[string][]string {
	if projectRoot == "" {
		return nil
	}
	skillsDir := filepath.Join(projectRoot, "skills")
	entries, err := os.ReadDir(skillsDir)
	if err != nil {
		return nil
	}
	found := map[string][]string{}
	_ = filepath.WalkDir(skillsDir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || d.Name() != "SKILL.md" {
			return nil
		}
		rel, err := filepath.Rel(skillsDir, filepath.Dir(path))
		if err != nil || rel == "." {
			return nil
		}
		parts := strings.Split(rel, string(os.PathSeparator))
		category := parts[0]
		name := parts[len(parts)-1]
		if category == "bundled" && len(parts) > 1 {
			category = "bundled"
			name = parts[1]
		}
		found[category] = append(found[category], name)
		return nil
	})
	_ = entries
	return found
}

// APIHostsFromConfig builds short host map for banner.
func APIHostsFromConfig(mcpURL, signalURL, dataURL string) map[string]string {
	hosts := map[string]string{}
	if h := shortHost(mcpURL); h != "" {
		hosts["geegoo-bot"] = h
	}
	if h := shortHost(signalURL); h != "" {
		hosts["signal"] = h
	}
	if h := shortHost(dataURL); h != "" {
		hosts["data"] = h
	}
	return hosts
}

func toolNamesForBanner(opts BannerOptions) []string {
	if len(opts.ToolNames) > 0 {
		return opts.ToolNames
	}
	if opts.Registry == nil {
		return nil
	}
	return opts.Registry.ListNames()
}

func formatVersionLabel(rev string) string {
	return fmt.Sprintf("GeeGoo Agent · upstream %s", rev)
}

func shortHost(raw string) string {
	s := strings.TrimSpace(raw)
	for _, p := range []string{"https://", "http://"} {
		if strings.HasPrefix(s, p) {
			s = s[len(p):]
		}
	}
	return strings.TrimRight(s, "/")
}
