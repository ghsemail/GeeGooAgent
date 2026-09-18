package tools

import (
	"fmt"
	"strings"
)

func registerKnowledgeTools(r *Registry, deps Deps) {
	r.Register(Tool{
		Name:        "search_knowledge",
		Description: "在 GeeGoo 知识库（WeKnora）中检索文档片段。仅在用户要求按知识库/策略文档/查库回答时使用，不能当作行情 API。",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"query": map[string]any{
					"type":        "string",
					"description": "检索问句，如「4小时 MACD 策略入场条件」",
				},
				"folder_path": map[string]any{
					"type":        "string",
					"description": "可选，限定目录，如 策略",
				},
			},
			"required": []any{"query"},
		},
		Handle: func(ctx Context, args map[string]any) Result {
			query := strArg(args, "query", "")
			if query == "" {
				return errResult(fmt.Errorf("query is required"))
			}
			if ctx.DryRun {
				return okDryRun("search_knowledge", map[string]any{"query": query})
			}
			client := deps.WeKnora
			if client == nil || !client.Configured() {
				return errResult(fmt.Errorf("weknora is not configured"))
			}
			hits, err := client.Search(ctx.GoContext(), query, strArg(args, "folder_path", ""), 8)
			if err != nil {
				return errResult(err)
			}
			if len(hits) == 0 {
				return Result{
					Status:  StatusOK,
					Summary: "search_knowledge: no hits",
					Data:    map[string]any{"hits": []any{}, "query": query},
				}
			}
			rows := make([]any, 0, len(hits))
			var b strings.Builder
			fmt.Fprintf(&b, "search_knowledge: %d hit(s)\n", len(hits))
			for i, hit := range hits {
				rows = append(rows, map[string]any{
					"content":  hit.Content,
					"filename": hit.Filename,
					"title":    hit.Title,
					"folder":   hit.Folder,
					"score":    hit.Score,
				})
				if i < 5 {
					name := hit.Filename
					if name == "" {
						name = hit.Title
					}
					fmt.Fprintf(&b, "- %s (%s): %s\n", name, hit.Folder, shorten(hit.Content, 240))
				}
			}
			return Result{
				Status:  StatusOK,
				Summary: strings.TrimSpace(b.String()),
				Data:    map[string]any{"hits": rows, "query": query},
			}
		},
	})
	r.Register(Tool{
		Name:        "save_strategy_knowledge",
		Description: "将策略认知 Markdown 写入 WeKnora 知识库（策略认知目录）。同名文档会更新。",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"strategy_name": map[string]any{
					"type":        "string",
					"description": "策略名称，用于标题与去重",
				},
				"content": map[string]any{
					"type":        "string",
					"description": "Markdown 正文",
				},
				"folder_path": map[string]any{
					"type":        "string",
					"description": "可选，默认 策略认知",
				},
			},
			"required": []any{"strategy_name", "content"},
		},
		Handle: func(ctx Context, args map[string]any) Result {
			name := strArg(args, "strategy_name", "")
			content := strArg(args, "content", "")
			if name == "" || content == "" {
				return errResult(fmt.Errorf("strategy_name and content are required"))
			}
			if ctx.DryRun {
				return okDryRun("save_strategy_knowledge", map[string]any{
					"strategy_name": name,
					"content_len":   len(content),
				})
			}
			client := deps.WeKnora
			if client == nil || !client.Configured() {
				return errResult(fmt.Errorf("weknora is not configured"))
			}
			folder := strArg(args, "folder_path", "")
			if folder == "" {
				folder = "策略认知"
			}
			title := strategyKnowledgeTitle(name)
			doc, err := client.UpsertManualKnowledge(ctx.GoContext(), folder, title, content)
			if err != nil {
				return errResult(err)
			}
			return Result{
				Status: StatusOK,
				Summary: fmt.Sprintf("save_strategy_knowledge: %s → %s (%s)", title, doc.ID, folder),
				Data: map[string]any{
					"knowledge_id":  doc.ID,
					"title":         title,
					"folder_path":   folder,
					"parse_status":  doc.ParseStatus,
					"strategy_name": name,
				},
			}
		},
	})
}

func strategyKnowledgeTitle(strategyName string) string {
	name := strings.TrimSpace(strategyName)
	if name == "" {
		return "策略认知"
	}
	return name + " · 策略认知"
}
