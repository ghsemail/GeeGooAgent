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
}
