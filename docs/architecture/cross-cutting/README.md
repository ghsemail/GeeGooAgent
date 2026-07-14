# Cross-Cutting — 横切能力

部署、质检、可观测性——贯穿各层、不属于单一层的工程约定。

## 模块索引

| 主题 | 文档 | 实现 |
|------|------|------|
| **Supervisor** | [supervisor.md](./supervisor.md) | `workflow/supervisor.go` |
| **验收** | — | `verify/verify.go`, `geegoo verify` |
| **可观测性** | [observability.md](./observability.md) | EventBus、execution_log、tool Meta |
| **部署** | [deployment.md](./deployment.md) | `install.sh`, `deploy_agent_server.py` |

## Supervisor 与 Verify

| 阶段 | 组件 | 输出 |
|------|------|------|
| Workflow 结束 | `Supervisor.Verify` | verdict: pass / recoverable / terminal |
| 人工 cutover | `geegoo verify` | 字段完整率矩阵，exit 1 |

Recoverable：列 missing steps，可 `geegoo resume`。  
Terminal：scheduler 退避后告警。

## 可观测性双轨

| 轨道 | 内容 | 位置 |
|------|------|------|
| 业务 | `write_execution_log` | `reports/<date>/execution-log.md` |
| 技术 | tool `Meta`（api_code, duration_ms） | chat 进度 / 日志 |
| 审计 | evidence_records | SQLite |

## 部署（GeeGooAgent 特化）

主机：**119.45.16.112**（`geegoo-agent`）

```text
本机 git push → 服务器 git reset --hard → start.sh build → restart-runtime
```

一键：`python scripts/deploy_agent_server.py`

验证：`geegoo doctor`、`:3400/health`

详见 [deployment.md](./deployment.md) 与 `~/.cursor/skills/remote-deploy/SKILL.md` §3.5。

## 与各层关系

```text
L4 workflow ──▶ Supervisor verdict
L2 tools    ──▶ execution_log + Result.Meta
L0 scheduler ──▶ 按 verdict 退避重试
L5 rules/   ──▶ 报告格式、attitude 映射
```

## MVP 范围

- execution_log 规范 ✅
- Supervisor checklist ✅
- systemd / 内置 scheduler ✅
- 分布式 tracing 📋
