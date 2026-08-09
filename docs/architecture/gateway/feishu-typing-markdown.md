# Feishu Typing + Markdown 出站 UX

**日期:** 2026-08-09  
**状态:** 已批准（卡片审批不做）

## 问题

1. Agent 处理中飞书无「思考中」反馈  
2. 回复以 `text` 发送，Markdown 不渲染  

## 方案（对齐 Hermes）

### Typing 反应

- 收到用户消息后：对该 `message_id` 加表情 `Typing`  
- 成功发出回复：删除该反应  
- Agent 失败：删除 Typing，改加 `CrossMark`  
- 缺权限只打日志，不阻断主流程  
- 权限建议：`im:message.reactions:write`（及文档中的 reactions readonly 若需）

### Markdown

- `SendText` 优先发 `msg_type=post`，正文 `tag=md`（代码围栏分段，对齐 Hermes）  
- post 被拒（content format invalid）回退纯 `text`  
- 有 `ReplyToID` 时优先 `Message.Reply`，失败再 `Create`  

### 非范围

- 交互卡片审批  
- 流式编辑正文  

## 接口

`PlatformAdapter` 可选实现：

```go
type ProcessingIndicator interface {
    MarkProcessing(ctx context.Context, messageID string) error
    ClearProcessing(ctx context.Context, messageID string) error
    MarkFailed(ctx context.Context, messageID string) error
}
```

Runner 在 `runAgentTurn` 前后调用。
