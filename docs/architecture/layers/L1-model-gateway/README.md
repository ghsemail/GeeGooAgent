# L1 — Model Gateway

Runtime **禁止**直连厂商 SDK；所有 LLM 调用经 Gateway。

> Go 实现：`internal/llm/`

## 模块索引

| 模块 | 文档 | 代码 | 状态 |
|------|------|------|------|
| Gateway | [gateway.md](./gateway.md) | `gateway.go` | ✅ 重试 + ctx 取消 |
| Providers | [providers.md](./providers.md) | `openai.go`, `presets.go` | ✅ 3 provider |
| Cost | [cost-manager.md](./cost-manager.md) | — | 📋 轻量 / 规划 |

## 调用链

```text
ReActLoop / report.Synthesizer / prompt.summary
    └── llm.Gateway.Chat(ctx, messages, schemas)
            └── Provider.Chat (OpenAI 兼容)
                    ├── 主模型重试（可配置）
                    └── context 取消 → 立即返回
```

## Provider 预设

`presets.go` — 当前支持：

| Provider | API 模式 | 特性 |
|----------|----------|------|
| DeepSeek | chat_completions | thinking / reasoning_content |
| OpenAI | chat_completions | 标准 |
| Minimax | chat_completions | 标准 |

`BuildProviderFromLLMFields(provider, model, token, thinking, effort)` 统一解析。

**不实现** Hermes 的 18+ provider、codox_responses、anthropic_messages（按需扩展）。

## 配置

`config.json` → `llm`：

```json
{
  "llm": {
    "provider": "deepseek",
    "model": "deepseek-chat",
    "token_key": "sk-...",
    "thinking": false,
    "max_steps": 80
  }
}
```

Chat 内 `/model` 可切换；workflow 合成阶段共用同一 Gateway。

## 上下文窗口

`context_window.go` — 按模型名解析 `context_length`，供 `prompt.Compressor` 阈值计算。

## 流式输出

`openai_stream.go` — HTTP runtime 与 chat TUI 可选用流式；workflow 合成为非流式。

## 边界

- **提供**：LLM 对话、tool_calls 解析、重试、取消
- **不提供**：Prompt 组装（`chatprompt`）、Tool 执行（L2）、记忆压缩逻辑（`prompt` 调用 Gateway 做摘要）

## 与 Hermes 对照

| Hermes | GeeGooAgent |
|--------|-------------|
| `runtime_provider.py` 18+ providers | `presets.go` 3 providers |
| 3 API modes | 仅 OpenAI 兼容 chat_completions |
| `prompt_caching.py` Anthropic 缓存 | DeepSeek 前缀缓存靠稳定 system |

## 延伸阅读

- [gateway.md](./gateway.md)
- [../L3-memory/compaction.md](../L3-memory/compaction.md)
