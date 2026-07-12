# L3 — Context Compaction

## 职责

长对话接近模型上下文窗时自动压缩，避免 API context-length 失败与 token 费用失控。

## 实现

GeeGooAgent 采用 **Hermes-style 双阈值压缩**（`internal/prompt/compressor.go`）：

1. **回合开始 hygiene**（默认 85%）：对齐 Hermes Gateway Session Hygiene，无 IM 进程。
2. **环内压缩**（默认 50%）：每轮 `gateway.Chat` 之前。

chat CLI 与 HTTP `agent-runtime` 共用同一路径。

完整设计见 [`../../../superpowers/specs/2026-07-12-context-compression-design.md`](../../../superpowers/specs/2026-07-12-context-compression-design.md)。

### 触发

| 层 | 条件 | 事件 |
| --- | --- | --- |
| Hygiene | `tokens ≥ hygiene_threshold × context_length`（默认 0.85） | `context_hygiene` |
| 环内 | `tokens ≥ threshold × context_length`（默认 0.50） | `context_compressed` |

优先用上一轮 API `prompt_tokens`，否则 `EstimateTokens`（chars/4）。`context_length`：配置显式值优先，否则 `llm.ResolveContextWindow(model)`（如 `gpt-5.5`→400k，`deepseek-v4-*`→128k）。

### 四阶段

| 阶段 | 动作 |
| --- | --- |
| Phase 1 | 清除保护尾之外的大 tool 结果（无 LLM） |
| Phase 2 | 确定头/中/尾边界，对齐 tool 组 |
| Phase 3 | 辅助 LLM 生成结构化摘要（可配置 `auxiliary.compression`，空则回退主 LLM） |
| Phase 4 | 组装头 + 摘要 + 尾，sanitize tool 对，写回 `session.Messages` |

摘要失败时**跳过本次压缩**，保留完整中间轮（比 Hermes 更保守）。确定性 `workflow` 路径不调用 Compressor。

### 与 Prompt 稳定性

压缩不修改 `Messages[0]` 的稳定 system 正文；compaction note 放在摘要消息内，与 P2a 前缀缓存约定一致。
