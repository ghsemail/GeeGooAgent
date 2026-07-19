# Agent 平台对标（Benchmark）

本目录收录 **GeeGooAgent** 与同类开源/公开 Agent 平台的功能整理与横向对比，供架构决策与缺口分析使用。

| 文档 | 内容 |
|------|------|
| **[agent-loop/](./agent-loop/)** | **Agent Loop 专项对标**（先 Hermes，再 Grok Build） |
| [agent-loop/optimization-roadmap.md](./agent-loop/optimization-roadmap.md) | Agent Loop **优化方案**（分阶段路线图） |
| [agent-loop/hermes.md](./agent-loop/hermes.md) | GeeGooAgent × Hermes Agent — ReAct 循环、压缩、子 Agent、优劣 |
| [agent-loop/grok-build.md](./agent-loop/grok-build.md) | GeeGooAgent × Grok Build — harness、Plan mode、并行子 Agent、Headless |
| [grok-build.md](./grok-build.md) | [Grok Build](https://github.com/xai-org/grok-build) 功能清单与架构要点 |
| [comparison.md](./comparison.md) | GeeGooAgent × Hermes Agent × Grok Build 三方对比 |
| [../deploy/hermes-parity-comparison.md](../../deploy/hermes-parity-comparison.md) | GeeGooAgent vs Hermes 深度对齐记录（P1–P8） |

## 定位速览

| 项目 | 语言 | 主要场景 | 开源 |
|------|------|----------|------|
| **GeeGooAgent** | Go | 金融盘前/盘中/盘后工作流、MCP 工具编排、可审计报告 | ✅ |
| **Hermes Agent** | Python | 通用多平台 Agent（IM、插件、70+ 工具） | ✅ |
| **Grok Build** | Rust | 终端编码 Agent（TUI / Headless / ACP） | ✅（Apache 2.0，不接受外部 PR） |

## 阅读建议

1. 若只关心 **Agent Loop 机制** → 进入 [agent-loop/](./agent-loop/)：先 [hermes.md](./agent-loop/hermes.md)，再 [grok-build.md](./agent-loop/grok-build.md)。
2. 若需要 **落地优化计划** → [agent-loop/optimization-roadmap.md](./agent-loop/optimization-roadmap.md)。
2. 若关心 **编码工作流**（Plan、多文件编辑、Git、CI Headless）→ 先看 [grok-build.md](./grok-build.md)，再查 [comparison.md](./comparison.md) 编码类行。
3. 若关心 **GeeGoo 与 Hermes 对齐度（含 P1–P8 交付）** → 以 [hermes-parity-comparison.md](../../deploy/hermes-parity-comparison.md) 为准；loop 细节见 [agent-loop/hermes.md](./agent-loop/hermes.md)。
4. 实现状态以 [implementation-status.md](../architecture/implementation-status.md) 与代码为准；对标文档可能滞后于发版。

## 外部链接

- Grok Build 仓库：<https://github.com/xai-org/grok-build>
- Grok Build 文档：<https://docs.x.ai/build/overview>
- Hermes Agent 架构：<https://hermes-agent.nousresearch.com/docs/zh-Hans/developer-guide/architecture>
