# IM Gateway M1 真机验收清单（飞书）

1. 运行 `geegoo gateway setup`，用飞书手机端扫描终端二维码，自动创建机器人并写入 `~/.geegoo/.env`（或 `--manual` 手填凭证）。
2. 在飞书开放平台确认应用已开**机器人**；权限含 `im:message`、`im:message:send_as_bot`、`im:chat`；**事件订阅**为**长连接**并订阅 `im.message.receive_v1`；发布版本。
3. 运行 `geegoo gateway run`，日志出现 Feishu WebSocket starting / gateway connecting。
4. 私聊机器人发送一句话 → 收到 Agent 文本回复。
5. 将机器人拉入群：未 @ 不回复；@ 机器人后回复。
6. 用不在白名单的账号私聊 → 无回复（debug 日志可见 discarded）。
7. `go test ./internal/gateway/... ./internal/config/ -run 'Onboard|UpsertEnv|Normalize|Session|Dedup|Authorize'` 本地通过。
