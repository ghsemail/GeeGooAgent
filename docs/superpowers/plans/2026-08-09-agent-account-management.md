# Agent Account Management Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Ops console User group gains 账户管理 (list Agent-enabled App users, per-channel models, Acting User); Gateway becomes status-only with Feishu bind kept; mcp_token auto-injected by BFF from Mongo.

**Architecture:** GeeGooBot lists `mcp.enabled` users and injects `X-MCP-Token` on ops proxy when `X-User-Id` is set; GeeGooAgent normalizes `feishu` gateway key and applies that user's LLM settings on IM turns; trading_operation adds Accounts nav + Acting User state and strips Gateway MCP/model cards.

**Tech Stack:** Go (GeeGooBot agentchat + usermcp, GeeGooAgent runtimeapi/gateway), Flutter/GetX (trading_operation).

## Global Constraints

- Per-user × channel models: `web` | `trading_app` | `feishu`
- Feishu App ID/Secret stay on Gateway → 飞书
- No mcp_token input UI; list API must not return token plaintext
- Acting User required for Dock Chat; BFF injects token for ops client
- Admin-only this phase (trading_operation)

---

### Task 1: Bot — list Agent-enabled users

**Files:**
- Modify: `D:\Geegoo\GeeGooBot\internal\repo\usermcp\repository.go`
- Modify: `D:\Geegoo\GeeGooBot\internal\repo\usermcp\mongo.go`
- Modify: `D:\Geegoo\GeeGooBot\internal\repo\usermcp\fake.go`
- Create: `D:\Geegoo\GeeGooBot\internal\agentchat\agent_users.go`
- Create: `D:\Geegoo\GeeGooBot\internal\agentchat\agent_users_test.go`
- Modify: `D:\Geegoo\GeeGooBot\internal\agentchat\proxy.go` (`Register`)

**Interfaces:**
- Produces: `usermcp.Repository.ListEnabledAgentUsers(ctx) ([]AgentUserRow, error)`
- Produces: `GET /op_agent/v1/agent-users` → `{"users":[{"user_id","username","mcp_enabled","has_mcp_token"}]}`

- [ ] **Step 1: Add types + interface method**

```go
// in usermcp/repository.go
type AgentUserRow struct {
	UserID       string
	Username     string
	MCPEnabled   bool
	HasMCPToken  bool
}

ListEnabledAgentUsers(ctx context.Context) ([]AgentUserRow, error)
```

- [ ] **Step 2: Implement Mongo list** — find users where NormalizeMCPSection(mcp).Enabled; projection `_id,username,mcp`; do **not** put token in AgentUserRow.

- [ ] **Step 3: Handler + Register** — ops-only: require `X-Client-Source: trading_operation` (and existing agent-api Bearer at edge).

- [ ] **Step 4: Test** list handler returns enabled users without `mcp_token` field.

- [ ] **Step 5: Commit** `feat(agentchat): list Agent-enabled users for ops account management`

---

### Task 2: Bot — BFF inject mcp_token from X-User-Id

**Files:**
- Modify: `D:\Geegoo\GeeGooBot\internal\agentchat\mcp_auth.go`
- Modify: `D:\Geegoo\GeeGooBot\internal\agentchat\proxy_test.go` (or new `mcp_auth_inject_test.go`)

**Interfaces:**
- Consumes: `FindMCPToken(ctx, userID)`
- Behavior: when `opsConsoleBypass(r, userID)` and request token empty and route needs tenant auth, set `token = FindMCPToken(userID)` before forwarding.

- [ ] **Step 1: Failing test** — ops request with `X-User-Id` only; upstream must see `X-MCP-Token` from FakeRepository.

- [ ] **Step 2: Implement inject in `resolveMCP`** after ops bypass branch (and optionally when token empty + ValidateMCPToken for tenant routes).

- [ ] **Step 3: Tests pass; commit** `feat(agentchat): inject mcp_token for ops acting user`

---

### Task 3: Agent — NormalizeSessionSource `feishu`

**Files:**
- Modify: `D:\Geegoo\GeeGooAgent\internal\runtimeapi\tenant.go`
- Modify: `D:\Geegoo\GeeGooAgent\internal\runtimeapi\tenant_test.go` (create if missing)

- [ ] **Step 1: Test** `NormalizeSessionSource("feishu") == "feishu"`, `"lark" == "feishu"`.

- [ ] **Step 2: Implement switch cases; commit** `feat(runtime): normalize feishu session source`

---

### Task 4: Agent — Feishu IM turn uses user `gateways.feishu` model

**Files:**
- Modify: `D:\Geegoo\GeeGooAgent\internal\gateway\multitenant\handle.go`
- Modify: `D:\Geegoo\GeeGooAgent\internal\runtimeapi\user_llm_settings.go` (export helper if needed)
- Test: unit test with stub settings load or extract `ApplyGatewayLLM(cfg, settings, "feishu")`

**Interfaces:**
- Consumes: `loadUserLLMSettings` / mergeInto for gateway key `feishu`
- Before `Agent.Run`, apply owner user’s feishu gateway LLM like `withUserAgentGateway` (session metadata `source=feishu`).

- [ ] **Step 1: Extract or reuse** apply-user-gateway-LLM for non-HTTP callers (gateway process has `*app.App`).

- [ ] **Step 2: In `runOwnedTurn`, load settings for `own.OwnerUserID`, merge `feishu`, rebuild/temp gateway or set catalog id for the turn; set `chat.Metadata["source"]="feishu"`.

- [ ] **Step 3: Test + commit `feat(gateway): apply per-user feishu LLM settings on IM turns`

---

### Task 5: Flutter — Acting User store + agent-users API

**Files:**
- Create: `D:\Geegoo\trading_operation\lib\modules\agent_mode\user\acting_user.dart`
- Modify: `D:\Geegoo\trading_operation\lib\api\agent_runtime_server.dart`
- Modify: `D:\Geegoo\trading_operation\lib\api\agent_session_context.dart` (or chat controller)

**Interfaces:**
- Produces: `ActingUserController` / prefs keys `acting_user_id`, `acting_username` (no token in prefs)
- Produces: `getAgentUsers()` → list DTO
- `AgentRuntimeServer._headers`: `X-User-Id` = acting user; **omit** client mcp_token (BFF injects)

- [ ] **Step 1: API + ActingUser persistence**
- [ ] **Step 2: Wire chat/runtime to Acting User; gate chat if empty**
- [ ] **Step 3: Commit** `feat(agent-mode): Acting User state and agent-users client`

---

### Task 6: Flutter — 账户管理 page

**Files:**
- Modify: `D:\Geegoo\trading_operation\lib\modules\agent_mode\theme\waku_theme.dart` (`AgentNavView.accounts`)
- Modify: `D:\Geegoo\trading_operation\lib\modules\agent_mode\shell\agent_nav.dart`, `agent_shell.dart`, `agent_mode_controller.dart`
- Create: `D:\Geegoo\trading_operation\lib\modules\agent_mode\user\views\agent_accounts_view.dart`
- Reuse model pickers from `agent_gateway_default_model_card.dart` pattern for three gateways targeting Acting User / selected row user

- [ ] **Step 1: Nav entry User → 账户管理**
- [ ] **Step 2: List users; select → set Acting User; configure web/trading_app/feishu models via existing settings API with that userId**
- [ ] **Step 3: Commit** `feat(agent-mode): accounts management UI`

---

### Task 7: Flutter — Gateway status-only

**Files:**
- Modify: `D:\Geegoo\trading_operation\lib\modules\agent_mode\user\views\agent_gateway_view.dart`
- Modify: `D:\Geegoo\trading_operation\lib\modules\agent_mode\user\widgets\agent_feishu_channel_panel.dart`

- [ ] **Step 1: Remove MCP card and DefaultModel cards from Gateway**
- [ ] **Step 2: Banner showing Acting User; Feishu bind uses Acting User headers**
- [ ] **Step 3: Keep Web chat mode card optional; commit** `feat(agent-mode): Gateway status-only for Acting User`

---

### Task 8: Deploy + smoke

- [ ] Push GeeGooBot, GeeGooAgent, trading_operation; `--git-deploy` bot agent-api; Agent install.sh + gateway restart; web static deploy
- [ ] Verify: list users; select user; chat without pasting token; save feishu model; Gateway has no mcp input

---

## Spec coverage

| Spec item | Task |
|-----------|------|
| 账户管理列表 mcp.enabled | 1, 6 |
| 每用户×渠道模型 | 3, 4, 6 |
| mcp 自动 / 不手输 | 2, 5, 7 |
| Gateway 只读 + 飞书绑定 | 7 |
| Acting User → Dock | 5, 6 |
| feishu IM 用模型 | 4 |

## Execution

User requested **开始** → **Inline Execution** in this session (executing-plans style), starting Task 1.
