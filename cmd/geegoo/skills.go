package main

import (
	"flag"
	"fmt"
	"os"
	"sort"

	"github.com/ghsemail/GeeGooAgent/internal/app"
	"github.com/ghsemail/GeeGooAgent/internal/config"
	"github.com/ghsemail/GeeGooAgent/internal/skills"
)

func runSkillsList(args []string) {
	fs := flag.NewFlagSet("skills list", flag.ExitOnError)
	configPath := fs.String("config", config.DefaultPath(), "path to config.json")
	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}

	fmt.Println("[Workflow Skills — geegoo run]")
	registry := skills.Default()
	specs := registry.List()
	if len(specs) == 0 {
		fmt.Println("  (none)")
	} else {
		fmt.Printf("  %-14s  %s\n", "SKILL", "DESCRIPTION")
		for _, s := range specs {
			fmt.Printf("  %-14s  %s\n", s.Name, s.Description)
		}
	}

	fmt.Println()
	fmt.Println("[Procedural Skills — SKILL.md]")
	application, err := app.LoadFromConfigPath(*configPath, false)
	if err != nil {
		fmt.Fprintf(os.Stderr, "skills list: load config: %v\n", err)
		os.Exit(2)
	}
	defer func() { _ = application.Close() }()

	if application.SkillLoader == nil {
		fmt.Println("  (loader not configured)")
		return
	}
	summaries := application.SkillLoader.ListSummaries()
	if len(summaries) == 0 {
		fmt.Println("  (none)")
		return
	}
	sort.Slice(summaries, func(i, j int) bool {
		if summaries[i].Kind != summaries[j].Kind {
			return summaries[i].Kind < summaries[j].Kind
		}
		return summaries[i].Name < summaries[j].Name
	})
	fmt.Printf("  %-22s %-10s %-8s %-4s  %s\n", "NAME", "KIND", "SOURCE", "CHAT", "STATUS")
	for _, s := range summaries {
		chat := " "
		if s.InjectInChat {
			chat = "*"
		}
		status := "on"
		if !s.Enabled {
			status = "off"
		}
		fmt.Printf("  %-22s %-10s %-8s %-4s  %s\n", s.Name, s.Kind, s.Provenance, chat, status)
	}
}

func runSkills(args []string) {
	if len(args) == 0 {
		runSkillsList(nil)
		return
	}
	switch args[0] {
	case "list":
		runSkillsList(args[1:])
	default:
		fmt.Fprintf(os.Stderr, "geegoo skills: unknown subcommand %q (try: list)\n", args[0])
		os.Exit(2)
	}
}
