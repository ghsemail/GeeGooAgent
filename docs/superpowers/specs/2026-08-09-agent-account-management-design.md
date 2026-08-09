# Agent User 账户管理 + Gateway 只读状态

**日期:** 2026-08-09  
**状态:** 已实现（MVP）  
**范围仓库:** `trading_operation`（UI）、`GeeGooBot`（用户/mcp 列表与解析）、`GeeGooAgent`（per-user LLM + 飞书绑定身份）

## 背景

当前 User → Gateway 要求运营员手输 `mcp_token`，并按渠道在 Gateway 页选模型。飞书绑定已按登录运营用户隔离，但身份与模型配置散落在 Gateway，且与 App 开通 Agent 的真实用户脱节。

目标改为：管理员在运营台集中管理「已开通 Agent 的 App 用户」；身份 token 按用户自动解析；Gateway 仅展示各渠道状态；飞书扫码仍在 Gateway（本期不迁到账户管理）。App 端自助配置属后续。

## 已确认决策

| 项 | 选择 |
|----|------|
| 模型粒度 | 每用户 × 渠道（Web / App / 飞书）独立 |
| 飞书 App ID/Secret | 仍在 Gateway → 飞书（扫码/手动）；去掉 mcp/模型录入 |
| Dock 身份 | 账户管理选中用户后，Chat/工具自动用该用户 `mcp_token` |
| 受众 | 本期仅运营台管理员；以后再开放到 App |
| mcp_token UI | 不再手输、不展示明文；按用户名/user_id 从 Mongo `user.mcp` 读取 |

## 信息架构（User 组）

```text
User
├── 账户管理（新）
│   ├── 列表：App 已开通 Agent 的用户（mcp.enabled == true）
│   ├── 选中 → 设为「当前代操作用户」
│   └── 为该用户配置 gateways 模型：web / trading_app / feishu
└── Gateway（改）
    ├── Web / GeeGooAgent / 飞书：只读状态（相对当前代操作用户或聚合视图，见下）
    └── 飞书：保留扫码/手动凭证（绑定到当前代操作用户）
```

侧栏：在现有 `AgentNavView.gateway`（group=`User`）旁新增 `AgentNavView.accounts`（或同等命名），label 如「账户管理」。

## 核心概念

### 当前代操作用户（Acting User）

- 运营员登录态不变（`X-Client-Source=trading_operation` + 管理员 Bearer）。
- 另维护 **Acting User** = 账户管理中选中的 App 用户（`user_id` + `username`）。
- 所有需租户身份的 Agent 调用携带：
  - `X-User-Id: <acting_user_id>`
  - `X-MCP-Token: <acting_user mcp_token>`（由 BFF/客户端从服务端解析后注入；**前端不提供输入框**）
- 未选中 Acting User 时：Dock Chat / 写操作应明确禁用或提示「请先在账户管理选择用户」，避免静默落到错误身份。

### 模型

- 存储：GeeGooAgent 已有 `user_llm_settings/{userId}.json` → `gateways[source].catalog_model_id`。
- `NormalizeSessionSource` 扩展：`feishu` / `lark` → 规范键 `feishu`（与 `web`、`trading_app` 并列）。
- 账户管理 UI：三个渠道选择器，写入对应 `gateways` 键。
- 运行时：Web Dock 用 `web`；App 用 `trading_app`；飞书 IM 会话用 `feishu`（`geegoo gateway` 创建/续聊时带 source=`feishu` 并解析该用户 settings）。

### mcp_token

- 来源：Mongo `user.mcp = { enabled, mcp_token }`（GeeGooBot）。
- 「开通 Agent」判定：`mcp.enabled == true` 且 `mcp_token` 非空（若仅有 token 的 legacy，沿用现有 Normalize 推断）。
- 运营台：**禁止** SharedPreferences 手输覆盖为产品主路径；Acting User 切换时刷新内存中的 token。
- 飞书绑定文件中的 `mcp_token`：Gateway 状态刷新/绑定时，用 Acting User 的 token **回填**（延续已有 header→store 同步），保证 IM 工具身份正确。

## Gateway 只读状态

### 去掉

- `AgentUserMcpSetupCard`（Gateway 内）
- `AgentGatewayDefaultModelCard` / Web 对话界面卡若与「模型配置」强绑定，则迁出或仅保留与状态无关的展示；**模型配置入口迁至账户管理**。Web「使用/调试模式」可留在 Gateway Web tab（属 UI 偏好，非用户模型），或随 Acting User 本地 prefs——本期可保留在 Gateway Web，不写入账户模型。

### 保留 / 调整

- Web / App：展示该 Acting User（或全局列表）下会话数、连通性摘要等只读信息。
- 飞书：进程心跳、是否已连接、凭证是否就绪、是否已绑 mcp（自动）、扫码/手动绑定 UI；绑定目标 = Acting User。

### 「不同用户当前 Gateway 状态」

推荐实现（MVP）：

1. Gateway 页顶部固定显示 **当前代操作用户**；各 tab 状态均相对该用户。
2. 可选轻量：账户管理列表行内展示三渠道「已配模型 / 飞书已连」微状态（数据来自 settings + feishu status API）。

完整多用户矩阵仪表盘可二期。

## API 草案

### GeeGooBot（agent-api 或 service-api，运营台可调）

- `GET /op_agent/v1/agent-users`（或等价）  
  - 鉴权：运营 Bearer + ops client source  
  - 返回：`[{ user_id, username, mcp_enabled, has_mcp_token }]`（**不返回** token 明文给列表；token 另见下）
- `GET /op_agent/v1/agent-users/{userId}/mcp-token` 或选中时由 BFF 内部解析  
  - 仅 ops 代操作路径；响应可一次性给前端注入请求头，或 BFF 在代理时按 `X-User-Id` 自动填 token（**更优：BFF 代填，前端永不持有长期展示**）。

**推荐安全形态：** 前端只存 `acting_user_id`；BFF 在代理 agent-runtime 时若 ops bypass + 有 `X-User-Id`，则 **服务端查 Mongo 注入 `X-MCP-Token`**，前端可不传 token。若短期仍由前端传头，则 token 仅存内存、不进 prefs、不进 UI。

### GeeGooAgent

- 已有 user LLM settings GET/PUT：确保支持 `gateway=feishu`。
- 飞书 Gateway APIs：继续要求 `X-User-Id`；token 由 BFF 注入或 header 同步进 feishu store。
- `geegoo gateway` multi-tenant turn：解析 owner 的 `user_llm_settings.gateways.feishu`（若空则回退运营/全局默认，行为与现网其他渠道一致）。

### trading_operation

- 新页：账户管理列表 + 模型编辑 + 选中 Acting User。
- Gateway 去录入；Feishu 绑定用 Acting User。
- `AgentRuntimeServer` / Chat：绑定 Acting User；去掉强制手输 mcp 卡。

## 非目标（本期）

- App 内用户自助开通 / 改模型 / 绑飞书
- 飞书凭证 UI 迁到账户管理
- 多用户 Gateway 全量矩阵大屏
- 修改 Mongo 里生成/轮换 mcp_token 的产品流（仅读取已有开通用户）

## 验收标准

1. User 下可见「账户管理」与「Gateway」。
2. 账户管理列出 `mcp.enabled` 用户；选中后 Dock Chat 请求以该用户身份调用工具（可用一条需 mcp 的工具验证）。
3. 可为同一用户分别保存 Web / App / 飞书模型，并在对应入口生效（Web Dock / 飞书一轮对话至少验证飞书键被读取）。
4. Gateway 页无 mcp_token 输入、无渠道模型选择卡；飞书仍可扫码绑定到 Acting User。
5. 前端无「粘贴 mcp_token」主路径；列表 API 不泄露 token 到日志友好的 UI 字段。

## 风险与缓解

| 风险 | 缓解 |
|------|------|
| 未选 Acting User 误用管理员空身份 | 硬门禁 + 文案 |
| BFF 自动填 token 权限过大 | 仅 ops client + 已校验的运营 Bearer；只解析目标 user_id |
| 飞书模型未接线导致选了不生效 | 实现计划中单列 gateway turn 读 settings 任务 |
| 旧 prefs 里残留手输 token | 迁移：忽略 prefs token，以 Acting User 为准 |

## 实现顺序建议

1. Bot：agent-users 列表 + ops 代操作 token 解析（BFF 注入优先）
2. Agent：`feishu` gateway 键 + IM turn 用该模型
3. UI：账户管理 + Acting User
4. UI：Gateway 去录入、状态对齐 Acting User
5. 回归：Web Chat、飞书消息、工具鉴权

---

**请审阅本文档。** 确认或提出修改后，再编写 `docs/superpowers/plans/2026-08-09-agent-account-management.md` 实现计划。
