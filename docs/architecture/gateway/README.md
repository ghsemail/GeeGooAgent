# IM Gateway

> **定位**：与 Hermes `gateway/` 同构的即时通讯入口。平台差异在此层；内核仍走 `Agent.Run`。  
> **与 L1 Model Gateway 区分**：[`internal/llm.Gateway`](../../internal/llm/gateway.go) 是模型通道；本目录对应 [`internal/gateway`](../../internal/gateway)。

## 入口

```bash
geegoo gateway setup [--domain feishu|lark]   # 扫码创建飞书机器人（推荐）或 --manual
geegoo gateway run [--config PATH] [--dry-run]
geegoo gateway status [--config PATH]
```

`setup` 默认走飞书 **device-code 扫码建应用**（对齐 Hermes `qr_register`）：终端打印 ASCII 二维码 → 手机飞书扫码 → 自动拿到 App ID/Secret → 写入 `~/.geegoo/.env`。

Dashboard（trading_operation Agent 模式 → Gateway → **飞书** tab）走同一套 API：

- `GET /v1/gateway/feishu/status`
- `POST /v1/gateway/feishu/setup/begin`（返回 `qr_png_base64`）
- `POST /v1/gateway/feishu/setup/poll`
- `POST /v1/gateway/feishu/setup/manual`

## 架构

```text
飞书/Lark (WS)
    → platforms/feishu.Adapter
    → gateway.Runner（鉴权 / 去重 / 按 chat 串行 / 会话映射）
    → Agent.Run
    → Adapter.SendText
```

## 飞书（第一平台）

| 里程碑 | 状态 | 范围 |
|--------|------|------|
| **M1** | ✅ | WebSocket；私聊/群 @；纯文本；白名单；Session→Agent→回飞书 |
| **M2** | 待做 | Markdown post、媒体、home channel、Scheduler `platform=feishu` |
| **M3** | 待做 | 交互卡片、反应、按群 ACL、Webhook |
| **M4** | 待做 | 文档评论、会议邀请等 |

### 环境变量（M1）

| 变量 | 必填 | 说明 |
|------|------|------|
| `FEISHU_APP_ID` | ✅ | 飞书/Lark App ID |
| `FEISHU_APP_SECRET` | ✅ | App Secret |
| `FEISHU_DOMAIN` | | `feishu`（默认）或 `lark` |
| `FEISHU_CONNECTION_MODE` | | `websocket`（M1 仅支持） |
| `FEISHU_ALLOWED_USERS` | 强烈建议 | 逗号分隔 open_id；空=放行全部（启动打警告） |
| `FEISHU_ALLOW_ALL_USERS` | | `true` 时明确允许全部用户 |
| `FEISHU_HOME_CHANNEL` | | cron/通知目标 chat_id（M2 使用） |
| `FEISHU_GROUP_POLICY` | | `allowlist`（默认）/ `open` / `disabled` |
| `FEISHU_REQUIRE_MENTION` | | 群内是否必须 @（默认 `true`） |

飞书开放平台：开机器人能力；权限 `im:message`、`im:message:send_as_bot`、`im:chat`，建议 `im:message.reactions:write`（处理中 Typing 反应）；事件订阅选**长连接**并订阅 `im.message.receive_v1`；发版生效。

出站：优先 `post`+`md` 渲染 Markdown（失败回退 `text`）；处理中对用户消息加 Typing 反应，成功后清除，失败改 CrossMark。工具调用默认发一条可编辑的「处理中」进度气泡（`FEISHU_TOOL_PROGRESS=0` 可关）。

**多租户**：凭证按运营台 `user_id` 存于 `{outputDir}/user_gateway_feishu/`；绑定该用户 `mcp_token`；`geegoo gateway run` 为每个已配置用户开一条 WS。详见 `feishu-per-user.md`。

### M1 验收

见 [acceptance-m1.md](./acceptance-m1.md)。后续里程碑见 [m2-plus.md](./m2-plus.md)。

## 会话键

`{platform}:{chat_id}:u:{user_id}` → 映射到 `chatsession` 中的会话 ID（持久化于 workspace `gateway/sessions.json`）。群内按用户隔离（对齐 Hermes `group_sessions_per_user`）。
