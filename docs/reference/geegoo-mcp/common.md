# MCP API 公共说明

本文档汇总 **geegoo mcp** 的**认证**、**`mcp_token`**、**账户持仓与 Bot 运行日志**、**机器人分类与 `frequency` 枚举**等跨专题约定。

**行情类接口**（搜码、报价、信号列表等）见 [market/trading-data.md](./market/trading-data.md)。**报告 Workflow** 见 [market/reports.md](./market/reports.md)。**分析与 Prompt** 见 [analyst/agent-analyst.md](./analyst/agent-analyst.md)。

接口分布总表：[interface-map.md](./interface-map.md)。

---

## 机器人分类与命名

产品侧的「机器人」分为**两类**：

| 类别 | 含义 |
|------|------|
| **提醒机器人** | 以**信号、网格或持仓**等维度做**提醒与监控**（是否涉及下单以各专题文档与实现为准）。 |
| **交易机器人** | 按策略执行**实盘或模拟交易**相关逻辑（见各专题文档）。 |

**提醒机器人（三类）**与专题文档、创建时 **`bot_type`** 的对应关系：

| 中文名称 | 专题文档 | `bot_type` |
|----------|----------|------------|
| **DCA 信号提醒机器人** | [reminder/dca-reminder.md](./reminder/dca-reminder.md) | `DCAReminder` |
| **GRID 网格提醒机器人** | [reminder/grid-reminder.md](./reminder/grid-reminder.md) | `GRIDReminder` |
| **Smart 交易提醒机器人** | [reminder/smart-reminder.md](./reminder/smart-reminder.md) | `SmartReminder` |

**说明**：**Smart 交易提醒机器人**与 **SmartTrade 交易机器人**名称相近，但 **`bot_type` 不同**（`SmartReminder` / `SmartTrade`），请勿混淆。

**交易机器人**与专题文档、**`bot_type`** 的对应关系：

| 中文名称 | 专题文档 | `bot_type` |
|----------|----------|------------|
| **DCA 信号交易机器人** | [bot/dca-bot.md](./bot/dca-bot.md) | `DCA` |
| **GRID 网格交易机器人** | [bot/grid-bot.md](./bot/grid-bot.md) | `GRID` |
| **SmartTrade 交易机器人** | [bot/smart-trade.md](./bot/smart-trade.md) | `SmartTrade` |
| **HDG 对冲交易机器人** | [bot/hdg-bot.md](./bot/hdg-bot.md) | `HDG` |

- **基础路径**：geegoo mcp 根地址（部署示例：`http://<host>:5700`）
- **标的搜索、报价、信号列表**：见 [market/trading-data.md](./market/trading-data.md)

---

## 认证与用户身份

| 项目 | 说明 |
|------|------|
| **Header** | `Authorization: Bearer <API_KEY>`，与 `mcpAPIServer` 中配置的 MCP API Key 一致 |
| **Content-Type** | `application/json`（`POST` 请求体为 JSON） |

需要绑定当前登录用户时，在 **JSON 请求体** 中传 **`mcp_token`**（存于用户文档 `mcp.mcp_token`）。服务端通过 `get_user_id_by_mcp_token` 解析为 `user_id`，再转发到 Bot / AIServer 等。

| 场景 | 说明 |
|------|------|
| 缺少或无效 `mcp_token` | 通常返回 `401` 或业务码 `102`（用户不存在） |
| 无需用户身份 | 部分接口可为 `{}`（如 Admin 信号列表）；详见 [trading-data.md](./market/trading-data.md) |

---

## 账户持仓查询（getPosition）

查询当前用户在**已绑定富途交易环境**（`user.trade` 中的 `bot_host`、`bot_port`、`trade_env`）下，某只标的的**实时账户持仓**。MCP 层将 **`mcp_token`** 解析为 `user_id` 后，转发至 Bot 服务的 **`POST /getPosition`**，与 App/HTTP 直连行为一致。

**SmartTrade `sell_only` 模式**：**成本价**仅能通过账户查询自动写入（**不可**在创建请求中手动填 `price`）；**头寸**默认由本接口与 Bot 使用同一查询自动带出全仓，若用户仅需管理部分持仓，可在创建时传**小于账户持仓**的 **`order_size.base_order_size`**（详见 [`smart-trade.md`](../bot/smart-trade.md)）。调用方可先通过本接口核对账户成本与数量。

| 项目 | 说明 |
|------|------|
| **URL** | `/getPosition` |
| **方法** | `POST` |
| **认证** | `Authorization: Bearer <API_KEY>`，且请求体需含 **`mcp_token`** |

**请求体**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| **mcp_token** | string | 是 | 用户 MCP 令牌，用于解析 `user_id`。 |
| **code** | string | 是 | 标的代码，如 `00700.HK`、`AAPL.US`。 |

**成功响应**（业务码 **`code === 100`**）：HTTP 200，`data` 为对象，常见字段包括（以实际返回为准）：

| 字段 | 说明 |
|------|------|
| **code** | 标的代码（与查询一致或经转换后的格式）。 |
| **position** | 持仓数量（股）。 |
| **can_sell_qty** | **当前可卖头寸**（可卖股数）。 |
| **cost_price** | 持仓成本价。 |
| **pl_val** / **pl_ratio** | 浮动盈亏金额 / 比例。 |

**其它业务码**（与 `botAPIServer` `/getPosition` 一致）：`101` 用户不存在；`102` 未查询到持仓；`103` 查询异常（`message` 含原因）。

---

## 按类型查询 Bot 运行日志（getBotLogByType）

通过 `type` 与 `bot_id` 统一查询 **DCA、DCA 提醒、GRID、GRID 提醒** 的运行日志，直读 Mongo 中对应 log 集合，逻辑与 `mcpAPIServer` 的 `getDCABotLog` / `getDCAReminderLog` / `getGRIDBotLog` / `getGRIDReminderLog` 一致；本接口在 `data` 中额外返回规范化后的 **`type`** 与 **`bot_id`**。

| 项目 | 说明 |
|------|------|
| **URL** | `/getBotLogByType` |
| **方法** | `POST` |
| **Content-Type** | `application/json` |
| **认证** | Header：`Authorization: Bearer <API_KEY>` |

**请求体参数**

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| **mcp_token** | string | 是 | 用户 MCP 令牌；无效时业务码 **102**，HTTP **401**。 |
| **type** | string | 是 | 机器人类型，不区分大小写；也支持去下划线形式，如 `dca_reminder` 与 `dcareminder` 均视为 DCA 提醒。 |
| **bot_id** | string | 是 | 目标文档的 Mongo `ObjectId` 字符串。DCA/GRID 为交易 bot 的 `_id`；`DCAReminder` / `GRIDReminder` 为提醒文档的 `_id`（与单独接口中的 `reminder_id` 同一含义）。 |

**`type` 允许值**（规范化后）：

| 请求写法示例 | 含义 |
|--------------|------|
| `DCA` | DCA 交易机器人日志 |
| `DCAReminder` / `dca_reminder` / `dcareminder` | DCA 提醒运行日志 |
| `GRID` | GRID 交易机器人日志 |
| `GRIDReminder` / `grid_reminder` / `gridreminder` | GRID 提醒运行日志 |

**成功响应**：HTTP **200**，`code=100` 时 **`data`** 结构因类型而异：

- **DCA**：`type`、`bot_id`、`info`（仓位摘要）、`log`（DCA 主循环日志）、`log_sr`（DCASR 相关日志）。
- **DCAReminder** / **GRIDReminder**：`type`、`bot_id`、`log`（列表）。
- **GRID**：`type`、`bot_id`、`info`（仓位摘要）、`log`（列表）。

**错误与业务码**

| code | HTTP 状态 | 说明 |
|------|------------|------|
| **401** | 400 | 缺少 `mcp_token` 或 `bot_id`。 |
| **400** | 400 | `type` 缺失或不在允许范围。 |
| **102** | 401 | 无效的 `mcp_token` 或用户不存在。 |
| **101** | 400 | 无效的 `bot_id`（非合法 ObjectId）。 |
| **103** | 400 | 未找到对应 bot/提醒，或当前用户无权限。 |
| **500** | 500 | 服务端异常。 |

**请求示例**

```bash
curl -X POST "http://<host>:5700/getBotLogByType" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <API_KEY>" \
  -d '{"mcp_token":"your_mcp_token","type":"DCA","bot_id":"66494754fbe37cd6846ebd89"}'
```

各 Bot/Reminder 专题文档中的 **`getDCABotLog`** 等单独接口仍可用；本接口为跨类型统一入口，归属校验与 **`mcp_token`** 约定同上。

---

## 行情类接口（索引）

`/searchCode`、`/getCurrentPrice`、`/getIndexSignalForSkill`、`/getSignalCombinationForSkill` 及交易日、逐笔、资金等，均在 **[market/trading-data.md](./market/trading-data.md)** 中说明。

---

## Prompt 模板与 AgentAnalyst（索引）

单项分析 Prompt、**`POST /getSinglePromptTemplate`**、**`getMCPAnalysis`**、竞品 / ETF 自建模板，以及枚举说明（三类 **`type`**、**TemplateType**、**creator**、**`period`**、**`attitude.analysis_period`**）均在 **[`agent-analyst.md`](../analyst/agent-analyst.md)**。

配置 **DCA / GRID** 等机器人的 **`attitude.analysis_prompt_list`**、**`analysis_period`** 时，请与该文档中的 **约定与枚举**、**`getSinglePromptTemplate`** 对齐 **`prompt_id`** 与 **`period`** 口径。

与本文件其它章节共用的仅为：**`Authorization`**、**`mcp_token`** → **`user_id`**（见上文 **认证与用户身份**）。

---

## frequency（K 线 / 检查频率）

不同机器人类型默认值不同，下表为常见**显式取值**（与 `GRID/BasicGridBot.getFrequency`、回测、提醒等文档对齐时可参考）。

| 取值 | 含义 |
|------|------|
| **1m** | 1 分钟 |
| **3m** | 3 分钟 |
| **5m** | 5 分钟 |
| **10m** | 10 分钟 |
| **15m** | 15 分钟 |
| **30m** | 30 分钟 |
| **60m** | 60 分钟 |
| **daily** | 日线 |

**说明**：

- **GRID 网格提醒机器人**：未传 `frequency` 时默认 **`5m`**（见 [reminder/grid-reminder.md](./reminder/grid-reminder.md)）。
- **Smart 交易提醒机器人**：未传时默认 **`60m`**（见 [reminder/smart-reminder.md](./reminder/smart-reminder.md)）。
- **DCA 信号交易 / 回测**：常用 **`5m`**、**`60m`** 等；回测接口若缺省，MCP 对 DCA 默认 **`60m`**、Grid 默认 **`5m`**（见 [strategy/loopback.md](./strategy/loopback.md)）。

信号列表项中的 **`frequency`** 表示该信号适用的周期，与机器人 **`frequency`** 配置应对齐或按产品说明选择。

---

## 文档索引

| 文档 | 内容 |
|------|------|
| [analyst/agent-analyst.md](./analyst/agent-analyst.md) | Prompt 约定与枚举、`getSinglePromptTemplate`、竞品/ETF CRUD、`getMCPAnalysis` |
| [reminder/dca-reminder.md](./reminder/dca-reminder.md) | DCA 信号提醒机器人 |
| [bot/dca-bot.md](./bot/dca-bot.md) | DCA 信号交易机器人 |
| [bot/hdg-bot.md](./bot/hdg-bot.md) | HDG 对冲交易机器人 |
| [reminder/grid-reminder.md](./reminder/grid-reminder.md) | GRID 网格提醒机器人；`frequency` 细节 |
| [reminder/smart-reminder.md](./reminder/smart-reminder.md) | Smart 交易提醒机器人 |
| [bot/grid-bot.md](./bot/grid-bot.md) | GRID 网格交易机器人 |
| [bot/smart-trade.md](./bot/smart-trade.md) | SmartTrade 交易机器人 |
| [strategy/loopback.md](./strategy/loopback.md) | 回测网格机器人和信号机器人|
| [strategy/strategy-generation.md](./strategy/strategy-generation.md) | 策略生成相关 |

本文档随 `mcpAPIServer` 中共用行为更新；若与上游 Bot/Admin/Prompt 服务行为不一致，以服务端实现为准。
