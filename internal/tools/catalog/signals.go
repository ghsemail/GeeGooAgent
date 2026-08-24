package catalog

// IndexSignalHTTPSpecs are single-indicator signal catalog tools (platform signal_index_db).
func IndexSignalHTTPSpecs() []HTTPSpec {
	return []HTTPSpec{
		{
			Name:           "get_index_signal",
			Description:    "获取全部单指标信号（完整字段，含 i18n 文档）。",
			Path:           "/getIndexSignal",
			DirectResponse: true,
		},
		{
			Name:           "get_all_index_signal_id",
			Description:    "返回全部单指标信号 signal_id 列表。",
			Path:           "/getAllIndexSignalId",
			DirectResponse: true,
		},
		{
			Name:           "add_index_signal",
			Description:    "新增单指标信号（name/frequency/index/brief/info 等）。写操作需用户确认。",
			Path:           "/addIndexSignal",
			DirectResponse: true,
			MergePayload:   true,
		},
		{
			Name:           "edit_index_signal",
			Description:    "更新单指标信号（signal_id 必填）。写操作需用户确认。",
			Path:           "/editIndexSignal",
			DirectResponse: true,
			MergePayload:   true,
			Parameters: map[string]any{
				"type": "object", "required": []string{"signal_id"},
				"properties": map[string]any{
					"signal_id": stringProp("信号 MongoDB ObjectId"),
				},
			},
		},
		{
			Name:           "delete_index_signal",
			Description:    "删除单指标信号。写操作需用户确认。",
			Path:           "/deleteIndexSignal",
			DirectResponse: true,
			MergePayload:   true,
			Parameters: map[string]any{
				"type": "object", "required": []string{"signal_id"},
				"properties": map[string]any{
					"signal_id": stringProp("信号 MongoDB ObjectId"),
				},
			},
		},
	}
}

// CombinationSignalHTTPSpecs are combination signal catalog tools (signal_combination_db).
func CombinationSignalHTTPSpecs() []HTTPSpec {
	return []HTTPSpec{
		{
			Name:           "get_signal_combination",
			Description:    "获取全部组合信号（完整 buy_signal/sell_signal 等字段）。",
			Path:           "/getSignalCombination",
			DirectResponse: true,
		},
		{
			Name:           "get_agent_signal_combination",
			Description:    "获取组合信号 Agent 摘要列表（signal_id、name、brief、info）。",
			Path:           "/getAgentSignalCombination",
			DirectResponse: true,
		},
		{
			Name:           "add_combination_signal",
			Description:    "新增组合信号。写操作需用户确认。",
			Path:           "/addCombinationSignal",
			DirectResponse: true,
			MergePayload:   true,
		},
		{
			Name:           "edit_combination_signal",
			Description:    "更新组合信号（signal_id 必填）。写操作需用户确认。",
			Path:           "/editCombinationSignal",
			DirectResponse: true,
			MergePayload:   true,
			Parameters: map[string]any{
				"type": "object", "required": []string{"signal_id"},
				"properties": map[string]any{
					"signal_id": stringProp("信号 MongoDB ObjectId"),
				},
			},
		},
		{
			Name:           "delete_combination_signal",
			Description:    "删除组合信号。写操作需用户确认。",
			Path:           "/deleteCombinationSignal",
			DirectResponse: true,
			MergePayload:   true,
			Parameters: map[string]any{
				"type": "object", "required": []string{"signal_id"},
				"properties": map[string]any{
					"signal_id": stringProp("信号 MongoDB ObjectId"),
				},
			},
		},
	}
}

// UsesBotMCPProxy reports tools that must call GeeGooBot mcp-api (:3120) so mcp_token
// is resolved to user_id before catalog-api (:3210).
func UsesBotMCPProxy(name string) bool {
	switch name {
	case "get_index_signals", "get_signal_combinations",
		"get_index_signal", "get_all_index_signal_id",
		"add_index_signal", "edit_index_signal", "delete_index_signal",
		"get_signal_combination", "get_agent_signal_combination",
		"add_combination_signal", "edit_combination_signal", "delete_combination_signal",
		"get_custom_signal", "get_custom_signal_for_skill", "get_all_custom_signal_id",
		"add_custom_signal", "edit_custom_signal", "delete_custom_signal":
		return true
	default:
		return false
	}
}
