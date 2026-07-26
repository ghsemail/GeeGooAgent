package catalog

// MondayHTTPSpecs returns SKILLServer-compatible tools on GeeGooSignal catalog-api :3210.
func MondayHTTPSpecs() []HTTPSpec {
	return []HTTPSpec{
		{
			Name:             "get_single_prompt_template_by_index",
			Description:      "按指标 variable 与 period 查询已启用的单项分析 Prompt 模板（返回单条或空对象）。",
			Path:             "/getSinglePromptTemplateByIndex",
			RequiresMCPToken: true,
			DirectResponse:   true,
			MergePayload:     true,
			Parameters: map[string]any{
				"type": "object",
				"required": []string{"index", "period"},
				"properties": map[string]any{
					"index":  stringProp("指标 variable，如 EMA"),
					"period": stringProp("周期，如 daily"),
					"type":   stringProp("可选类型过滤：index / industry / etf 等"),
				},
			},
		},
		{
			Name:             "add_single_prompt_template",
			Description:      "新增单项分析 Prompt 模板（Monday/运营）。写操作需用户确认。",
			Path:             "/addSinglePromptTemplate",
			RequiresMCPToken: true,
			DirectResponse:   true,
			MergePayload:     true,
		},
		{
			Name:             "edit_prompt_template",
			Description:      "按 id 编辑单项分析 Prompt 模板。写操作需用户确认。",
			Path:             "/editPromptTemplate",
			RequiresMCPToken: true,
			DirectResponse:   true,
			MergePayload:     true,
		},
		{
			Name:             "delete_prompt_template",
			Description:      "按 id 删除单项分析 Prompt 模板。写操作需用户确认。",
			Path:             "/deletePromptTemplate",
			RequiresMCPToken: true,
			DirectResponse:   true,
			MergePayload:     true,
			Parameters: map[string]any{
				"type": "object", "required": []string{"id"},
				"properties": map[string]any{"id": stringProp("模板 MongoDB ObjectId")},
			},
		},
		{
			Name:             "switch_prompt_status",
			Description:      "切换 Prompt 模板启用状态。写操作需用户确认。",
			Path:             "/switchPromptStatus",
			RequiresMCPToken: true,
			DirectResponse:   true,
			MergePayload:     true,
			Parameters: map[string]any{
				"type": "object", "required": []string{"id", "switch"},
				"properties": map[string]any{
					"id":     stringProp("模板 ID"),
					"switch": map[string]any{"type": "boolean", "description": "true 启用 / false 禁用"},
				},
			},
		},
		{
			Name:             "get_custom_signal",
			Description:      "获取全部定制策略（完整 i18n 文档 + supported_frequencies）。",
			Path:             "/getCustomSignal",
			RequiresMCPToken: true,
			DirectResponse:   true,
		},
		{
			Name:             "get_custom_signal_for_skill",
			Description:      "获取定制策略 Skill 友好列表（中文 name/brief/info + custom + supported_frequencies）。",
			Path:             "/getCustomSignalForSkill",
			RequiresMCPToken: true,
			DirectResponse:   true,
		},
		{
			Name:             "get_all_custom_signal_id",
			Description:      "返回全部定制策略 signal_id 列表。",
			Path:             "/getAllCustomSignalId",
			RequiresMCPToken: true,
			DirectResponse:   true,
		},
		{
			Name:             "get_custom_strategy_definitions",
			Description:      "查询代码注册的定制策略定义（strategy_key、defaults、param_schema、supported_frequencies）。新增/编辑前应先调用。",
			Path:             "/getCustomStrategyDefinitions",
			RequiresMCPToken: true,
			DirectResponse:   true,
		},
		{
			Name:             "add_custom_signal",
			Description:      "新增定制策略（custom.index 须在注册表中）。写操作需用户确认。",
			Path:             "/addCustomSignal",
			RequiresMCPToken: true,
			DirectResponse:   true,
			MergePayload:     true,
		},
		{
			Name:             "edit_custom_signal",
			Description:      "更新定制策略。写操作需用户确认。",
			Path:             "/editCustomSignal",
			RequiresMCPToken: true,
			DirectResponse:   true,
			MergePayload:     true,
		},
		{
			Name:             "delete_custom_signal",
			Description:      "删除定制策略。写操作需用户确认。",
			Path:             "/deleteCustomSignal",
			RequiresMCPToken: true,
			DirectResponse:   true,
			MergePayload:     true,
			Parameters: map[string]any{
				"type": "object", "required": []string{"signal_id"},
				"properties": map[string]any{"signal_id": stringProp("策略 MongoDB ObjectId")},
			},
		},
	}
}

// CatalogPromptTemplatePaths are competitor/etf template CRUD on catalog-api.
var CatalogPromptTemplatePaths = map[string]struct{}{
	"create_competitor_prompt_template": {},
	"edit_competitor_prompt_template":   {},
	"delete_competitor_prompt_template": {},
	"create_etf_prompt_template":        {},
	"edit_etf_prompt_template":          {},
	"delete_etf_prompt_template":          {},
}

// MondayToolNames lists all Monday SKILLServer tools including bespoke get_single_prompt_template.
func MondayToolNames() []string {
	names := []string{"get_single_prompt_template"}
	for _, spec := range MondayHTTPSpecs() {
		names = append(names, spec.Name)
	}
	return names
}

// UsesSignalCatalog reports whether a tool should call GeeGooSignal catalog-api :3210.
func UsesSignalCatalog(name string) bool {
	if _, ok := CatalogPromptTemplatePaths[name]; ok {
		return true
	}
	switch name {
	case "get_single_prompt_template", "get_index_signals", "get_signal_combinations":
		return true
	}
	for _, spec := range MondayHTTPSpecs() {
		if spec.Name == name {
			return true
		}
	}
	return false
}
