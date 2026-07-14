# MCP API：DCABot 接口说明

## 概述

本文档描述通过 MCP（Skills）对 **DCA 信号交易机器人**（`bot_type: DCA`）的**创建、修改、删除、列表与运行日志**接口。分类与命名见 [`common.md`](common.md)「机器人分类与命名」。调用方不传 `user_id`，改为传入 `mcp_token`，由服务端根据 `mcp_token` 解析出对应用户后再调用 Bot 服务对应逻辑（列表与日志在 MCP 进程内直读数据库）。

- **基础路径**：GeeGooBot mcp-api 根地址（默认示例：`http://127.0.0.1:3120`）
- **认证方式**：请求头 `Authorization: Bearer <API_KEY>`；缺少或错误的 API Key 时 HTTP **401**（响应体为 `error` 字段说明，非下文 `code` 体系）。
- **缺少 `mcp_token` 或必填业务 ID**：未传 `mcp_token`，或更新/删除时未传 `bot_id`，HTTP 为 **400**，响应 JSON 中 **`code` 为 401**（`message` 提示缺少的字段）。这与 **无效 `mcp_token`**（找不到用户）时的 **`code` 102**、HTTP **401** 不同，调用方需区分。
- **选股与仓位**：建议先通过 Signal 的 **`/searchCode`**（或项目内 `Utility.searchCode`）按代码/名称搜索，由用户选定 **唯一标的** 后，将返回的 **`code`、`lot_size`**（及名称等）用于创建请求。若请求中带 **`lot_size`**（见下表），MCP 会按模板生成**完整 `order_size`**（`lot_size`、`base_order_size`、`safety_order_size` 与该值一致，其余字段为 DCA 默认模板，并可被请求中的 `order_size` 覆盖）。

**公共约定**：认证与 **`mcp_token`**、MCP **`/searchCode`**、**`frequency`**、信号查询见 [`common.md`](common.md)；技术分析 **`prompt_id` / `period`**、**`attitude`** 见 [`agent-analyst.md`](../analyst/agent-analyst.md)。

---

## 创建 DCABot

### 接口定义

| 项目 | 说明 |
|------|------|
| **URL** | `/createDCABot` |
| **方法** | `POST` |
| **Content-Type** | `application/json` |
| **认证** | 需要在 Header 中携带 `Authorization: Bearer <API_KEY>` |

### 请求体参数

请求体为 JSON。下表列出**创建时允许传入**的字段；未传入的项由服务端按内置默认模板填充。其中**对冲相关嵌套结构**（如 `hedging`）使用模板默认值，**不能**通过本接口请求体单独配置或覆盖。

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| **mcp_token** | string | 是 | 用户 MCP 令牌，用于在服务端解析 `user_id`，不传则返回 400。 |
| **botname** | string | 是* | 机器人名称，用于展示与唯一性校验（全局 `botname` 不可重复）。 |
| **stock_name** | string | 否 | 标的名称。 |
| **code** | string | 否 | 标的代码，如 `00700.HK`、`AAPL.US`（与选股结果一致）。 |
| **lot_size** | integer | 否 | **每手股数**，来自选股接口返回；与 **`order_size.lot_size` 二选一**（任一方有值即触发下方整表 `order_size` 生成）。不传则不在 MCP 层展开默认仓位，由 Bot 服务端模板处理。 |
| **frequency** | string | 否 | K 线/检查频率，如 `60m`、`5m`；共用枚举见 [`common.md`](common.md)。 |
| **signal** | object | 否 | 买卖信号配置，结构见下文 **signal 结构说明**。 |
| **tp** | object | 否 | 止盈配置，含义与触发逻辑见下文 **止盈止损参数逻辑**、字段见 **tp 结构说明**。 |
| **sl** | object | 否 | 止损配置，含义与触发逻辑见下文 **止盈止损参数逻辑**、字段见 **sl 结构说明**。 |
| **order_size** | object | 否 | 仓位配置，字段含义见下文 **order_size 结构说明**。若请求根级 **`lot_size`** 或本对象内 **`lot_size`** 有值，MCP 会按模板生成完整对象后再与本对象合并（见概述「选股与仓位」）。 |
| **advanced_setting** | object | 否 | 高级交易限制，结构见下文 **advanced_setting 结构说明**；未传时使用表中默认值。 |
| **attitude** | object | 否 | 态度/分析配置，结构见下文 **attitude 结构说明**。 |

\* 业务上应提供有效 `botname`；服务端对空值按「未传」处理，可能导致使用默认空名称或校验失败。

**说明**：调用方不要传 `type`（固定为 DCA）、`bot_id` / `_id`（由服务端写入数据库）；`user_id` 由服务端根据 `mcp_token` 解析后填入。

#### signal 结构说明

`signal` 为对象，包含 **`buy_signal`**、**`sell_signal`** 两个数组，表示买入侧与卖出侧指标链。

```json
{
  "buy_signal": [
    {
      "index": "SAR",
      "type": "signal",
      "param": { "acceleration": "0.02", "maximum": "0.2" }
    },
    {
      "index": "MACD",
      "type": "flag",
      "param": { "fastPeriod": "12", "signalPeriod": "9", "slowPeriod": "26" }
    }
  ],
  "sell_signal": [
    {
      "index": "nosignal",
      "type": "",
      "param": {}
    }
  ]
}
```

- **buy_signal**：买入信号列表，按顺序组合指标。每一项包含：
  - **index**：指标名，如 `SAR`、`MACD`、`EMA` 等。
  - **type**：**`signal`** 表示**信号**，**`flag`** 表示**趋势**；与 DCA Reminder 的 `signal` 配置含义一致。
  - **param**：该指标参数字典；**键和值在传输中一般为字符串**，例如 SAR 的 `acceleration`、`maximum`，MACD 的 `fastPeriod`、`signalPeriod`、`slowPeriod` 等。
- **sell_signal**：卖出信号列表，结构与 **buy_signal** 相同。若不做卖出信号判断，可使用一项：`index` 为 `"nosignal"`，`type` 可为空字符串，`param` 可为 `{}`。

#### attitude 结构说明

`attitude` 为对象，用于配置是否启用「态度/分析」类能力及关联的分析 Prompt。

```json
{
  "analysis_prompt_list": ["66436c877c670036b234fd40"],
  "analysis_period": "daily",
  "controll_switch": false,
  "switch": false
}
```

- **analysis_prompt_list**：字符串 ID 数组，每项对应一条技术类分析 Prompt 的 **`prompt_id`**。请先调用 MCP **`POST /getSinglePromptTemplate`**，请求体传入 **`mcp_token`**、**`type`: `tech`**，可按需传入 **`period`**（例如 **`monthly`**，**仅使用该周期下的 Prompt**，不要使用其它周期条目）；响应为 JSON 数组，将所需项的 **`prompt_id`** 填入本数组。详见 [`agent-analyst.md`](../analyst/agent-analyst.md)。
- **analysis_period**：字符串，单项分析 / 态度刷新采用的周期；枚举见 [`agent-analyst.md`](../analyst/agent-analyst.md) **约定与枚举 · period**；未传时默认 **`daily`**（与 **`Constants/Basic_bot.py`** 一致）。选取 **`prompt_id`** 时建议与 **`getSinglePromptTemplate`** 的 **`period`** 一致。
- **controll_switch**：是否开启态度控制相关开关。
- **switch**：态度/分析功能的总开关。

#### order_size 结构说明（仓位）

`order_size` 为对象，描述每笔手数、加仓规则与次数上限等；与 DCA 策略中「首单 + 多笔加仓」逻辑一致。未传时由服务端使用内置默认模板。

```json
{
  "lot_size": 100,
  "base_order_size": 100,
  "safety_order_size": 100,
  "max_safety_traders_count": 3,
  "max_active_safety_trades_count": 3,
  "price_deviation_to_open_safety_orders": 0,
  "safety_orders_volume_scale": 1,
  "safety_order_step_scale": 1
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| **lot_size** | number | **每手股数**（一手多少股），随标的而异；通常与选股接口返回的 `lot_size` 一致。 |
| **base_order_size** | number | **首单头寸**（股数），第一次买入数量；**一般为 `lot_size` 的整数倍**（便于按「手」下单）。 |
| **safety_order_size** | number | **加仓头寸基数**（股数），加仓单的计算基准；**一般亦为 `lot_size` 的整数倍**。 |
| **max_safety_traders_count** | number | **总开仓次数上限**（含首单与加仓），达到后不再开新仓。 |
| **max_active_safety_trades_count** | number | **当日开仓次数上限**，用于限制同一交易日内加仓频率。 |
| **price_deviation_to_open_safety_orders** | number | **加仓价格递减比例**（百分比等约定单位，与策略实现一致），用于计算下一笔加仓委托价相对当前价的偏移。 |
| **safety_orders_volume_scale** | number | **加仓头寸递增规模**系数，用于按加仓次数放大后续单笔数量。 |
| **safety_order_step_scale** | number | **加仓步长/递减系数**，与价格偏离、加仓阶梯计算相关（与 `price_deviation_to_open_safety_orders` 等配合使用）。 |

**与 MCP 创建请求的配合**：若请求中带根级 **`lot_size`** 或 **`order_size.lot_size`**，MCP 会以模板生成完整 `order_size`，并将 **`lot_size`、`base_order_size`、`safety_order_size`** 置为该每手股数；你仍可在同一请求中通过 **`order_size`** 传入上表其它字段覆盖默认值。

#### advanced_setting 结构说明（高级）

`advanced_setting` 为对象，用于限制开仓价格、两次交易之间的冷却时间、以及累计成交轮数上限等。未传时服务端使用与 `DCA_Bot.advanced_setting` 一致的默认模板。

**默认值（模板）**：

```json
{
  "maximum_price_to_open_deal": 1000,
  "cooldown_between_deals": 3600,
  "open_deals_stop": 10
}
```

| 字段 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| **maximum_price_to_open_deal** | number | `1000` | **最大允许开仓价格**（与标的报价同单位）。仅当**当前价低于该值**时才允许新开仓/加仓，用于避免在过高价位接盘；需按标的实际价格区间调整（低价股可改小，高价股需改大）。 |
| **cooldown_between_deals** | number | `3600` | **交易冷却时间**（**秒**）。上一笔成交与下一笔允许开仓之间的间隔下限；间隔不足则不会新开仓。 |
| **open_deals_stop** | number | `10` | **累计成交轮数上限**（`open_deals` 达到该值后关闭 Bot，不再交易）。 |

#### 止盈止损参数逻辑

DCA Bot 在持仓过程中按配置持续监测价格，达到止盈或止损条件时触发卖出等操作。`tp`、`sl` 的**定义与设置逻辑**如下。

**默认约定（便于理解各字段）**：动态止盈/止损默认可使用 **SAR** 指标（`tp_dynamic_index` / `sl_dynamic_index` 为 `"SAR"`）；止盈跟踪与止损跟踪**可以**默认开启（`profit_trailing`、`stop_loss_trailing` 为 `true`），跟踪幅度**可以**默认 **1%**（`profit_trailing_deviation`、`stop_loss_trailing_deviation` 为 `1`）。**若创建时未传 `tp` 或 `sl`，以服务端为 DCA Bot 内置的默认模板为准，数值可能与下述「默认」表述不完全一致。**

##### 止盈（tp）逻辑

- **开关**（`tp_switch`）：为 `true` 时启用止盈；达到止盈条件后触发。
- **模式**：二选一。
  - **动态止盈**（`tp_mode: "dynamic"`）：以**高于当前价格**的指标曲线为触发价，价格上穿该曲线即触发。需指定 `tp_dynamic_index`，主要枚举值为 `"SAR"`、`"BBAND"`（布林线）；`tp_dynamic_factor` 为在指标价之上的线性系数，系数越大止盈价越高，止盈点随行情动态变化。
  - **固定止盈**（`tp_mode: "fix"`）：以成本价为基准，涨幅达到 `fix_tp`（百分比）即触发。例如 `fix_tp: 5` 表示涨 5% 止盈。
- **止盈跟踪**（`profit_trailing: true`）：触发止盈后不立即卖出，继续持有；从**触发后的最高价**起算，当回落幅度达到 `profit_trailing_deviation`（百分比，如 1）时再卖出。用于在趋势行情中争取更高收益，但会承受自高点回落的利润回吐。

##### 止损（sl）逻辑

- **开关**（`sl_switch`）：为 `true` 时启用止损；达到止损条件后触发。
- **模式**：二选一。
  - **动态止损**（`sl_mode: "dynamic"`）：以**低于当前价格**的指标曲线为触发价，价格下穿该曲线即触发。需指定 `sl_dynamic_index`，止损点随行情动态变化。
  - **固定止损**（`sl_mode: "fix"`）：以成本价为基准，跌幅达到 `fix_sl`（百分比）即触发。例如 `fix_sl: 3` 表示跌 3% 止损。
- **止损后动作**（`stop_loss_action`）：触发止损并卖出后的行为，如：仅关闭当前交易（机器人继续运行、可再次买入）或关闭当前机器人（不再交易）。具体取值以服务端约定为准。
- **止损跟踪**（`stop_loss_trailing: true`）：触发止损后不立即卖出，从**触发后的最高价**起算，当回落幅度达到 `stop_loss_trailing_deviation`（百分比）时再卖出。用于在下跌中保留反弹空间，但会承受继续下跌带来的更大亏损风险。

#### tp 结构说明（止盈）

```json
{
  "tp_switch": true,
  "tp_mode": "dynamic",
  "tp_dynamic_index": "SAR",
  "tp_dynamic_factor": 1.0,
  "fix_tp": 5.0,
  "profit_trailing": true,
  "profit_trailing_deviation": 1.0
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| **tp_switch** | boolean | **止盈开关**。为 `true` 时开启止盈，达到止盈条件后触发止盈（卖出）。 |
| **tp_mode** | string | 止盈模式：`"dynamic"` 动态止盈（默认），`"fix"` 固定止盈。 |
| **tp_dynamic_index** | string | **动态止盈**时使用的指标，主要枚举：`"SAR"`（默认）、`"BBAND"`（布林线）；价格触及该指标价即触发。空字符串表示未选或使用固定止盈。 |
| **tp_dynamic_factor** | number | **动态止盈系数**。在标准指标止盈价之上按该系数线性上移，使止盈点随系数提高。 |
| **fix_tp** | number | **固定止盈**涨幅百分比。例如 5 表示买入后上涨 5% 即触发止盈。 |
| **profit_trailing** | boolean | **止盈跟踪**开关。为 `true` 时，触发止盈后不立即卖出，持续持有直至价格自「触发后的最高点」回落达到 `profit_trailing_deviation` 所设幅度再卖出。 |
| **profit_trailing_deviation** | number | **止盈跟踪幅度**（百分比）。默认常用 `1`，表示自最高点回落 1% 时卖出。 |

#### sl 结构说明（止损）

```json
{
  "sl_switch": true,
  "sl_mode": "dynamic",
  "sl_dynamic_index": "SAR",
  "fix_sl": 3.0,
  "stop_loss_trailing": true,
  "stop_loss_trailing_deviation": 1,
  "stop_loss_action": "trade"
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| **sl_switch** | boolean | **止损开关**。为 `true` 时开启止损，达到止损条件后触发止损（卖出）。 |
| **sl_mode** | string | 止损模式：`"dynamic"` 动态止损（默认），`"fix"` 固定止损。 |
| **sl_dynamic_index** | string | **动态止损**时使用的指标，主要枚举：`"SAR"`（默认）、`"BBAND"`（布林线）；价格触及该指标价即触发。空字符串表示未选或使用固定止损。 |
| **fix_sl** | number | **固定止损**跌幅百分比。例如 3 表示买入后下跌 3% 即触发止损。 |
| **stop_loss_trailing** | boolean | **止损跟踪**开关。为 `true` 时，触发止损后不立即卖出，持续持有直至价格自「触发后的最高点」回落达到 `stop_loss_trailing_deviation` 所设幅度再卖出，以保留反弹空间。 |
| **stop_loss_trailing_deviation** | number | **止损跟踪幅度**（百分比）。默认常用 `1`，表示自最高点回落 1% 时卖出。 |
| **stop_loss_action** | string | **止损后动作**。触发止损并卖出后的行为（如仅关闭当前交易或关闭当前机器人），具体取值以服务端约定为准；DCA 默认模板常为 `"trade"`（见 `Constants/Basic_bot.DCA_Bot`）。 |

---

### 响应说明

- **成功**：`code === 100`，表示创建 DCA Bot 成功。
- **业务错误**：`code` 为 101（如 bot 名称已存在）、102、103、105 等时表示业务校验失败，HTTP 状态码一般为 400。
- **mcp_token 无效**：未找到对应用户时返回 `code: 102`，HTTP 401。
- **调用 Bot 服务失败**：返回 502 及相应错误信息。

响应体示例（成功）：

```json
{
  "code": 100,
  "message": "创建DCA bot成功"
}
```

响应体示例（bot 已存在）：

```json
{
  "code": 101,
  "message": "bot已存在"
}
```

---

### 请求示例

```bash
curl -X POST "http://localhost:3120/createDCABot" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <API_KEY>" \
  -d '{"mcp_token":"mcp_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx","botname":"腾讯-DCA","stock_name":"腾讯控股","code":"00700.HK","lot_size":100,"frequency":"60m","signal":{"buy_signal":[{"index":"SAR","type":"signal","param":{"acceleration":"0.02","maximum":"0.2"}}],"sell_signal":[{"index":"nosignal","type":"","param":{}}]},"attitude":{"analysis_prompt_list":[],"analysis_period":"daily","switch":false,"controll_switch":false}}'
```

---

### 与 Bot 服务的关系

- 本接口用 `mcp_token` 解析 `user_id`，将参数转发至 Bot 服务的 `POST /createBot`（`bot_type: "DCA"`）。
- 创建逻辑、数量与权限校验、调度任务注册与用户通知等在 **Bot 服务** 中完成；部署时需将 MCP 所连的 Bot 服务基地址配置为可访问的 HTTP 地址（常见默认如 `http://127.0.0.1:3230`，以实际环境为准）。

---

## 修改 DCABot

### 接口定义

| 项目 | 说明 |
|------|------|
| **URL** | `/updateDCABot` |
| **方法** | `POST` |
| **Content-Type** | `application/json` |
| **认证** | 需要在 Header 中携带 `Authorization: Bearer <API_KEY>` |

### 请求体参数

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| **mcp_token** | string | 是 | 用户 MCP 令牌，用于在服务端解析 `user_id`。 |
| **bot_id** | string | 是 | 要修改的 DCA Bot ID（即库中 `dca_bot` 文档 `_id`），可从「获取所有 DCABot」或业务侧查询取得。 |
| **botname** | string | 否 | 机器人名称。 |
| **stock_name** | string | 否 | 标的名称。 |
| **code** | string | 否 | 标的代码；与创建接口相同。 |
| **frequency** | string | 否 | 检查频率。 |
| **signal** | object | 否 | 买卖信号配置，字段含义与创建接口中的 **signal 结构说明**相同。 |
| **tp** | object | 否 | 止盈配置，与创建接口中的 **止盈止损参数逻辑**、**tp 结构说明** 一致。 |
| **sl** | object | 否 | 止损配置，与创建接口中的 **止盈止损参数逻辑**、**sl 结构说明** 一致。 |
| **lot_size** | integer | 否 | 与创建接口相同；若提供则按创建接口规则生成完整 **order_size**（可与库中已有 `order_size` 合并后再覆盖）。 |
| **order_size** | object | 否 | 仓位配置；字段含义与创建接口中的 **order_size 结构说明** 相同。 |
| **advanced_setting** | object | 否 | 与创建接口中的 **advanced_setting 结构说明** 相同（含默认值表）。 |
| **attitude** | object | 否 | 态度/分析配置，字段含义与创建接口中的 **attitude 结构说明**相同。 |

**说明**：仅传需要修改的字段即可，未传字段保持数据库中原有值。

### 响应说明

- **成功**：`code === 100`，表示更新 DCA Bot 配置成功。
- **业务错误**：`code` 为 101、102、103、105 等时，HTTP 状态码一般为 400（MCP 层与创建接口使用相同映射；具体消息以 Bot 服务为准）。
- **mcp_token 无效**：未找到对应用户时返回 `code: 102`，HTTP 401。
- **调用 Bot 服务失败**：返回 502。

响应体示例（成功）：

```json
{
  "code": 100,
  "message": "更新DCA Bot配置成功"
}
```

### 请求示例

```bash
curl -X POST "http://localhost:3120/updateDCABot" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <API_KEY>" \
  -d '{
    "mcp_token": "mcp_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
    "bot_id": "67175fa4d6f9d6cf92bc5dd8",
    "frequency": "30m"
  }'
```

### 与 Bot 服务的关系

- 用 `mcp_token` 解析 `user_id`，将 `bot_id` 及可更新字段转发至 Bot 服务的 `POST /editBot`（`bot_type: "DCA"`）。
- 修改后的参数落库与调度任务更新在 **Bot 服务** 中完成。

---

## 删除 DCABot

### 接口定义

| 项目 | 说明 |
|------|------|
| **URL** | `/deleteDCABot` |
| **方法** | `POST` |
| **Content-Type** | `application/json` |
| **认证** | 需要在 Header 中携带 `Authorization: Bearer <API_KEY>` |

### 请求体参数

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| **mcp_token** | string | 是 | 用户 MCP 令牌。 |
| **bot_id** | string | 是 | 要删除的 DCA Bot ID（即 `dca_bot` 集合 `_id`）。 |

### 响应说明

- **成功**：`code === 100`，`message` 可能为「删除bot成功」「已撤单，删除bot成功」等（与持仓状态有关）。
- **业务错误**：`code` 为 102、103、104 等，HTTP 状态码为 400。
- **mcp_token 无效**：未找到对应用户时返回 `code: 102`，HTTP 401。
- **调用 Bot 服务失败**：返回 502。

### 请求示例

```bash
curl -X POST "http://localhost:3120/deleteDCABot" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <API_KEY>" \
  -d '{
    "mcp_token": "mcp_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
    "bot_id": "67175fa4d6f9d6cf92bc5dd8"
  }'
```

### 与 Bot 服务的关系

- 用 `mcp_token` 解析 `user_id`，将 `bot_id` 转发至 Bot 服务的 `POST /deleteBot`（`bot_type: "DCA"`）。
- 删除逻辑、必要时撤单、调度任务移除与可用机器人配额扣减在 **Bot 服务** 中完成。

---

## 获取所有 DCABot

### 接口定义

| 项目 | 说明 |
|------|------|
| **URL** | `/getAllDCABots` |
| **方法** | `POST` |
| **Content-Type** | `application/json` |
| **认证** | 需要在 Header 中携带 `Authorization: Bearer <API_KEY>` |

### 请求体参数

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| **mcp_token** | string | 是 | 用户 MCP 令牌。 |
| **code** | string | 否 | 按标的代码筛选；不传则返回该用户下全部 DCA Bot。 |

### 响应说明

- **成功**：`code === 100`，`data` 为当前用户（及可选 `code` 筛选下）的 DCA Bot 列表。每条记录在策略参数文档基础上合并运行信息文档（含开关等），并包含：**`bot_id`**（策略主文档标识）、**`dca_bot_id`**（运行信息文档标识）、**`bot_switch`**（是否启用调度/交易的字符串形式开关），以及 **`signal`、`tp`、`sl`、`order_size`、`hedging`、`attitude`** 等完整配置。若 **`attitude.switch`** 为真且存在历史态度日志，响应中的 `attitude` 内可能额外包含 **`attitude`**（最新态度文本）、**`date`**（对应时间）。
- **mcp_token 无效**：未找到对应用户时返回 `code: 102`，HTTP 401。

响应体示例（成功，字段仅供参考）：

```json
{
  "code": 100,
  "message": "success",
  "data": [
    {
      "user_id": "6366170502d5c175fd586fe8",
      "bot_id": "67175fa4d6f9d6cf92bc5dd8",
      "dca_bot_id": "67175fa4d6f9d6cf92bc5dd9",
      "botname": "腾讯-DCA",
      "stock_name": "腾讯控股",
      "code": "00700.HK",
      "frequency": "60m",
      "type": "DCA",
      "bot_switch": "True",
      "signal": { "buy_signal": [], "sell_signal": [] },
      "tp": {},
      "sl": {},
      "order_size": {},
      "hedging": {},
      "attitude": { "analysis_prompt_list": [], "analysis_period": "daily", "switch": false, "controll_switch": false }
    }
  ]
}
```

### 请求示例

```bash
# 获取该用户全部 DCA Bot
curl -X POST "http://localhost:3120/getAllDCABots" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <API_KEY>" \
  -d '{"mcp_token": "mcp_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"}'

# 仅获取指定标的的 DCA Bot
curl -X POST "http://localhost:3120/getAllDCABots" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <API_KEY>" \
  -d '{"mcp_token": "mcp_xxx", "code": "00700.HK"}'
```

### 与 Bot 服务的关系

- MCP 层通过 `mcp_token` 解析 `user_id` 后，从持久化存储中读取策略配置、运行信息与态度日志并组装为列表；**仅返回 DCA 交易机器人**，不包含网格、智能交易等其他类型。

---

## 获取 DCABot 运行日志

与 Bot 服务 **`POST /getDCABotLog`** 在「全量日志」口径上一致：从 **`dca_log`** 中分别查询 **DCA** 与 **DCASR** 两类记录，并附带 **`dca_info`** 中的**当前持仓情况**（**`info`**，与 GRID **`getGRIDBotLog`** 的 **`info`** 同角色）。**MCP 接口固定返回 **`info`**、**`log`**、**`log_sr`** 全量，不提供日志筛选类请求参数。** MCP 层先用 **`mcp_token`** 解析 **`user_id`**，并校验该 **`bot_id`** 属于当前用户后再读库，**不可**跨用户访问他人机器人日志（与 [`dca-reminder.md`](../reminder/dca-reminder.md) 中 **`getDCAReminderLog`** 的归属校验方式一致，只是业务 ID 为 **`bot_id`**）。

### 接口定义

| 项目 | 说明 |
|------|------|
| **URL** | `/getDCABotLog` |
| **方法** | `POST` |
| **Content-Type** | `application/json` |
| **认证** | `Authorization: Bearer <API_KEY>` |

### 请求体参数

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| **mcp_token** | string | 是 | 用户 MCP 令牌。 |
| **bot_id** | string | 是 | DCA 机器人 ID（与 **`getAllDCABots`** 返回的 **`bot_id`** 一致）。 |

**说明**：库中 **`dca_log`** 以 **`type: 'DCA'`**（信号）与 **`type: 'DCASR'`**（止盈止损）区分两类记录，响应中分别落在 **`data.log`** 与 **`data.log_sr`**；**`info`** 为**当前持仓情况**（来自 **`dca_info.position`**，与 GRID 日志的 **`info`** 角色相同）。**`data.log`** 为信号侧 DCA 日志（**按 `time` 倒序**，**最多 100 条**）；**`data.log_sr`** 为止盈止损侧（**按 `time` 倒序**，**最多 100 条**）。

### 响应说明

**成功（HTTP 200）**：`code === 100`，`message` 一般为 `"success"`，业务载荷在 **`data`** 对象中（与 Bot 服务返回的 **`info` + `log` + `log_sr`** 结构一致，外加 MCP 外层 **`code` / `message`**）。

**两类日志的含义**（便于与提醒类接口对照）：

- **`data.log`**：**信号侧**运行记录——每个调度周期里指标链 **`buy_signal` / `sell_signal`** 的评估结果与合成的 **`next_opt`**，语义上与 DCA 提醒的 **`getDCAReminderLog`**（见 [`dca-reminder.md`](../reminder/dca-reminder.md)）返回的「信号日志」**同类**，区别是此处属于**实盘 DCA 交易机器人**的 **`dca_log`**（`type === 'DCA'`）。
- **`data.log_sr`**：**止盈/止损侧**运行记录——持仓后的 **`tp_sl`**、**`trailing`** 等与风控相关的快照；由 **DCASR** 任务写入 **`dca_log`**（`type === 'DCASR'`）。

#### `data` 整体结构

| 字段 | 类型 | 说明 |
|------|------|------|
| **info** | object | **当前持仓情况**：来自 **`dca_info.position`** 的汇总（股数、成本价、浮动盈亏等）；无运行信息文档时可能为空对象 `{}`。 |
| **log** | array | **信号类**日志：DCA 调度快照（`type === 'DCA'`），**按 `time` 从新到旧**，**最多 100 条**；字段侧重 **`buy_signal` / `sell_signal`** 与 **`next_opt`**，与上文「两类日志的含义」中信号侧说明一致。 |
| **log_sr** | array | **止盈止损类**日志：DCASR 快照（`type === 'DCASR'`），**最多 100 条**，排序同上；字段侧重 **`tp_sl`**、**`trailing`** 等，与上文「两类日志的含义」中止盈/止损侧说明一致。 |

#### `data.info`（当前持仓情况）

表示该 DCA 机器人在 **`dca_info`** 中的**当前持仓与浮动盈亏汇总**，与 GRID **`getGRIDBotLog`** 的 **`info`** 同为「持仓快照」口径（列表接口 **`getAllDCABots`** 合并展示时也用同一套 **`position`** 字段）。典型字段如下（数值随行情与成交变化）：

| 字段 | 类型 | 说明 |
|------|------|------|
| **qty** | number | **持仓头寸**（股数）。 |
| **price** | number | 持仓成本价（或与策略展示一致的成本价）。 |
| **pl_val** | number | 浮动盈亏金额。 |
| **pl_ratio** | number | 浮动盈亏比例（百分比数值，如 `-20.42` 表示约 -20.42%）。 |

#### `data.log[]`（单条 DCA 日志 · 信号记录）

每条对应 **`dca_log`** 中 **`type === 'DCA'`** 且满足查询条件的一条记录，字段来自该条文档内的 **`log`** 子文档；记录的是**该周期信号与合成操作**的快照，**不是**止盈止损专表（止盈止损见 **`log_sr`**）。

| 字段 | 类型 | 说明 |
|------|------|------|
| **time** | string | 该条日志对应的时间；MCP 可能对 `datetime` 序列化为 **`YYYY-MM-DD HH:MM:SS`** 或 ISO 8601，以实际返回为准。 |
| **type** | string | 固定为 **`DCA`**，便于与 **`log_sr`** 区分。 |
| **next_opt** | string / object | 本周期合成后的操作意图；Bot 侧默认可能为对象 **`{}`**（若未写入），也可能为 **`hold`** / **`buy`** / **`sell`** 等字符串，以实际存储为准。 |
| **buy_signal** | object | **买入侧**信号评估结果，与机器人 **`signal.buy_signal`** 配置链对应；通常含 **`name`**（指标链列表，每项含 **`index`**、**`param`**、**`type`** 等）、**`value`**（本周期汇总，常含 **`signal`** 整型 **`0` / `1` / `-1`** 等）。 |
| **sell_signal** | object | **卖出侧**信号评估结果，结构同 **`buy_signal`**。 |
| **position** | object | 该快照下的**持仓概要**（头寸与订单侧）；见下表。 |

**`data.log[].position` 常见子字段**（同一对象内描述**持仓头寸**、**成本价**、**仓位操作**、**订单标识与状态**及盈亏等）：

| 子字段 | 类型 | 说明 |
|--------|------|------|
| **qty** | number | **持仓头寸**（股数）。 |
| **price** | number | **成本价**（或与该快照一致的持仓均价）。 |
| **pl_val** / **pl_ratio** | number | 该快照下的浮动盈亏金额 / 比例（若策略写入）。 |
| **opt** | string | **仓位操作**侧状态或意图（如 **`hold`** 等）。 |
| **order_id** | string | 关联 **订单 ID**；无订单时可为空字符串。 |
| **order_status** | string | **订单状态**；无在途订单时可为空字符串。 |
| **order_time** | string | **订单时间**（若有）。 |

相邻两条 **`time`** 的间隔取决于机器人的 **`frequency`** 与交易日历。

#### `data.log_sr[]`（单条 DCASR 日志 · 止盈止损记录）

每条对应 **`dca_log`** 中 **`type === 'DCASR'`** 的记录，记录持仓后的**止盈、止损与跟踪**等风控状态；与 **`data.log`** 中的 **`buy_signal` / `sell_signal`** 信号链**分工不同**，不要混在同一数组解读。

| 字段 | 类型 | 说明 |
|------|------|------|
| **time** | string | 该条日志时间，格式同 **`data.log[].time`**。 |
| **type** | string | 固定为 **`DCASR`**。 |
| **next_opt** | string | 本周期建议操作，默认 **`hold`**；其它取值以策略写入为准。 |
| **trailing** | object | 跟踪止盈/止损相关状态（未启用时可能为空对象 **`{}`**）。 |
| **tp_sl** | object | 止盈/止损价格与状态（如 **`tp_price`** 等字段，以实际存储为准）。 |
| **position** | object | 该快照下的**持仓概要**，与 **`data.log[].position`** 子字段含义相同（见上节「**`data.log[].position` 常见子字段**」）。 |

**说明**：**`info`** 为**当前持仓情况**（非历史序列）；**`log`**（信号）与 **`log_sr`**（止盈止损）均为**历史快照**，用于复盘；两类日志在同一响应中一并返回。

#### 响应体示例（节选）

```json
{
  "code": 100,
  "message": "success",
  "data": {
    "info": {
      "qty": 100.0,
      "price": 620.0,
      "pl_val": -12660.0,
      "pl_ratio": -20.42
    },
    "log": [
      {
        "time": "2026-03-27 16:00:00",
        "type": "DCA",
        "next_opt": "hold",
        "buy_signal": {
          "name": [
            { "index": "SAR", "param": { "acceleration": "0.02", "maximum": "0.2" }, "type": "signal" }
          ],
          "value": { "next_opt": "hold", "signal": 0 }
        },
        "sell_signal": {
          "name": [{ "index": "nosignal", "param": {}, "type": "" }],
          "value": { "next_opt": "hold", "signal": 0 }
        },
        "position": {}
      }
    ],
    "log_sr": [
      {
        "time": "2026-03-27 16:00:00",
        "type": "DCASR",
        "next_opt": "hold",
        "trailing": {},
        "tp_sl": {},
        "position": {}
      }
    ]
  }
}
```

**错误响应**：

- **缺少参数**：`code === 401`（未传 **`mcp_token`** 或 **`bot_id`**），HTTP 400。
- **无效令牌**：`code === 102`，HTTP 401。
- **ID 非法**：`code === 101`（非合法 ObjectId 字符串），HTTP 400。
- **无权限或不存在**：`code === 103`（该 **`bot_id`** 不属于当前用户或记录不存在），HTTP 400。

### 与 Bot 服务的关系

- Bot 侧：**`POST /getDCABotLog`**（请求体字段以 Bot 服务为准），不经 **`mcp_token`** 校验归属。
- MCP 侧：请求体仅需 **`mcp_token`**、**`bot_id`**；归属校验后在 **`mcpAPIServer`** 直读 **`dca_log`** / **`dca_info`**，固定返回全量 **`info`** + **`log`** + **`log_sr`**；成功响应统一为 **`{ code, message, data }`**；**`time`** 等字段可能经 MCP 做 JSON 友好序列化。

### 请求示例

```bash
curl -X POST "http://localhost:3120/getDCABotLog" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <API_KEY>" \
  -d '{"mcp_token": "mcp_xxx", "bot_id": "<BOT_ID>"}'
```

---

## 行为摘要

- **创建 / 修改 / 删除**：MCP 将请求转发到 Bot 服务的 `POST /createBot`、`POST /editBot`、`POST /deleteBot`，请求体中带 `bot_type: "DCA"`（创建与修改时由 MCP 填入 `user_id`）。
- **列表 / 运行日志**：`getAllDCABots`、`getDCABotLog` 在 MCP 服务进程内查询数据库并组装结果，不转发上述三个路由。
- **MCP 对外路径**（**均为 `POST`**，均需 Header `Authorization: Bearer <API_KEY>`，且请求体均需 **`mcp_token`** 解析用户）：`/createDCABot`、`/updateDCABot`、`/deleteDCABot`、`/getAllDCABots`、`/getDCABotLog`。其中 **`/getAllDCABots`**、**`/getDCABotLog`** 虽为「直读」MongoDB、不转发 Bot，但 HTTP 方法仍为 **`POST`**（非 `GET`）。
