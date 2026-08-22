package tools

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/chatprompt"
	"github.com/ghsemail/GeeGooAgent/internal/memory/episodic"
	"github.com/ghsemail/GeeGooAgent/internal/memory/facts"
	"github.com/ghsemail/GeeGooAgent/internal/memory/procedural"
	"github.com/ghsemail/GeeGooAgent/internal/memory/scoped"
)

const soulLearnedRules = "## Learned rules"

var skillSlugRE = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{1,40}$`)

func registerMemoryTools(r *Registry, deps Deps) {
	if deps.Facts != nil {
		registerSaveNote(r, deps)
		registerManageMemory(r, deps)
	}
	if strings.TrimSpace(deps.Home) != "" {
		registerUpdateSoul(r, deps)
	}
	if deps.Preferences != nil || strings.TrimSpace(deps.Home) != "" {
		registerUpdatePreference(r, deps)
	}
	if strings.TrimSpace(deps.Home) != "" {
		registerCreateSkill(r, deps)
	}
}

func registerSaveNote(r *Registry, deps Deps) {
	r.Register(Tool{
		Name: "save_note",
		Description: "Save a durable fact to long-term memory. Use when the user tells you something " +
			"worth remembering about themselves, a person, or a project — especially if they say 'remember' or share a preference.",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"subject": map[string]any{"type": "string", "description": "Who/what this is about, e.g. 'alex' or 'acme-project'"},
				"content": map[string]any{"type": "string", "description": "The fact, one sentence"},
			},
			"required": []any{"subject", "content"},
		},
		Handle: func(ctx Context, args map[string]any) Result {
			subject := strings.TrimSpace(strArg(args, "subject", ""))
			content := strings.TrimSpace(strArg(args, "content", ""))
			if subject == "" || content == "" {
				return Result{Status: StatusError, Summary: "save_note: subject and content required", ExitCode: 1}
			}
			if ctx.DryRun {
				return okDryRun("save_note", map[string]any{"subject": subject, "content": content})
			}
			if err := deps.Facts.Add(ctx.GoContext(), ctx.UserID, subject, content, "user"); err != nil {
				return errResult(err)
			}
			return Result{
				Status:  StatusOK,
				Summary: fmt.Sprintf("Saved to memory under '%s': %s", subject, content),
				Data:    map[string]any{"subject": subject, "content": content},
			}
		},
	})
}

func registerManageMemory(r *Registry, deps Deps) {
	r.Register(Tool{
		Name: "manage_memory",
		Description: "Search, correct, or delete the user's long-term memory (facts and episodes). " +
			"ALWAYS search first to get the numeric id, then update or delete that id. " +
			"Use when the user says something you remember is wrong or should be forgotten.",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"action":  map[string]any{"type": "string", "enum": []any{"search", "update", "delete"}},
				"kind":    map[string]any{"type": "string", "enum": []any{"fact", "episode"}, "description": "default fact"},
				"id":      map[string]any{"type": "integer", "description": "row id (from a prior search)"},
				"query":   map[string]any{"type": "string", "description": "keywords for search"},
				"content": map[string]any{"type": "string", "description": "new text for update"},
				"subject": map[string]any{"type": "string", "description": "optional new subject for a fact update"},
			},
			"required": []any{"action"},
		},
		Handle: func(ctx Context, args map[string]any) Result {
			action := strings.ToLower(strings.TrimSpace(strArg(args, "action", "")))
			kind := strings.ToLower(strings.TrimSpace(strArg(args, "kind", "fact")))
			if kind == "" {
				kind = "fact"
			}
			if ctx.DryRun {
				return okDryRun("manage_memory", args)
			}
			switch action {
			case "search":
				return manageMemorySearch(ctx, deps, kind, strArg(args, "query", ""))
			case "update":
				if kind != "fact" {
					return Result{Status: StatusError, Summary: "Only facts can be updated (episodes are historical).", ExitCode: 1}
				}
				id := int64(intArg(args, "id", 0))
				ok, err := deps.Facts.Update(ctx.GoContext(), id, strArg(args, "content", ""), strArg(args, "subject", ""))
				if err != nil {
					return errResult(err)
				}
				if !ok {
					return Result{Status: StatusError, Summary: fmt.Sprintf("No fact with id %d.", id), ExitCode: 1}
				}
				return Result{Status: StatusOK, Summary: fmt.Sprintf("Updated fact #%d.", id)}
			case "delete":
				id := int64(intArg(args, "id", 0))
				if kind == "episode" {
					if deps.Episodic == nil {
						return Result{Status: StatusError, Summary: "episodic memory not enabled", ExitCode: 1}
					}
					if err := deps.Episodic.Delete(ctx.GoContext(), id); err != nil {
						return Result{Status: StatusError, Summary: fmt.Sprintf("No episode with id %d.", id), ExitCode: 1}
					}
					return Result{Status: StatusOK, Summary: fmt.Sprintf("Deleted episode #%d.", id)}
				}
				ok, err := deps.Facts.Delete(ctx.GoContext(), id)
				if err != nil {
					return errResult(err)
				}
				if !ok {
					return Result{Status: StatusError, Summary: fmt.Sprintf("No fact with id %d.", id), ExitCode: 1}
				}
				return Result{Status: StatusOK, Summary: fmt.Sprintf("Deleted fact #%d.", id)}
			default:
				return Result{Status: StatusError, Summary: "action must be one of: search, update, delete", ExitCode: 1}
			}
		},
	})
}

func registerUpdateSoul(r *Registry, deps Deps) {
	home := strings.TrimSpace(deps.Home)
	if home == "" {
		return
	}
	r.Register(Tool{
		Name: "update_soul",
		Description: "Save a durable rule about how you should behave for this user (their preferences and standing instructions). " +
			"Appends to your persona; takes effect next turn. Use when the user tells you how they want you to act.",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"rule": map[string]any{"type": "string", "description": "one behaviour rule, imperative"},
			},
			"required": []any{"rule"},
		},
		Handle: func(ctx Context, args map[string]any) Result {
			rule := strings.TrimSpace(strings.TrimLeft(strArg(args, "rule", ""), "-"))
			if rule == "" {
				return Result{Status: StatusOK, Summary: "Nothing to add."}
			}
			if ctx.DryRun {
				return okDryRun("update_soul", map[string]any{"rule": rule})
			}
			text := chatprompt.LoadSoulForUser(home, ctx.UserID)
			if len(text) > 8000 {
				return Result{Status: StatusError, Summary: "SOUL.md is at its size limit — edit it in the dashboard instead.", ExitCode: 1}
			}
			if !strings.Contains(text, soulLearnedRules) {
				text = strings.TrimRight(text, "\n") + "\n\n" + soulLearnedRules + "\n"
			}
			text = strings.TrimRight(text, "\n") + fmt.Sprintf("\n- %s\n", rule)
			if err := chatprompt.SaveSoulForUser(home, ctx.UserID, text); err != nil {
				return errResult(err)
			}
			return Result{Status: StatusOK, Summary: "Noted, I'll remember to: " + rule}
		},
	})
}

func registerUpdatePreference(r *Registry, deps Deps) {
	r.Register(Tool{
		Name: "update_preference",
		Description: "Save a scoped preference rule (context profile) for this user — market, stock, or automation scope. " +
			"Takes effect next turn. Use when the user gives standing instructions about a specific stock, market, or bot.",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"scope": map[string]any{
					"type":        "string",
					"description": "memory scope, e.g. user, market:HK, stock:00700.HK, automation:bot-id (bot: alias ok)",
				},
				"rule": map[string]any{"type": "string", "description": "one preference rule, imperative"},
			},
			"required": []any{"rule"},
		},
		Handle: func(ctx Context, args map[string]any) Result {
			rule := strings.TrimSpace(strings.TrimLeft(strArg(args, "rule", ""), "-"))
			if rule == "" {
				return Result{Status: StatusOK, Summary: "Nothing to add."}
			}
			scope := scoped.NormalizeScope(strArg(args, "scope", scoped.ScopeUser))
			if ctx.DryRun {
				return okDryRun("update_preference", map[string]any{"scope": scope, "rule": rule})
			}
			if deps.Preferences != nil {
				if err := deps.Preferences.AppendRule(ctx.GoContext(), ctx.UserID, scope, rule, "chat"); err != nil {
					return errResult(err)
				}
				return Result{Status: StatusOK, Summary: fmt.Sprintf("Saved preference for %s: %s", scope, rule)}
			}
			home := strings.TrimSpace(deps.Home)
			ref, ok := scoped.RefFromScope(scope)
			if !ok || home == "" {
				return Result{Status: StatusError, Summary: "scoped preferences require PostgreSQL or a valid scope ref", ExitCode: 1}
			}
			lp, loaded := chatprompt.LoadProfile(home, ctx.UserID, ref)
			text := lp.Content
			if !loaded || strings.TrimSpace(text) == "" {
				text = "- " + rule + "\n"
			} else {
				text = strings.TrimRight(text, "\n") + "\n- " + rule + "\n"
			}
			if err := chatprompt.SaveProfile(home, ctx.UserID, ref, text); err != nil {
				return errResult(err)
			}
			return Result{Status: StatusOK, Summary: fmt.Sprintf("Saved preference for %s: %s", scope, rule)}
		},
	})
}

func registerCreateSkill(r *Registry, deps Deps) {
	home := strings.TrimSpace(deps.Home)
	if home == "" {
		return
	}
	r.Register(Tool{
		Name: "create_skill",
		Description: "Write a new reusable skill (a SKILL.md the agent loads when relevant) so you can repeat a workflow the user taught you. " +
			"Only call this after the user agrees. body = step-by-step instructions; description = when to use it (include trigger words).",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"name":        map[string]any{"type": "string", "description": "short slug, e.g. weekly-review"},
				"description": map[string]any{"type": "string", "description": "one line: what it does and when to use it"},
				"body":        map[string]any{"type": "string", "description": "step-by-step instructions (markdown)"},
			},
			"required": []any{"name", "description", "body"},
		},
		Handle: func(ctx Context, args map[string]any) Result {
			name := strings.ToLower(strings.TrimSpace(strArg(args, "name", "")))
			name = strings.ReplaceAll(name, " ", "-")
			description := strings.TrimSpace(strArg(args, "description", ""))
			body := strings.TrimSpace(strArg(args, "body", ""))
			if !skillSlugRE.MatchString(name) {
				return Result{Status: StatusError, Summary: "Skill name must be a short slug like 'weekly-review' (lowercase, hyphens).", ExitCode: 1}
			}
			if description == "" || body == "" {
				return Result{Status: StatusError, Summary: "description and body required", ExitCode: 1}
			}
			if ctx.DryRun {
				return okDryRun("create_skill", map[string]any{"name": name})
			}
			userSkills := filepath.Join(home, "skills", name, "SKILL.md")
			if _, err := os.Stat(userSkills); err == nil {
				return Result{Status: StatusError, Summary: fmt.Sprintf("A skill named '%s' already exists — pick another name.", name), ExitCode: 1}
			}
			if deps.ProjectRoot != "" {
				repoSkill := filepath.Join(deps.ProjectRoot, "skills", name, "SKILL.md")
				if _, err := os.Stat(repoSkill); err == nil {
					return Result{Status: StatusError, Summary: fmt.Sprintf("A skill named '%s' already exists — pick another name.", name), ExitCode: 1}
				}
			}
			text := fmt.Sprintf("---\nname: %s\ndescription: %s\n---\n\n%s\n", name, description, body)
			if procedural.ParseSkillText(text) == nil {
				return Result{Status: StatusError, Summary: "That didn't validate — description must be present and non-trivial.", ExitCode: 1}
			}
			if err := os.MkdirAll(filepath.Dir(userSkills), 0o755); err != nil {
				return errResult(err)
			}
			if err := os.WriteFile(userSkills, []byte(text), 0o644); err != nil {
				return errResult(err)
			}
			if deps.SkillLoader != nil {
				deps.SkillLoader.Refresh()
			}
			return Result{Status: StatusOK, Summary: fmt.Sprintf("Created skill '%s'. It will trigger on: %s", name, description)}
		},
	})
}

func manageMemorySearch(ctx Context, deps Deps, kind, query string) Result {
	if kind == "episode" {
		if deps.Episodic == nil {
			return Result{Status: StatusError, Summary: "episodic memory not enabled", ExitCode: 1}
		}
		rows, err := deps.Episodic.Search(ctx.GoContext(), query, ctx.UserID, 8)
		if err != nil {
			return errResult(err)
		}
		if len(rows) == 0 {
			return Result{Status: StatusOK, Summary: "no episodes"}
		}
		var lines []string
		for _, r := range rows {
			lines = append(lines, fmt.Sprintf("#%d [%s] %s", r.ID, r.Scope, episodic.Format(r.HappenedAt, r.Summary)))
		}
		return Result{Status: StatusOK, Summary: strings.Join(lines, "\n")}
	}
	rows, err := deps.Facts.SearchRows(ctx.GoContext(), ctx.UserID, query, 8)
	if err != nil {
		return errResult(err)
	}
	if len(rows) == 0 {
		return Result{Status: StatusOK, Summary: "no matching facts"}
	}
	var lines []string
	for _, r := range rows {
		lines = append(lines, fmt.Sprintf("#%d %s", r.ID, facts.Format(r.Subject, r.Content)))
	}
	return Result{Status: StatusOK, Summary: strings.Join(lines, "\n")}
}
