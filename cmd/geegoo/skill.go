package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/config"
)

func runSkillManage(args []string) {
	if len(args) == 0 {
		printSkillUsage()
		os.Exit(2)
	}
	switch args[0] {
	case "install":
		runSkillInstall(args[1:])
	case "enable":
		runSkillToggle(args[1:], true)
	case "disable":
		runSkillToggle(args[1:], false)
	default:
		fmt.Fprintf(os.Stderr, "geegoo skill: unknown subcommand %q\n", args[0])
		printSkillUsage()
		os.Exit(2)
	}
}

func printSkillUsage() {
	fmt.Fprintf(os.Stderr, `usage:
  geegoo skill install <name>   (not implemented — use ~/.geegoo/skills/ or repo skills/extensions/)
  geegoo skill enable <name>    toggle extension in config.json skills.extensions
  geegoo skill disable <name>   disable skill in config.json skills.disabled

`)
}

func runSkillInstall(args []string) {
	if len(args) == 0 || strings.TrimSpace(args[0]) == "" {
		fmt.Fprintln(os.Stderr, "geegoo skill install: missing skill name")
		os.Exit(2)
	}
	name := strings.TrimSpace(args[0])
	home, _ := os.UserHomeDir()
	userDir := filepath.Join(home, ".geegoo", "skills", name)
	fmt.Printf("geegoo skill install %q is not implemented yet.\n", name)
	fmt.Println()
	fmt.Println("Options:")
	fmt.Printf("  1. Copy SKILL.md into %s/\n", userDir)
	fmt.Printf("  2. Enable a repo extension: geegoo skill enable %s\n", name)
	fmt.Println("  3. Future: install from COS/Git marketplace (P1)")
}

func runSkillToggle(args []string, enable bool) {
	fs := flag.NewFlagSet("skill toggle", flag.ExitOnError)
	configPath := fs.String("config", config.DefaultPath(), "path to config.json")
	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}
	if fs.NArg() == 0 {
		verb := "disable"
		if enable {
			verb = "enable"
		}
		fmt.Fprintf(os.Stderr, "geegoo skill %s: missing skill name\n", verb)
		os.Exit(2)
	}
	name := strings.TrimSpace(fs.Arg(0))
	raw, err := os.ReadFile(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "geegoo skill: read config: %v\n", err)
		os.Exit(2)
	}
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		fmt.Fprintf(os.Stderr, "geegoo skill: parse config: %v\n", err)
		os.Exit(2)
	}
	skillsBlock, _ := doc["skills"].(map[string]any)
	if skillsBlock == nil {
		skillsBlock = map[string]any{}
		doc["skills"] = skillsBlock
	}

	if enable {
		ext, _ := skillsBlock["extensions"].(map[string]any)
		if ext == nil {
			ext = map[string]any{}
			skillsBlock["extensions"] = ext
		}
		ext[name] = map[string]any{"enabled": true}
		disabled := filterStringList(skillsBlock["disabled"], name)
		if len(disabled) > 0 {
			skillsBlock["disabled"] = disabled
		} else {
			delete(skillsBlock, "disabled")
		}
	} else {
		disabled := appendStringUnique(readStringList(skillsBlock["disabled"]), name)
		skillsBlock["disabled"] = disabled
	}

	out, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "geegoo skill: marshal config: %v\n", err)
		os.Exit(2)
	}
	out = append(out, '\n')
	if err := os.WriteFile(*configPath, out, 0o600); err != nil {
		fmt.Fprintf(os.Stderr, "geegoo skill: write config: %v\n", err)
		os.Exit(2)
	}
	verb := "disabled"
	if enable {
		verb = "enabled"
	}
	fmt.Printf("skill %q %s in %s\n", name, verb, *configPath)
}

func readStringList(v any) []string {
	arr, ok := v.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(arr))
	for _, item := range arr {
		if s, ok := item.(string); ok && strings.TrimSpace(s) != "" {
			out = append(out, strings.TrimSpace(s))
		}
	}
	return out
}

func filterStringList(v any, drop string) []string {
	out := make([]string, 0)
	for _, s := range readStringList(v) {
		if s != drop {
			out = append(out, s)
		}
	}
	return out
}

func appendStringUnique(list []string, name string) []string {
	for _, s := range list {
		if s == name {
			return list
		}
	}
	return append(list, name)
}
