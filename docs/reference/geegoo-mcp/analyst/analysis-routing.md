# MCP 分析路由（文档索引）

**权威源文件（Agent 运行时嵌入同一内容）：**

[`internal/chatprompt/analysis_routing.md`](../../../../internal/chatprompt/analysis_routing.md)

请只修改上述源文件；部署 agent-runtime 后系统 prompt 自动更新。本文档为 docs 侧索引，避免与内嵌文件分叉。

涵盖内容：

- 技术分析（`type=tech`）/ 指标分析（`type=index`）/ 基本面分析（`type=fundamental`）定义与选用条件
- 用户意图 → 工具链路由决策树
- `get_single_prompt_template` / `get_mcp_analysis` 标准调用顺序
- 与 Bot 交易信号（MACDResonance 等）的边界
