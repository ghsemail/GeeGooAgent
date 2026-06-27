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
	return strings.Join(lines, "\n") + "\n"
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
	lines := []string{"", renderHero(), "",
		styleAmber.Render(modelShort) + styleDim.Render(" · ") + styleDim.Render(opts.Provider),
		styleDim.Render(fmt.Sprintf("think %s · dry-run %s", think, dry)),
	}
	if opts.Workspace != "" {
		cwd := opts.Workspace
		if len(cwd) > 36 {
			cwd = "…" + cwd[len(cwd)-35:]
		}
		lines = append(lines, styleDim.Render(cwd))
	}
	lines = append(lines, styleDim.Render("Session: "+opts.SessionID))
	return strings.Join(lines, "\n")
}

func buildBannerRight(opts BannerOptions) string {
	lines := []string{styleAmber.Render("Available Tools")}
	groups := groupTools(opts.Registry, opts.ToolNames)
	order := []string{"perceive", "analyze", "decide", "act", "meta", "other"}
	shown := 0
	for _, label := range order {
		names := groups[label]
		if len(names) == 0 {
			continue
		}
		lines = append(lines, styleDim.Render(label+":")+" "+styleText.Render(truncateToolList(names, 44)))
		shown++
		if shown >= 7 {
			break
		}
	}
	if len(opts.APIHosts) > 0 {
		lines = append(lines, "")
		lines = append(lines, styleAmber.Render("APIs"))
		for k, v := range opts.APIHosts {
			lines = append(lines, styleDim.Render(k)+" "+styleText.Render("("+v+")"))
		}
	}
	skills := scanSkills(opts.ProjectRoot)
	totalSkills := 0
	for _, v := range skills {
		totalSkills += len(v)
	}
	lines = append(lines, "")
	lines = append(lines, styleAmber.Render("Available Skills"))
	if totalSkills == 0 {
		lines = append(lines, styleDim.Render("No skills in project"))
	} else {
		count := 0
		for cat, names := range skills {
			lines = append(lines, styleDim.Render(cat+":")+" "+styleText.Render(truncateToolList(names, 52)))
			count++
			if count >= 8 {
				break
			}
		}
	}
	totalTools := len(toolNamesForBanner(opts))
	lines = append(lines, "")
	lines = append(lines, styleDim.Render(fmt.Sprintf("%d tools · %d skills · /help for commands", totalTools, totalSkills)))
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
