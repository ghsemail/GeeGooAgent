package catalog

func reportCRUD(slug, labelZh, create, update, delete, list string, includeCreate bool) []HTTPSpec {
	var specs []HTTPSpec
	if includeCreate {
		specs = append(specs, HTTPSpec{
			Name:         "create_" + slug + "_report",
			Description:  "创建" + labelZh + "（report_workflow 🔒）。必填 stock_name、code 等；写操作需用户确认。",
			Path:         create,
			MergePayload: true,
			Parameters:   reportWriteParameters(true),
		})
	}
	specs = append(specs,
		HTTPSpec{
			Name:         "update_" + slug + "_report",
			Description:  "更新" + labelZh + "。必填 report_id。",
			Path:         update,
			MergePayload: true,
			Parameters:   reportUpdateParameters(),
		},
		HTTPSpec{
			Name:        "delete_" + slug + "_report",
			Description: "删除" + labelZh + "。必填 report_id。",
			Path:        delete,
			Parameters:  reportDeleteParameters(),
		},
		HTTPSpec{
			Name:        "get_" + slug + "_reports",
			Description: "查询已存" + labelZh + "。可选 code、report_date(YYYY-MM-DD)。",
			Path:        list,
			Parameters:  reportQueryParameters(),
		},
	)
	return specs
}

func reportWriteParameters(requireStock bool) map[string]any {
	required := []string{"code", "stock_name"}
	if !requireStock {
		required = []string{"code"}
	}
	return map[string]any{
		"type":     "object",
		"required": required,
		"properties": map[string]any{
			"code":        stringProp("标的代码"),
			"stock_name":  stringProp("标的名称"),
			"report_date": stringProp("报告日期 YYYY-MM-DD，默认今天"),
			"content":     stringProp("报告正文/markdown"),
		},
	}
}

func reportUpdateParameters() map[string]any {
	return map[string]any{
		"type":     "object",
		"required": []string{"report_id"},
		"properties": map[string]any{
			"report_id": stringProp("报告 _id"),
			"content":   stringProp("更新内容"),
		},
	}
}

func reportDeleteParameters() map[string]any {
	return map[string]any{
		"type":     "object",
		"required": []string{"report_id"},
		"properties": map[string]any{
			"report_id": stringProp("报告 _id"),
		},
	}
}

func reportQueryParameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"code":        stringProp("标的代码"),
			"report_date": stringProp("YYYY-MM-DD"),
		},
	}
}

func codeQueryParameters(desc string) map[string]any {
	return map[string]any{
		"type":     "object",
		"required": []string{"code"},
		"properties": map[string]any{
			"code": stringProp(desc),
		},
	}
}
