# 代码仓库布局（通用模板）

> 智能体生成新 Agent 时**必须按此树创建文件**。语言默认 **Go**（与 GeeGooAgent 参考实现一致）；若选 Python，将 `internal/` 映射为 `src/<agent>/`，接口不变。

---

## 完整目录树

```text
<AgentName>/
├── cmd/
│   ├── <agent>/                    # 主 CLI 二进制
│   │   ├── main.go                 # setup | doctor | chat | run | resume | update
│   │   ├── chat.go
│   │   ├── run.go
│   │   └── ops.go                  # setup, doctor
│   └── agent-runtime/              # 可选 HTTP API（Phase 2+）
│       └── main.go
├── internal/
│   ├── app/                        # 依赖注入：组装 Config/MCP/Registry/Gateway/Workflow
│   │   └── app.go
│   ├── config/
│   │   ├── config.go               # 加载 config.json + 环境变量
│   │   ├── paths.go                # 默认 ~/.<agent>/
│   │   └── endpoints.go            # 出站 URL 解析
│   ├── infra/                      # L0
│   │   ├── events.go               # EventBus
│   │   ├── state.go                # FileStateStore
│   │   └── checkpoint.go           # CheckpointManager
│   ├── llm/                        # L1
│   │   ├── types.go                # Message, ToolSchema, Response
│   │   ├── gateway.go              # 重试/Fallback
│   │   ├── openai.go               # OpenAI 兼容 Provider
│   │   └── mock.go                 # 测试用
│   ├── clients/                    # L2 底层 HTTP/MCP
│   │   └── <domain>/
│   │       ├── client.go
│   │       └── types.go
│   ├── tools/                      # L2
│   │   ├── registry.go
│   │   ├── bootstrap.go            # RegisterAll
│   │   ├── bespoke.go              # 手写领域 Tool
│   │   └── catalog/
│   │       └── catalog.go          # HTTP Tool 声明式注册
│   ├── memory/                     # L3
│   │   ├── models.go               # Working 结构体
│   │   └── working.go              # Create/Load/Save/Apply
│   ├── runtime/                    # L4
│   │   ├── session.go
│   │   ├── react.go                # ReActLoop
│   │   ├── executor.go             # 委托 Registry
│   │   ├── prompt.go
│   │   └── progress.go             # TUI 进度回调
│   ├── workflow/                   # L4
│   │   ├── runner.go               # WorkflowRunner
│   │   ├── steps.go                # Step 定义（Phase 1 可硬编码）
│   │   └── loader.go               # 读 manifest.yaml（Phase 3）
│   ├── runtimeapi/                 # HTTP handler（可选）
│   ├── doctor/
│   │   ├── doctor.go
│   │   └── connectivity.go
│   ├── chatsession/                # chat 持久化（Phase 2）
│   ├── chatprompt/
│   ├── auth/                       # Bearer 等
│   └── httpserver/                 # 共用 HTTP 栈
├── skills/                         # L5 Skill Pack
│   └── <first_skill>/
│       ├── SKILL.md
│       ├── manifest.yaml
│       ├── workflow.md
│       ├── template.md
│       └── supervisor_checks.yaml
├── rules/                          # 全局规则（markdown）
├── prompts/
│   └── identity.md
├── docs/
│   └── architecture/               # 项目专属架构（可 fork 本 blueprint）
├── deploy/
│   ├── systemd/
│   │   ├── <agent>-<skill>.service
│   │   └── <agent>-<skill>.timer
│   └── env.example
├── config.example.json
├── go.mod
├── README.md
└── start.sh                        # 服务器可选
```

---

## 层 → 包映射

| 层 | 包路径 | 禁止导入 |
|----|--------|----------|
| L5 | `cmd/`, `skills/`, `rules/`, `prompts/` | — |
| L4 | `internal/runtime`, `internal/workflow` | 不 import `clients` 直连 HTTP |
| L3 | `internal/memory`, `internal/chatsession` | 不 import `llm` |
| L2 | `internal/tools`, `internal/clients` | 不 import `runtime` |
| L1 | `internal/llm` | 不 import `tools` |
| L0 | `internal/infra` | 不 import 上层任何包 |
| 组装 | `internal/app` | 唯一允许横层 wiring 处 |

---

## 配置文件

**默认路径**：`~/.<agent>/config.json`（可用环境变量 `<AGENT>_HOME` 覆盖）。

**最小 schema**（`config.example.json`）：

```json
{
  "base_url": "http://127.0.0.1:8080",
  "api_key": "REPLACE",
  "user_token": "",
  "output_dir": "./data",
  "dry_run": false,
  "max_steps": 80,
  "llm": {
    "provider": "openai",
    "token_key": "",
    "model": "",
    "temperature": 0.2,
    "max_tokens": 4096
  },
  "sandbox": {
    "allowed_hosts": ["127.0.0.1", "localhost"]
  }
}
```

| 字段 | 必填 | 说明 |
|------|------|------|
| `base_url` | 是 | 主外部 API |
| `api_key` | 是 | Bearer / 服务密钥 |
| `user_token` | 视 Tool | 用户级 token（GeeGoo 为 `mcp_token`） |
| `output_dir` | 是 | StateStore / 报告 / checkpoint |
| `dry_run` | 否 | 全局 dry-run |
| `llm.token_key` | chat/ synthesis | LLM API Key |

---

## CLI 表面（必须实现）

| 命令 | Phase | 行为 |
|------|-------|------|
| `<agent> setup` | 0 | 写默认 config |
| `<agent> doctor` | 0 | 配置 + 连通性 |
| `<agent> chat` | 0/2 | ReAct 交互 |
| `<agent> run <skill>` | 1 | Workflow 执行 |
| `<agent> resume --session <id>` | 1 | 从 checkpoint 续跑 |
| `<agent> update` | 可选 | 拉代码重建二进制 |

---

## 测试目录

```text
tests/                              # 或 **/*_test.go 与源码同包
├── integration/
│   └── golden_test.go              # HTTP mock 对拍
internal/**/**/*_test.go            # 单元测试
```

**Phase 0 结束前必须有**：Registry mock Tool、Gateway mock Provider、StateStore 读写、ReAct 单轮测试。

---

## 智能体生成检查清单

创建仓库后逐项打勾：

- [ ] `go mod init` + `cmd/<agent>/main.go` 可编译
- [ ] `config.example.json` + `setup` 写 `~/.<agent>/config.json`
- [ ] `internal/infra` 四件套存在
- [ ] `internal/tools/registry.go` + 1 个 `echo` mock Tool
- [ ] `internal/llm/gateway.go` + `mock.go`
- [ ] `internal/runtime/react.go` 单轮可测
- [ ] `doctor` 输出 OK/FAIL 行
- [ ] `skills/<first>/manifest.yaml` 占位
- [ ] `README.md` 含 Quick Start 三命令
