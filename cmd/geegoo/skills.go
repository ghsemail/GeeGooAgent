package main

import (
	"fmt"
	"os"

	"github.com/ghsemail/GeeGooAgent/internal/app"
	"github.com/ghsemail/GeeGooAgent/internal/skills"
)

func runSkillsList(args []string) {
	_ = args
	registry := skills.Default()
	specs := registry.List()
	if len(specs) == 0 {
		fmt.Println("(no skills registered)")
		return
	}
	fmt.Printf("%-12s  %s\n", "SKILL", "DESCRIPTION")
	for _, s := range specs {
		fmt.Printf("%-12s  %s\n", s.Name, s.Description)
	}
}

// ensureAppSkillsLoaded is a no-op guard; the registry is package-level in app.
var _ = app.DefaultSkills

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
