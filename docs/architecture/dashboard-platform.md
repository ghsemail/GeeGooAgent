# GeeGoo Agent 配套 Web 平台（SSOT）

> **定稿**：产品 UI 在 `trading_operation`；`GeeGooAgent` 提供 Runtime HTTP API 与数据存储。  
> 交互模式对标 [waku-agent](https://github.com/ShenSeanChen/waku-agent) Dashboard；分工对标 ADP Op Agent（Chat 浮标 + 管理端 Cockpit）。

## 1. 三仓分工

| 仓库 | 职责 | 端口 |
|------|------|------|
| **trading_operation** | Flutter Web 运营门户：Chat 对话 + Agent Cockpit | Nginx 静态 |
| **GeeGooAgent** | agent-runtime：ReAct、SSE、会话、Cockpit 读 API | `:3400` |
| **GeeGooBot**（可选 BFF） | agent-api：鉴权、CORS、MCP Token 注入 | `:3110` |

```text
trading_operation
  ├── lib/modules/geegoo_agent_chat/     # 日常对话（后续可加全站浮标）
  └── lib/modules/geegoo_agent_mgt/      # Waku 风格 Cockpit
              │
              ▼  HTTPS（Bearer + X-MCP-Token）
GeeGooAgent agent-runtime :3400
              │
              ▼
GeeGooBot mcp-api :3120  ·  GeeGooData :3300  ·  GeeGooSignal :3210
```

**原则**

- 前端不承载 Agent 逻辑；只消费 `runtimeapi`。
- Runtime 不对公网裸暴露；生产经 Nginx / BFF。
- CLI `geegoo chat` 仅调试，非产品入口。

## 2. 双入口 UI

| 入口 | 模块 | 用户 | 功能 |
|------|------|------|------|
| **Agent 对话** | `geegoo_agent_chat` | 运营 | 多轮 Chat、SSE trace、Plan/Clarify 卡片（二期） |
| **Agent 管理** | `geegoo_agent_mgt` | 运维 | Overview / Loop / Tools / Doctor / Memory |

### Cockpit Tab ↔ Waku 映射

| Tab | Waku | GeeGoo 数据源 |
|-----|------|---------------|
| Overview | 成本、gate、汇总 | `GET /v1/metrics/overview` |
| Loop | turn timeline | `GET /v1/sessions`、`GET /v1/sessions/{id}/trace` |
| Tools | 注册表 | `GET /v1/tools` |
| Doctor | 健康检查 | `GET /v1/doctor` |
| Memory | 三支柱 | `GET /v1/memory/status`（向量库上线后扩展） |

## 3. HTTP API 契约

鉴权：`Authorization: Bearer <GEEGOO_AGENT_RUNTIME_API_KEY>`  
用户 Tool 调用：`X-MCP-Token: <user_mcp_token>`  
写操作跳过审批（慎用）：`X-Approve-Writes: true`

### 3.1 Chat（已有）

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/v1/chat/stream` | SSE：tool 进度 → `turn_end` → `done` |
| POST | `/v1/chat/clarify` | 澄清续跑 |
| POST | `/v1/chat/plan` | Plan 审批 |
| GET | `/v1/sessions/status` | 会话快照 |
| GET | `/v1/sessions/events/stream` | 会话事件 SSE |

请求体（stream）：

```json
{ "message": "查腾讯股价", "session_id": "", "mcp_token": "" }
```

### 3.2 Cockpit（本阶段新增）

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/v1/metrics/overview` | runs/sessions/tools 汇总 |
| GET | `/v1/sessions` | 会话列表（`?limit=50`） |
| GET | `/v1/sessions/{id}/trace` | 单会话 step_records 时间线 |
| GET | `/v1/tools` | 已注册 Tool 列表 |
| GET | `/v1/doctor` | 结构化健康检查 |
| GET | `/v1/memory/status` | 记忆后端状态（SQLite / 向量库） |

### 3.3 SSE 事件（`/v1/chat/stream`）

| event | 含义 |
|-------|------|
| `connected` | 会话已建立 |
| `tool_start` / `tool_end` | Tool 执行 |
| `text_delta` | 流式文本（若启用） |
| `turn_end` | 本轮结束（含 `plan_pending`） |
| `done` | 流关闭 |

## 4. 数据层（GeeGooAgent 服务器）

| 存储 | 阶段 | 用途 |
|------|------|------|
| **SQLite**（`GEEGOO_DB`） | 现网 / MVP | `chatsession` SSOT、FTS |
| **PostgreSQL** | Phase 2 | 多用户 sessions、runs、approvals、metrics |
| **pgvector / Qdrant** | Phase 3 | `memport` 语义记忆 |
| **文件** `~/.geegoo/` | 始终 | traces、execution-log、checkpoints |

环境变量：

```bash
GEEGOO_AGENT_RUNTIME_PORT=3400
GEEGOO_AGENT_RUNTIME_API_KEY=...
GEEGOO_BOT_MCP_URL=http://127.0.0.1:3120
GEEGOO_DB=on   # SQLite（默认）
# 二期
GEEGOO_PG_DSN=postgres://...
GEEGOO_VECTOR_URL=http://127.0.0.1:6333
```

## 5. trading_operation 集成

### 5.1 配置

`lib/api/server_url.dart`（生产推荐 BFF）：

```dart
const bool agent_use_bff = true;
const String agent_bff_ip = bot_server_ip;
const String agent_bff_port = '3110';
const String agent_api_key = '<GEEGOO_BOT_AGENT_API_KEY>';
```

内网调试可 `agent_use_bff = false` 直连 `:3400`。

`lib/api/agent_runtime_server.dart`：Dio + SSE 封装。

### 5.2 侧栏入口

- 「Agent 对话」→ `GeegooAgentChatView`
- 「Agent 管理」→ `GeegooAgentMgtView`

### 5.3 部署注意

- Web 跨域：Nginx 反代 `/agent-runtime/*` → `:3400`，或经 GeeGooBot BFF。
- API Key 存运营后台配置或环境注入，勿提交仓库。

### 5.4 运营登录与 MCP Token

登录成功后 `SharedPreferences` 写入 `user_id` / `mcp_token`（Admin `login` 接口返回）。

- Chat / Plan / Clarify 自动携带 `X-MCP-Token`、`X-User-Id`
- 无 Token 时在对话面板提示前往「个人中心」生成
- 生产 BFF 开启 `GEEGOO_AGENT_VALIDATE_MCP_TOKEN` 时，Token 必须与 Mongo `user.mcp.mcp_token` 一致

## 6. 演进路线

| Phase | 前端 | 后端 | 状态 |
|-------|------|------|------|
| **1** | Chat + Cockpit 骨架 | Cockpit 读 API | ✅ |
| **2** | FAB 浮标、Plan/Clarify、CORS | PG schema、`GEEGOO_CORS_ORIGINS`、stream clarify | ✅ |
| **3** | Memory chunks 列表 | PostgreSQL Session SSOT、`geegoo migrate --to postgres`、pgvector | ✅ |
| **4** | BFF 接入、Memory 向量检索 | GeeGooBot `/op_agent` BFF、OpenAI embedding、summary 自动入库 | ✅ |
| **5** | Loop 架构图实时动画、Live Events | `sessionEventsStream` SSE、节点高亮 | ✅ |
| **6** | Nginx 同源、BFF MCP 校验 | `agent_bff_use_proxy`、Mongo MCP 验证 | ✅ |
| **7** | 运营登录态 → MCP Token | `AgentSessionContext`、Chat/Plan/Clarify 透传 | ✅ |

部署与联调见 [GeeGooBot deployment](../../GeeGooBot/docs/deployment.md)、`scripts/verify_agent_bff.py`。

部署说明见 [deploy/web-platform.md](../../deploy/web-platform.md)。

## 7. 非目标

- 独立公网 Cockpit 域名
- GeeGooAgent 仓库内嵌完整产品 Flutter
- 第二套登录系统
- 产品化 TUI

## 8. 参考

- Waku Dashboard：`waku/ops/dashboard.py`、`waku/ops/static/`
- ADP 对齐：`adp_ops_agent/ops-assistant-technical-development/agent-ops/waku-alignment.md`
- Runtime API 详表：`docs/refactor/GeeGooAgent/runtime-api.md`（若存在）或 `internal/runtimeapi/`
