# IM Gateway M2+ backlog

M1（飞书 WebSocket 文本双向）已在本仓库落地。后续按里程碑拆 PR：

| 里程碑 | 内容 | 预留接口 |
|--------|------|----------|
| M2 | Markdown→post、媒体进出、home channel、`Job.Platform=feishu` | `SendMedia`、`Runner.DeliverToHome`、`NotifySchedulerResult` |
| M3 | 交互卡片（审批）、反应、按群 ACL、Webhook 模式 | adapter 事件扩展 |
| M4 | 文档评论、会议邀请 | `feishu_comment` 类模块 |

配置项已预留：`FEISHU_HOME_CHANNEL`、`FEISHU_CONNECTION_MODE=webhook`（M1 拒绝非 websocket）。
