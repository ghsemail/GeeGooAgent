# Cross-Cutting — 横切能力

部署、质检、可观测性——**贯穿六层**、不属于单一层的工程能力。

## 模块设计说明

横切文档描述「如何让 Agent 在生产环境**跑得稳、查得到、交得出**」。它们不实现业务 Tool，也不替代 L0/L4，而是定义各层在上线时必须满足的**共用约定**。

**三大主题**

| 主题 | 文档 | 设计要点 |
|------|------|----------|
| Supervisor | [supervisor.md](./supervisor.md) | 跑后质检：每股是否有 report_id、必填字段、态度映射是否正确 |
| Observability | [observability.md](./observability.md) | 双轨日志（业务 execution log + 技术 trace）、EventBus 订阅、告警钩子 |
| Deployment | [deployment.md](./deployment.md) | systemd timer、Linux 路径、config 布局、与 Hermes cron 切换 |

**与各层关系**

```text
L4 Runtime ──emit──▶ EventBus ──▶ Observability（Logging/Tracing）
L5 Skill   ──▶ Supervisor（跑完检查 Working + 本地 md）
L0 Scheduler + Deployment ──▶ 触发 CLI，不内嵌 cron
L2 Tools   ──▶ execution_log 格式（rules/execution-log.md）
```

**设计原则**

- **可观测默认开启**：每次 Tool 调用有 `ToolCalled`/`ToolCompleted` 事件，MVP 即写 JSONL
- **Supervisor 不阻塞 Loop**：盘前允许「单股失败继续下一只」；Supervisor 汇总失败列表发告警
- **部署与代码分离**：`deploy/` 放 unit 文件；密钥走 L0 Secrets，不进仓库

**MVP 范围**

`execution_log` 规范 + 文件日志 + systemd 盘前 timer + 基础 Supervisor checklist（每股 report 存在）。

## 文档索引

- [supervisor.md](./supervisor.md)
- [observability.md](./observability.md)
- [deployment.md](./deployment.md)
