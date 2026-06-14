# L1 — Model Gateway

Runtime **禁止**直连 OpenAI/Anthropic；Planner 只调 Gateway。

## 模块设计说明

L1 是 Agent 与**大模型供应商**之间的唯一网关。把「选哪个模型、失败了怎么办、花了多少钱」从 L4 Planner 中剥离，避免 Runtime 代码里散落 provider 特化逻辑。

**核心设计决策**


| 决策                            | 理由                                                           |
| ----------------------------- | ------------------------------------------------------------ |
| 统一 `chat(messages, tools)` 接口 | Planner 只关心 Message + ToolSchema，不关心 OpenAI/Anthropic SDK 差异 |
| 主模型 + 可选 Fallback             | 盘前长任务怕单点故障；主模型 3 次重试后再切备用                                    |
| CostManager 旁路记账              | 按 session/step 累计 token，为后续限流与成本告警预留                         |
| 轻量限流（MVP）                     | 指数分析 × 多股并行时防止 burst 打满 quota                                |


**调用链**

```text
Planner (L4)
    └── Gateway.chat(messages, tools)
            ├── Provider.primary  → OpenAI / Anthropic
            ├── retry (3×, 5s)
            ├── Provider.fallback (optional, 1×)
            └── CostManager.record(usage)
```

**边界**

- **提供**：LLM 对话、Tool Calling 响应解析、重试/Fallback、成本记录
- **不提供**：Prompt 组装（L4 ContextBuilder）、Tool 执行（L2）、记忆压缩（L3）
- **配置来源**：`config.json` 的 `llm` 段 + `SecretsProvider` 中的 API Key

**MVP 范围**

单主模型（如 `gpt-4o`）+ 固定重试即可跑通盘前；Fallback 与精细 Cost 报表可 Phase 2 加强。

## 模块索引


| 模块           | 文档                                   |
| ------------ | ------------------------------------ |
| Gateway      | [gateway.md](./gateway.md)           |
| Cost Manager | [cost-manager.md](./cost-manager.md) |
| Providers    | [providers.md](./providers.md)       |


## 代码

`src/geegoo/llm/gateway.py`