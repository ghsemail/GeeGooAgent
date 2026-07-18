package chatprompt

// ServiceEndpoints returns outbound service hints for the model.
func ServiceEndpoints() string {
	return `出站服务：GeeGooBot mcp-api :3120（Tool 主路径）；GeeGooSignal catalog :3210 / analyze :3230；GeeGooData :3300（可选直读）。`
}
