# GeeGoo MCP API 文档

**服务名称**：GeeGooBot mcp-api  
**实现**：`mcp-api`（共享工具见 `mcp/constants.py`、`mcp/http_client.py`、`mcp/json_utils.py`）  
**默认端口**：3120  
**Base URL**：`http://<host>:3120`  
**认证**：`Authorization: Bearer <sk-...>` + 请求体 `mcp_token`（部分接口不需要 token，见总表）

> 原 GeeGooBot mcp-api（3120）已合并入 GeeGooBot mcp-api。

## 从这里开始

| 文档 | 说明 |
|------|------|
| **[interface-map.md](./interface-map.md)** | **接口分布总表** — 73 路由 × geegoo Skill × GeeGoo Agent Tool |
| [architecture.md](./architecture.md) | 三层架构与文档 SSOT 原则 |
| [../README.md](../README.md) | GeeGooBot 文档根目录 |

## 专题文档（参数与示例）

| 分类 | 文档 |
|------|------|
| 公共与账户 | [common.md](./common.md) |
| 行情与资金 | [trading-data](./market/trading-data.md) |
| 报告与 Workflow | [reports](./market/reports.md) |
| 分析 | [agent-analyst](./analyst/agent-analyst.md) |
| 策略（生成+回测） | [strategy/](./strategy/README.md) |
| Bot | [bot/](./bot/) |
| Reminder | [reminder/](./reminder/) |

## 维护

```bash
python scripts/generate_interface_map.py
```

geegoo Skill 与 GeeGoo Agent 文档与 Tool 命名 **以 `interface-map.md` 为准**；变更 SSOT 后运行 `scripts/sync_geegoo_consumers.py`。
