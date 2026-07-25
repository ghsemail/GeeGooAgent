package tools

import (
	"fmt"
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/chatprompt"
	"github.com/ghsemail/GeeGooAgent/internal/memory/facts"
)

const soulLearnedRules = "## Learned rules"

func registerMemoryTools(r *Registry, deps Deps) {
	if deps.Facts == nil {
		return
	}
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
			text := chatprompt.LoadSoulFromHome(home)
			if len(text) > 8000 {
				return Result{Status: StatusError, Summary: "SOUL.md is at its size limit — edit it in the dashboard instead.", ExitCode: 1}
			}
			if !strings.Contains(text, soulLearnedRules) {
				text = strings.TrimRight(text, "\n") + "\n\n" + soulLearnedRules + "\n"
			}
			text = strings.TrimRight(text, "\n") + fmt.Sprintf("\n- %s\n", rule)
			if err := chatprompt.SaveSoulToHome(home, text); err != nil {
				return errResult(err)
			}
			return Result{Status: StatusOK, Summary: "Noted, I'll remember to: " + rule}
		},
	})
}

func manageMemorySearch(ctx Context, deps Deps, kind, query string) Result {
	if kind == "episode" {
		if deps.Episodic == nil {
			return Result{Status: StatusError, Summary: "episodic memory not enabled", ExitCode: 1}
		}
		rows, err := deps.Episodic.List(ctx.GoContext(), ctx.UserID, 20)
		if err != nil {
			return errResult(err)
		}
		q := strings.ToLower(strings.TrimSpace(query))
		var lines []string
		for _, r := range rows {
			if q != "" && !strings.Contains(strings.ToLower(r.Summary), q) {
				continue
			}
			lines = append(lines, fmt.Sprintf("#%d (%s) %s", r.ID, r.HappenedAt.Format("2006-01-02"), r.Summary))
			if len(lines) >= 8 {
				break
			}
		}
		if len(lines) == 0 {
			return Result{Status: StatusOK, Summary: "no episodes"}
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
