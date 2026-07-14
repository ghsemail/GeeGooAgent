# MCP API：GRIDBot 接口说明

## 概述

本文档描述通过 MCP（Skills）对 **GRID 网格交易机器人**（`bot_type: GRID`）的**创建、修改、删除、列表与运行日志**接口。分类与命名见 [`common.md`](common.md)「机器人分类与命名」。调用方不传 `user_id`，改为传入 `mcp_token`，由服务端根据 `mcp_token` 解析出对应用户后再调用 Bot 服务对应逻辑（列表与日志在 MCP 进程内直读数据库）。

- **基础路径**：GeeGooBot mcp-api 根地址（默认示例：`http://127.0.0.1:3120`）
- **认证方式**：请求头 `Authorization: Bearer <API_KEY>`；缺少或错误的 API Key 时 HTTP **401**（响应体为 `error` 字段说明，非下文 `code` 体系）。
- **缺少 `mcp_token` 或必填业务 ID**：未传 `mcp_token`，或更新/删除时未传 `bot_id`，HTTP 为 **400**，响应 JSON 中 **`code` 为 401**（`message` 提示缺少的字段）。这与 **无效 `mcp_token`**（找不到用户）时的 **`code` 102**、HTTP **401** 不同，调用方需区分。
- **选股与仓位**：建议先通过 Signal 的 **`/searchCode`**（或项目内 `Utility.searchCode`）按代码/名称搜索，由用户选定 **唯一标的** 后，将返回的 **`code`、`lot_size`**（及名称等）用于创建请求。若请求中带 **`lot_size`**（见下表），MCP 会按模板生成**完整 `order_size`**（`base_order_size`、`safety_order_size` 与该值一致，其余字段为 Grid 默认模板，并可被请求中的 `order_size` 覆盖）。

**公共约定**：认证与 **`mcp_token`**、MCP **`/searchCode`**、**`frequency`**、信号查询见 [`common.md`](common.md)；技术分析 **`prompt_id` / `period`**、**`attitude`** 见 [`agent-analyst.md`](../analyst/agent-analyst.md)。

**GRID 网格交易机器人**为实盘网格策略；未在请求中给出的字段由服务端按内置默认策略模板填充，字段含义见下文各节。

---

## 创建 GRIDBot

### 接口定义

| 项目 | 说明 |
|------|------|
| **URL** | `/createGRIDBot` |
| **方法** | `POST` |
| **Content-Type** | `application/json` |
| **认证** | 需要在 Header 中携带 `Authorization: Bearer <API_KEY>` |

### 请求体参数

请求体为 JSON。下表列出**创建时允许传入**的字段；未传入的项由服务端按内置默认模板填充。其中**对冲相关嵌套结构**（`hedging`）使用模板默认值，**不能**通过本接口请求体单独配置或覆盖。

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| **mcp_token** | string | 是 | 用户 MCP 令牌，用于在服务端解析 `user_id`，不传则返回 400。 |
| **botname** | string | 是* | 机器人名称，用于展示与唯一性校验（全局 `botname` 不可重复）。 |
| **stock_name** | string | 否 | 标的名称。 |
| **code** | string | 否 | 标的代码，如 `518880.SH`、`00700.HK`（与选股结果一致）。 |
| **lot_size** | integer | 否 | **每手股数**，来自选股接口返回；与 **`order_size.lot_size` 二选一**（任一方有值即触发 MCP 层按模板生成完整 **`order_size`**）。 |
| **frequency** | string | 否 | K 线/检查频率，如 `60m`、`5m`；默认与模板一致；共用枚举见 [`common.md`](common.md)。GRID 调度间隔见下文 **frequency 取值说明**。 |
| **grid** | object | 否 | 网格配置，结构见下文 **grid 结构说明**。 |
| **order_size** | object | 否 | 仓位与加仓规模配置，字段含义见下文 **order_size 结构说明**。若请求根级 **`lot_size`** 或本对象内 **`lot_size`** 有值，MCP 会按模板生成完整对象后再与本对象合并（见概述「选股与仓位」）。 |
| **attitude** | object | 否 | 态度/分析配置，结构见下文 **attitude 结构说明**。 |

\* 业务上应提供有效 `botname`；服务端对空值按「未传」处理，可能导致使用默认空名称或校验失败。

**说明**：调用方不要传 `type`（固定为 `GRID`）、`bot_id` / `_id`（由服务端写入数据库）；`user_id` 由服务端根据 `mcp_token` 解析后填入。

#### frequency 取值说明（GRID）

`frequency` 表示检查周期，调度逻辑见 `GRID/BasicGridBot.py` 中的 `getFrequency()`。常用取值如下：

| 取值 | 含义 | 调度间隔（分钟） |
|------|------|------------------|
| **5m** | 5 分钟 | 5 |
| **60m** | 60 分钟 | 60 |
| **daily** | 每日 | 1440 |

未在上述取值中或未传时：未传则通常使用模板默认（如 `5m`）；若传入其它字符串，服务端可能按 1 分钟间隔处理（不推荐）。

#### grid 结构说明

网格参数对象：

```json
{
  "upper_limit_price": 180.0,
  "lower_limit_price": 100.0,
  "grid_num": 9
}
```

- **upper_limit_price**：网格上限价格。
- **lower_limit_price**：网格下限价格。
- **grid_num**：网格数量（整数）。

#### order_size 结构说明（GRID）

`order_size` 描述首单股数、加仓基数与加仓规模递增系数等；未传时由服务端使用内置默认模板。

**约定**：**`base_order_size`（首单头寸）**与 **`safety_order_size`（加仓头寸基数）**均应为标的 **`lot_size`（每手股数）的整数倍**，便于按「手」下单；若未从选股结果获知 `lot_size`，也应自行按交易所每手股数对齐后再填。

```json
{
  "base_order_size": 100,
  "safety_order_size": 100,
  "safety_orders_volume_scale": 2
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| **base_order_size** | number | **首单头寸**（股数）；须为 **`lot_size` 的整数倍**。 |
| **safety_order_size** | number | **加仓头寸基数**（股数）；须为 **`lot_size` 的整数倍**。 |
| **safety_orders_volume_scale** | number | **加仓头寸递增规模**系数。 |

**与 MCP 创建请求的配合**：若请求中带根级 **`lot_size`** 或 **`order_size.lot_size`**，MCP 会按内置模板生成完整 **`order_size`**，并将 **`base_order_size`、`safety_order_size`** 置为该每手股数（自然满足为 **`lot_size` 的一倍**）；你仍可在同一请求中通过 **`order_size`** 传入上表其它字段覆盖默认值，覆盖时同样应保持 **`base_order_size`、`safety_order_size` 为 `lot_size` 的整数倍**。

#### attitude 结构说明

`attitude` 为对象，用于配置是否启用「态度/分析」类能力及关联的分析 Prompt：

```json
{
  "analysis_prompt_list": ["66436c877c670036b234fd40"],
  "analysis_period": "daily",
  "controll_switch": false,
  "switch": false
}
```

- **analysis_prompt_list**：字符串 ID 数组，每项为技术类分析 Prompt 的 **`prompt_id`**。请先调用 MCP **`POST /getSinglePromptTemplate`**，请求体含 **`mcp_token`**、**`type`: `tech`**，可按需传 **`period`**（例如 **`monthly`**，**仅使用该周期下的 Prompt**）；将所需项的 **`prompt_id`** 填入本数组。详见 [`agent-analyst.md`](../analyst/agent-analyst.md)。
- **analysis_period**：字符串；枚举见 [`agent-analyst.md`](../analyst/agent-analyst.md) **约定与枚举 · period**；未传时默认 **`daily`**（与 **`Constants/Basic_bot.py`** 一致）。详见同文档 **attitude.analysis_period**。
- **controll_switch**：是否开启态度控制相关开关。
- **switch**：态度/分析功能的总开关。

---

### 响应说明

- **成功**：`code === 100`，表示创建 GRID Bot 成功（Bot 服务消息示例：`创建GridBot成功`）。
- **业务错误**：`code` 为 101（如 bot 名称已存在）、102、103、105 等时表示业务校验失败，HTTP 状态码一般为 400。
- **mcp_token 无效**：未找到对应用户时返回 `code: 102`，HTTP 401。
- **调用 Bot 服务失败**：返回 502 及相应错误信息。

响应体示例（成功）：

```json
{
  "code": 100,
  "message": "创建GridBot成功"
}
```

---

### 请求示例

```bash
curl -X POST "http://localhost:3120/createGRIDBot" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <API_KEY>" \
  -d '{"mcp_token":"mcp_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx","botname":"黄金网格","stock_name":"黄金ETF","code":"518880.SH","lot_size":100,"frequency":"5m","grid":{"upper_limit_price":350,"lower_limit_price":300,"grid_num":6},"attitude":{"analysis_prompt_list":[],"analysis_period":"daily","switch":false,"controll_switch":false}}'
```

---

### 与 Bot 服务的关系

- 本接口用 `mcp_token` 解析 `user_id`，将参数转发至 Bot 服务的 `POST /createBot`（`bot_type: "GRID"`）。
- 创建逻辑、数量与权限校验、调度任务注册与用户通知等在 **Bot 服务** 中完成。

---

## 修改 GRIDBot

### 接口定义

| 项目 | 说明 |
|------|------|
| **URL** | `/updateGRIDBot` |
| **方法** | `POST` |
| **Content-Type** | `application/json` |
| **认证** | 需要在 Header 中携带 `Authorization: Bearer <API_KEY>` |

### 请求体参数

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| **mcp_token** | string | 是 | 用户 MCP 令牌，用于在服务端解析 `user_id`。 |
| **bot_id** | string | 是 | 要修改的 GRID Bot ID（即库中 `grid_bot` 文档 `_id`），可从「获取所有 GRIDBot」或业务侧查询取得。 |
| **botname** | string | 否 | 机器人名称。 |
| **stock_name** | string | 否 | 标的名称。 |
| **code** | string | 否 | 标的代码。 |
| **frequency** | string | 否 | 检查频率。 |
| **grid** | object | 否 | 网格配置，结构同创建接口。 |
| **lot_size** | integer | 否 | 与创建接口相同；若提供则按创建接口规则生成完整 **order_size**（可与库中已有 `order_size` 合并后再覆盖）。 |
| **order_size** | object | 否 | 仓位配置；字段含义与创建接口中的 **order_size 结构说明** 相同。 |
| **attitude** | object | 否 | 态度/分析配置。 |

**说明**：仅传需要修改的字段即可，未传字段保持数据库中原有值。若存在存量持仓，Bot 服务可能返回提示要求手动处理卖出（见 Bot 服务 `editGRIDBot` 返回的 `message`）。

### 响应说明

- **成功**：`code === 100`，`message` 可能为「更新GRID Bot配置成功」或「存量持仓请手动卖出，更新后的GRID Bot将初始化所有参数」等。
- **业务错误**：`code` 为 101、102、103、105 等时，HTTP 状态码一般为 400。
- **mcp_token 无效**：`code: 102`，HTTP 401。
- **调用 Bot 服务失败**：502。

### 请求示例

```bash
curl -X POST "http://localhost:3120/updateGRIDBot" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <API_KEY>" \
  -d '{
    "mcp_token": "mcp_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
    "bot_id": "67175fa4d6f9d6cf92bc5dd8",
    "frequency": "60m"
  }'
```

### 与 Bot 服务的关系

- 用 `mcp_token` 解析 `user_id`，将 `bot_id` 及可更新字段转发至 Bot 服务的 `POST /editBot`（`bot_type: "GRID"`）。

---

## 删除 GRIDBot

### 接口定义

| 项目 | 说明 |
|------|------|
| **URL** | `/deleteGRIDBot` |
| **方法** | `POST` |
| **Content-Type** | `application/json` |
| **认证** | 需要在 Header 中携带 `Authorization: Bearer <API_KEY>` |

### 请求体参数

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| **mcp_token** | string | 是 | 用户 MCP 令牌。 |
| **bot_id** | string | 是 | 要删除的 GRID Bot ID（即 `grid_bot` 集合 `_id`）。 |

### 响应说明

- **成功**：`code === 100`，`message` 可能为「删除bot成功」「已撤单，删除bot成功」「删除bot成功,请手动处理存量头寸」等（与持仓、挂单状态有关）。
- **业务错误**：`code` 为 102、103、104 等，HTTP 状态码为 400。
- **mcp_token 无效**：`code: 102`，HTTP 401。
- **调用 Bot 服务失败**：502。

### 与 Bot 服务的关系

- 用 `mcp_token` 解析 `user_id`，将 `bot_id` 转发至 Bot 服务的 `POST /deleteBot`（`bot_type: "GRID"`）。

---

## 获取所有 GRIDBot

### 接口定义

| 项目 | 说明 |
|------|------|
| **URL** | `/getAllGRIDBots` |
| **方法** | `POST` |
| **Content-Type** | `application/json` |
| **认证** | 需要在 Header 中携带 `Authorization: Bearer <API_KEY>` |

### 请求体参数

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| **mcp_token** | string | 是 | 用户 MCP 令牌。 |
| **code** | string | 否 | 按标的代码筛选；不传则返回该用户下全部 GRID Bot。 |

### 响应说明

- **成功**：`code === 100`，`data` 为当前用户（及可选 `code` 筛选下）的 GRID Bot 列表。每条记录在策略参数基础上合并运行信息（`grid_info`，不含 `grid_list` 大字段），并包含：**`bot_id`**（策略主文档 `_id`）、**`grid_bot_id`**（运行信息文档 `_id`）、**`bot_switch`**（是否启用的字符串形式开关），以及 **`grid`、`order_size`、`hedging`、`attitude`** 等。若 **`attitude.switch`** 为真且存在历史态度日志，响应中的 `attitude` 内可能额外包含 **`attitude`**（最新态度文本）、**`date`**。
- **mcp_token 无效**：`code: 102`，HTTP 401。

响应体示例（成功，字段仅供参考）：

```json
{
  "code": 100,
  "message": "success",
  "data": [
    {
      "user_id": "6366170502d5c175fd586fe8",
      "bot_id": "67175fa4d6f9d6cf92bc5dd8",
      "grid_bot_id": "67175fa4d6f9d6cf92bc5dd9",
      "botname": "黄金网格",
      "stock_name": "黄金ETF",
      "code": "518880.SH",
      "frequency": "5m",
      "type": "GRID",
      "bot_switch": "True",
      "grid": {
        "upper_limit_price": 350,
        "lower_limit_price": 300,
        "grid_num": 6
      },
      "order_size": {
        "base_order_size": 100,
        "safety_order_size": 100,
        "safety_orders_volume_scale": 2
      },
      "hedging": {
        "bullish": { "switch": false, "hedging_code": "", "hedging_stock_name": "" },
        "bearish": { "switch": false, "hedging_code": "", "hedging_stock_name": "" }
      },
      "attitude": {
        "analysis_prompt_list": [],
        "analysis_period": "daily",
        "switch": false,
        "controll_switch": false
      }
    }
  ]
}
```

### 请求示例

```bash
curl -X POST "http://localhost:3120/getAllGRIDBots" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <API_KEY>" \
  -d '{"mcp_token": "mcp_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"}'
```

### 与 Bot 服务的关系

- MCP 层通过 `mcp_token` 解析 `user_id` 后，从持久化存储中读取策略配置、运行信息与态度日志并组装为列表；**仅返回 GRID 交易机器人**，不包含 DCA、GRID Reminder 等其它类型。

---

## 获取 GRIDBot 运行日志

与 Bot 服务 **`POST /getGRIDBotLog`** 语义一致：返回网格决策日志（**`log`**）及持仓摘要（**`info`**）。MCP 校验 **`bot_id`** 属于当前用户后直读 **`grid_log`**、**`grid_info`**；**不提供日志筛选类请求参数**，固定返回最多 **100** 条全量记录。

### 接口定义

| 项目 | 说明 |
|------|------|
| **URL** | `/getGRIDBotLog` |
| **方法** | `POST` |
| **Content-Type** | `application/json` |
| **认证** | `Authorization: Bearer <API_KEY>` |

### 请求体参数

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| **mcp_token** | string | 是 | 用户 MCP 令牌。 |
| **bot_id** | string | 是 | GRID 机器人 ID（与 **`getAllGRIDBots`** 返回的 **`bot_id`** 一致）。 |

### 响应说明

- **成功**：HTTP 200，`code === 100`，`message` 一般为 `success`，业务载荷在 **`data`** 中（与 Bot 服务返回的 **`info` + `log`** 结构一致，外加 MCP 外层包装）。
- **错误码**：与 **`getDCABotLog`** 相同（缺少参数 **`401`**、无效 **`bot_id`** **`101`**、无权限 **`103`**、无效 token **`102`**）。

#### `data` 整体结构

| 字段 | 类型 | 说明 |
|------|------|------|
| **info** | object | 来自 **`grid_info.position`** 的持仓摘要；无运行信息文档时可能为空对象 `{}`。 |
| **log** | array | 网格运行快照列表，**按 `time` 倒序**，**最多 100 条**（与 `grid_log` 查询一致）。 |

#### `data.info`（持仓摘要）

与列表接口中合并后的持仓口径一致，典型字段如下（数值随行情与成交变化）：

| 字段 | 类型 | 说明 |
|------|------|------|
| **qty** | number | **持仓头寸**（股数）。 |
| **price** | number | 持仓成本价（或与策略展示一致的成本价）。 |
| **pl_val** | number | 浮动盈亏金额（示例中为负表示浮亏）。 |
| **pl_ratio** | number | 浮动盈亏比例（百分比数值，如 `-20.42` 表示约 -20.42%）。 |

#### `data.log[]`（单条网格日志）

每条对应一次写入 **`grid_log`** 的快照，除 **`time`** 外，其余多来自该条文档内的 **`log`** 子文档。

| 字段 | 类型 | 说明 |
|------|------|------|
| **time** | string | 该条日志对应的时间；MCP 层可能对 `datetime` 做 JSON 序列化（形态可为 ISO 字符串或 `"YYYY-MM-DD HH:MM:SS"`，以实际返回为准）。 |
| **current_grid** | number | 当前价格所在的网格档位价格（或策略定义的「当前格」参考价）。 |
| **buy_grid** | array of number | 买入侧网格线价格列表，**下标与 `buy_position` 一一对应**（同一索引表示该买价档上的头寸）。 |
| **sell_grid** | array of number | 卖出侧网格线价格列表。 |
| **buy_position** | array of number | 与各 **`buy_grid`** 档位对齐的持股数量（示例中常见为每档 100 股等）。 |
| **sell_position** | array | 卖出侧各档待卖数量；无挂卖时可为**空数组 `[]`**。 |
| **next_opt** | string | 策略给出的下一步操作意图，示例中多为 **`hold`**（观望）；其它取值以运行中策略写入为准。 |
| **position** | object | 该快照下的**头寸与订单摘要**，字段见下表。 |

#### `data.log[].position`（单条快照内的头寸对象）

| 字段 | 类型 | 说明 |
|------|------|------|
| **qty** | number | **持仓头寸**（股数）。 |
| **price** | number | 成本价或与展示一致的价格。 |
| **pl_val** | number | 该快照下的浮动盈亏金额。 |
| **pl_ratio** | number | 该快照下的浮动盈亏比例（% 数值）。 |
| **can_sell_qty** | number | **当前可卖头寸**（可卖股数）。 |
| **opt** | string | 头寸侧操作状态（如 **`hold`**）。 |
| **order_id** | string | 关联订单 ID；无订单时可为空字符串。 |
| **order_status** | string | 订单状态；无订单时可为空字符串。 |
| **order_time** | string | **订单时间**（若有）。 |

**说明**：随网格参数变化，**`buy_grid` / `sell_grid` / `current_grid`** 的档位会整体切换（例如从 `[300,400]` / `[600,700]` 与 **`current_grid: 500`** 变为 `[400,475]` / `[625,700]` 与 **`current_grid: 550`**），表示网格区间或档位已更新；**`info`** 通常对应当前最新汇总，**`log`** 为历史快照，用于复盘。

#### 响应体示例（节选）

```json
{
  "code": 100,
  "message": "success",
  "data": {
    "info": {
      "pl_ratio": -20.42,
      "pl_val": -12660.0,
      "price": 620.0,
      "qty": 100.0
    },
    "log": [
      {
        "time": "2026-03-27 16:00:00",
        "buy_grid": [300.0, 400.0],
        "sell_grid": [700.0, 600.0],
        "current_grid": 500.0,
        "buy_position": [100, 100],
        "sell_position": [],
        "next_opt": "hold",
        "position": {
          "can_sell_qty": 100.0,
          "opt": "hold",
          "order_id": "",
          "order_status": "",
          "order_time": "2025-11-19 13:30:03.774",
          "pl_ratio": -20.42,
          "pl_val": -12660.0,
          "price": 620.0,
          "qty": 100.0
        }
      }
    ]
  }
}
```

### 请求示例

```bash
curl -X POST "http://localhost:3120/getGRIDBotLog" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <API_KEY>" \
  -d '{"mcp_token": "mcp_xxx", "bot_id": "<BOT_ID>"}'
```

**盈亏明细**：Bot 服务 **`POST /getGRIDBotProfit`** 仍仅在 Bot 服务侧；MCP 若需封装可另行扩展。

---

## 行为摘要

- **创建 / 修改 / 删除**：MCP 将请求转发到 Bot 服务的 `POST /createBot`、`POST /editBot`、`POST /deleteBot`，请求体中带 `bot_type: "GRID"`（创建与修改时由 MCP 填入 `user_id`）。
- **列表 / 运行日志**：`getAllGRIDBots`、`getGRIDBotLog` 在 MCP 服务进程内查询 MongoDB 并组装结果，不经过上述三个路由。
- **MCP 对外路径**（**均为 `POST`**，均需 Header `Authorization: Bearer <API_KEY>`，且请求体均需 **`mcp_token`** 解析用户）：`/createGRIDBot`、`/updateGRIDBot`、`/deleteGRIDBot`、`/getAllGRIDBots`、`/getGRIDBotLog`。

**策略回测**：MCP 提供 **`POST /loopBackStrategy`**：请求体中 **`strategy_type`**（或 **`type`**）填 **`grid`** 时，须提供 **`code`**、**`frequency`**、**`fund`**、**`grid_param`**（网格参数对象）等；**`months_back`**、**`base_order_size`** 等为选填，以服务端校验为准。基于数据库中已保存 Bot 配置的回测由 Bot 服务侧 **`loopbackGRIDBot`** 等逻辑处理（是否在部署环境中暴露为 HTTP，以实际配置为准）。
