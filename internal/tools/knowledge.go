package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/clients/weknora"
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
					"description": "可选，限定目录：策略资料（参考文档）、策略档案（Agent 档案）；兼容旧名 策略 / 策略认知",
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
			hits, err := searchKnowledgeMerged(ctx.GoContext(), client, query, strArg(args, "folder_path", ""), 8)
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
		Description: "将策略档案 Markdown 写入 WeKnora 知识库（策略档案目录）。同名文档会更新。",
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
					"description": "可选，默认 策略档案",
				},
				"title": map[string]any{
					"type":        "string",
					"description": "可选，文档标题；默认 strategy_name · 策略档案",
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
			folder := NormalizeKnowledgeFolderForWrite(strArg(args, "folder_path", ""))
			if folder == "" {
				folder = StrategyArchiveFolder
			}
			title := strings.TrimSpace(strArg(args, "title", ""))
			if title == "" {
				title = strategyKnowledgeTitle(name)
			}
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
		return StrategyArchiveFolder
	}
	return name + " · " + StrategyArchiveFolder
}

func searchKnowledgeMerged(ctx context.Context, client *weknora.Client, query, folder string, limit int) ([]weknora.SearchHit, error) {
	if client == nil {
		return nil, fmt.Errorf("weknora client is nil")
	}
	folders := ExpandKnowledgeFolderFilter(folder)
	seen := make(map[string]struct{})
	out := make([]weknora.SearchHit, 0, limit)
	for _, fp := range folders {
		hits, err := client.Search(ctx, query, fp, limit)
		if err != nil {
			return nil, err
		}
		for _, hit := range hits {
			key := hit.Filename + "\x00" + hit.Title + "\x00" + shorten(hit.Content, 120)
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			out = append(out, hit)
			if len(out) >= limit {
				return out, nil
			}
		}
	}
	return out, nil
}
