# MCP API：SmartTrade 接口说明

## 概述

本文档描述通过 MCP（Skills）对 **SmartTrade 交易机器人**（`bot_type: SmartTrade`）的**创建、修改、删除、列表与运行日志**接口。分类与命名见 [`common.md`](common.md)「机器人分类与命名」。调用方不传 `user_id`，改为传入 `mcp_token`，由服务端根据 `mcp_token` 解析出对应用户后再调用 Bot 服务对应逻辑（列表与日志在 MCP 进程内直读数据库）。

- **基础路径**：GeeGooBot mcp-api 根地址（默认示例：`http://127.0.0.1:3120`）
- **认证方式**：请求头 `Authorization: Bearer <API_KEY>`；缺少或错误的 API Key 时 HTTP **401**（响应体为 `error` 字段说明，非下文 `code` 体系）。
- **缺少 `mcp_token` 或必填业务 ID**：未传 `mcp_token`，或更新/删除时未传 `bot_id`，HTTP 为 **400**，响应 JSON 中 **`code` 为 401**（`message` 提示缺少的字段）。这与 **无效 `mcp_token`**（找不到用户）时的 **`code` 102**、HTTP **401** 不同，调用方需区分。

**公共约定**：认证与 **`mcp_token`**、**`frequency`** 等共用说明，见 [`common.md`](common.md)。

---

## 交易模式与创建行为

SmartTrade 使用 **`trade_mode`** 声明运行模式：**`buy_then_sell`** 表示先买后卖；**`sell_only`** 表示仅对已持仓做卖出侧管理、不发起买入。请求未带该字段时，服务端通常按 **`buy_then_sell`** 处理。

| 取值 | 含义 |
|------|------|
| **buy_then_sell** | 先买后卖：创建成功后 Bot 服务会按配置尝试**开仓买入**（具体是否下单、是否使用限价由 Bot 内 `createSmartTrade` / `placeOrder` 决定）。 |
| **sell_only** | 仅卖出监控：假定已有持仓，只做止盈/止损与跟踪卖出；**不发起买入**。 |

**`price` 与 `order_size` 含义随模式变化**：

- **sell_only**：
  - **持仓成本价**：**只能**通过 **`/getPosition`** 查询绑定交易账户后**自动写入** Bot（创建时 Bot 使用与 **`/getPosition`** 相同的持仓查询）；**创建时不可手动录入 `price`**，调用方应**不要传 `price`**。
  - **头寸（股数）**：默认不传 **`order_size`** 时，由 **`/getPosition`** 同一套查询**自动带出该标的账户持仓（全仓）**。若需仅对部分持仓做卖出侧管理，可在 **`order_size.base_order_size`** 中填写**小于账户该标的当前持仓**的股数（**不可**大于账户持仓）；**宜为该标的 `lot_size` 的整数倍**。
  - 若账户侧查询失败（无持仓或连接异常等），创建失败。Skills 可先调用 [`common.md`](common.md) 的 **`/getPosition`** 核对成本与可卖数量。
- **buy_then_sell**：`price` 可作为**限价买入价**（也可不传，由策略用市价等逻辑）；`order_size.base_order_size` 为**首笔买入股数**，**宜为 `lot_size` 的整数倍**。若传入头寸且 ≤0，创建失败。

创建成功后，美股标的会进入美国调度器的 SmartTrade 任务，其它市场进入主调度器（与 `botAPIServer` 一致）。

---

## 修改限制与机器人状态（`trade_info.status`）

修改接口为 **`/updateSmartTrade`**（经 MCP 转发至 **`POST /editBot`**，`bot_type: SmartTrade`）。下列限制由 **botAPIServer `editSmartTrade`** 强制执行；调用方应先通过 **`/getAllSmartTrades`**（或 App 内列表）读取当前 **`trade_mode`** 与 **`status`**，再决定能否传 **price** / **order_size**。

### 状态码含义（`status`，与 `trade_info` 一致）

| `status` | 含义（业务） |
|----------|----------------|
| **1** | 开仓买入：首笔买入订单尚未结束（可视为「买入成交前」窗口）。 |
| **2** | 持仓：已有持仓，监控止盈/止损/卖出等。 |
| **3** | 减仓卖出：存在未完成卖出订单等。 |
| **4** | 关闭：机器人已关闭或流程结束。 |

（实现细节以 **`TRADE/SmartTrade.py`** 中状态机为准。）

### 按 `trade_mode` 区分：何时可改「价格」与「头寸」

| `trade_mode` | 可修改 **price** | 可修改 **order_size**（`base_order_size`） | 说明 |
|--------------|------------------|---------------------------------------------|------|
| **buy_then_sell** | 仅当 **`status === 1`** 时，更新请求中才可带 **price** 或 **order_size** | 同上 | 若 **`status !== 1`** 且请求中仍传入 **price** 或 **order_size**，返回 **`code: 101`**，提示仅买入成交前可改价格与下单头寸。其它字段（如 **tp**、**sl**、**frequency**、**botname** 等）不受此条限制（仍以服务端校验为准）。 |
| **sell_only** | **禁止**在更新请求中传入 **price**（成本价不可通过编辑修改） | 允许 | 若请求中带 **price**，返回 **`code: 101`**：`sell_only 模式下不允许修改价格（成本价）`。更新 **order_size** 仍可用于与持仓数量相关的同步（与 Bot 内逻辑一致）；**tp** / **sl** 等其它字段一般可照常修改。 |

**小结**：

- **buy_then_sell**：能否改「限价/下单价」和「首笔头寸」取决于是否仍处于 **status = 1**。
- **sell_only**：**永远不能**通过修改接口改 **price**；**order_size** 不在禁止之列。

---

## 止盈止损参数逻辑

SmartTrade 在持仓过程中按配置监测价格，触发止盈或止损条件后执行卖出等操作。`tp`、`sl` 的字段含义如下（默认值以服务端模板为准，未传字段由 Bot 侧补齐）。

**便于理解的默认倾向**：动态止盈/止损可使用 **SAR** 等指标（`tp_dynamic_index` / `sl_dynamic_index`）；模板中止盈跟踪、止损跟踪的默认开关与偏离度以数据库写入结果为准。

### 止盈（tp）

- **tp_switch**：为 `true` 时启用止盈。
- **tp_mode**：`dynamic` 为动态止盈（按指标与系数）；`fix` 为固定止盈（相对成本涨幅 `fix_tp` 百分比）。
- **tp_dynamic_index**：动态止盈使用的指标名，如 `SAR`、`BBAND`。
- **tp_dynamic_factor**：动态止盈在指标价之上的调整系数。
- **fix_tp**：固定止盈时，相对成本上涨的百分比阈值。
- **profit_trailing**：触发止盈后是否启用跟踪，自高点回落 `profit_trailing_deviation` 后再卖。
- **profit_trailing_deviation**：止盈跟踪的回落幅度（百分比）。

### 止损（sl）

- **sl_switch**：为 `true` 时启用止损。
- **sl_mode**：`dynamic` / `fix`，含义与止盈对称，固定止损用 **fix_sl**（跌幅百分比）。
- **sl_dynamic_index**：动态止损使用的指标名。
- **stop_loss_trailing** / **stop_loss_trailing_deviation**：止损跟踪开关与回落幅度。
- **stop_loss_action**：止损平仓后的策略动作（字符串，可为空），具体取值以产品约定为准。

---

## 创建 SmartTrade

### 接口定义

| 项目 | 说明 |
|------|------|
| **URL** | `/createSmartTrade` |
| **方法** | `POST` |
| **Content-Type** | `application/json` |
| **认证** | 需要在 Header 中携带 `Authorization: Bearer <API_KEY>` |

### 请求体参数

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| **mcp_token** | string | 是 | 用户 MCP 令牌，用于在服务端解析 `user_id`。 |
| **botname** | string | 是* | 机器人名称，用于展示；建议全局唯一，重复时可能返回业务错误。 |
| **stock_name** | string | 否 | 标的名称。 |
| **code** | string | 否 | 标的代码，如 `00700.HK`、`AAPL.US`。 |
| **frequency** | string | 否 | K 线/检查频率；常用取值见 [`common.md`](common.md) 中的 **frequency** 表。 |
| **trade_mode** | string | 否 | `buy_then_sell`（默认）或 `sell_only`。 |
| **price** | number / null | 否 | **buy_then_sell**：见上文。**sell_only**：**勿传**；成本价仅由账户自动查询写入，不由用户录入。 |
| **order_size** | object 或 number | 否 | **buy_then_sell**：为对象时使用 **`base_order_size`** 作为首笔买入股数。**sell_only**：可不传（由账户自动带出全仓）；若传，**`base_order_size`** 表示 Bot 管理的股数，宜为**小于账户该标的持仓**的部分头寸（见上文）。为对象或整数时 Bot 会规范为含 **`base_order_size`** 的对象。**`base_order_size` 宜为该标的 `lot_size` 的整数倍**；`lot_size` 由 **`/searchCode`** 返回（见 [`common.md`](common.md)）。 |
| **tp** | object | 否 | 止盈配置，字段见上文 **止盈（tp）**。 |
| **sl** | object | 否 | 止损配置，字段见上文 **止损（sl）**。 |
| **attitude** | object | 否 | 态度/分析配置。结构与 DCA/GRID 的 `attitude` 一致：`analysis_prompt_list`、`analysis_period`、`switch`、`controll_switch`。 |
| **binding_bot_id** | string | 否 | 关联的其它 Bot ID。 |
| **binding_bot_name** | string | 否 | 关联的 Bot 名称。 |

\* 若 `botname` 为空或未校验通过，可能导致创建失败，以 Bot 服务返回为准。

**说明**：不要传 `type`（固定为 SmartTrade）、`bot_id`（由服务端生成）；`user_id` 由服务端根据 `mcp_token` 填入转发请求。

#### tp / sl JSON 示例

```json
{
  "tp": {
    "tp_switch": true,
    "tp_mode": "dynamic",
    "tp_dynamic_index": "SAR",
    "tp_dynamic_factor": 1,
    "fix_tp": 5.0,
    "profit_trailing": false,
    "profit_trailing_deviation": 1.0
  },
  "sl": {
    "sl_switch": true,
    "sl_mode": "dynamic",
    "sl_dynamic_index": "SAR",
    "fix_sl": 2.0,
    "stop_loss_trailing": false,
    "stop_loss_trailing_deviation": 1.0
  }
}
```

#### order_size 示例

若该标的 **`lot_size`** 为 100，则 **`base_order_size`** 取 100、200、300 等整数倍均可；勿随意填无法整除每手的股数，以免券商拒单。**`sell_only`** 且需**部分头寸**时：在账户持仓大于所填股数的前提下，填入**小于全仓**的 **`base_order_size`**；全仓由账户自动带出时**可不传 `order_size`**。

```json
{ "base_order_size": 100 }
```

### 响应说明

- **成功**：`code === 100`，HTTP 200；`message` 为 Bot 返回的说明字符串（如创建并下单成功、sell_only 已从账户初始化持仓等）。
- **业务错误**：`code` 为 101、102、103、105 等时，HTTP **400**（如机器人名额不足、未绑定交易账号、参数非法、下单失败导致回滚未创建等）。
- **无效 mcp_token**：`code: 102`，HTTP **401**。
- **调用 Bot 服务失败**：HTTP **502**，`code` 多为 502。

### 请求示例

```bash
curl -X POST "http://localhost:3120/createSmartTrade" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <API_KEY>" \
  -d "{\"mcp_token\":\"mcp_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx\",\"botname\":\"腾讯SmartTrade\",\"stock_name\":\"腾讯控股\",\"code\":\"00700.HK\",\"frequency\":\"60m\",\"trade_mode\":\"sell_only\",\"order_size\":{\"base_order_size\":100},\"tp\":{\"tp_switch\":true,\"tp_mode\":\"fix\",\"fix_tp\":5,\"profit_trailing\":true,\"profit_trailing_deviation\":1},\"sl\":{\"sl_switch\":true,\"sl_mode\":\"fix\",\"fix_sl\":3}}"
```

### 与 Bot 服务的关系

MCP 将 `user_id` 与参数转发至 **`POST /createBot`**，`bot_type` 为 **`SmartTrade`**。创建逻辑、名额校验、下单与调度均在 **botAPIServer** 中完成。

---

## 修改 SmartTrade

修改前请务必阅读上文 **「修改限制与机器人状态（trade_info.status）」**：**buy_then_sell** 仅在 **status = 1** 时可改 **price** / **order_size**；**sell_only** 下**禁止**在请求中传 **price**。

### 接口定义

| 项目 | 说明 |
|------|------|
| **URL** | `/updateSmartTrade` |
| **方法** | `POST` |
| **Content-Type** | `application/json` |
| **认证** | `Authorization: Bearer <API_KEY>` |

### 请求体参数

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| **mcp_token** | string | 是 | 用户 MCP 令牌。 |
| **bot_id** | string | 是 | 要修改的 SmartTrade ID（即 `trade_bot._id` / 与 `trade_info.bot_id` 一致）。 |
| **botname** | string | 否 | 机器人名称。 |
| **stock_name** | string | 否 | 标的名称。 |
| **code** | string | 否 | 标的代码。 |
| **frequency** | string | 否 | 检查频率。 |
| **trade_mode** | string | 否 | `buy_then_sell` 或 `sell_only`；合并后决定适用上表限制。 |
| **price** | number | 否 | **buy_then_sell**：仅 **status = 1** 时可传。**sell_only**：**请勿传**（服务端会拒绝）。 |
| **order_size** | object 或 number | 否 | **buy_then_sell**：仅 **status = 1** 时可传。**sell_only**：可传以同步管理的股数；可为**小于账户该标的持仓**的部分头寸；为数字时会规范为含 `base_order_size` 的对象。**`base_order_size` 仍宜为 `lot_size` 的整数倍**。 |
| **tp** | object | 否 | 止盈配置。 |
| **sl** | object | 否 | 止损配置。 |
| **attitude** | object | 否 | 态度/分析配置。可更新 `analysis_prompt_list`、`analysis_period`、`switch`、`controll_switch`。 |
| **binding_bot_id** | string | 否 | 关联 Bot ID。 |
| **binding_bot_name** | string | 否 | 关联 Bot 名称。 |

仅传需要修改的字段即可。

### 响应说明

- **成功**：`code === 100`，HTTP 200。
- **业务错误**：`code` 为 101、102、103、104 等，HTTP **400**（常见：买入成交后仍改价/头寸；或 **sell_only** 下传入 **price**）。
- **无效 mcp_token**：`code: 102`，HTTP **401**。

转发至 **`POST /editBot`**（**`bot_type`**: **`SmartTrade`**）。调度器中的任务会在修改后更新。

---

## 删除 SmartTrade

### 接口定义

| 项目 | 说明 |
|------|------|
| **URL** | `/deleteSmartTrade` |
| **方法** | `POST` |
| **Content-Type** | `application/json` |
| **认证** | `Authorization: Bearer <API_KEY>` |

### 请求体参数

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| **mcp_token** | string | 是 | 用户 MCP 令牌。 |
| **bot_id** | string | 是 | 要删除的 SmartTrade ID。 |

### 响应说明

- **成功**：`code === 100`，HTTP 200。
- **业务错误**：`code` 为 102、103、104、105 等，HTTP **400**。
- **无效 mcp_token**：`code: 102`，HTTP **401**。

转发至 **`POST /deleteBot`**（**`bot_type`**: **`SmartTrade`**）。

---

## 获取所有 SmartTrade

### 接口定义

| 项目 | 说明 |
|------|------|
| **URL** | `/getAllSmartTrades` |
| **方法** | `POST` |
| **Content-Type** | `application/json` |
| **认证** | `Authorization: Bearer <API_KEY>` |

### 请求体参数

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| **mcp_token** | string | 是 | 用户 MCP 令牌。 |
| **code** | string | 否 | 若传入，仅返回该标的代码对应的 SmartTrade（按 `trade_bot.code` 筛选）。 |

### 响应说明

- **成功**：HTTP 200，`code === 100`，**`data`** 为数组。每条记录由 **`trade_info`** 与 **`trade_bot`** 合并而成，并与 **getUserBot** 中 SmartTrade 列表形态一致，主要字段包括：
  - **trade_bot_id**：`trade_info` 文档 `_id` 字符串。
  - **bot_id**：与 `trade_bot._id` 一致。
  - **bot_switch**：来自 `trade_info.switch`（字符串形式的 `"True"` / `"False"`）。
  - **user_id**：用户 ID 字符串。
  - **type**：`SmartTrade`。
  - **botname**、**code**、**stock_name**、**frequency**、**trade_mode**、**price**、**tp**、**sl**、**order_size**、**binding_bot_id**、**binding_bot_name** 等：来自参数库 `trade_bot`。
  - **status**、**notice_switch**、**position**、**trailing_active**、**trailing_high_price**、**trailing_type** 等：来自 `trade_info`。
  - 若存在态度分析且开启，**attitude** 中可能附带最新 **attitude** 文本与 **date**。

- **无效 mcp_token**：`code: 102`，HTTP **401**。

### 响应示例（节选）

```json
{
  "code": 100,
  "message": "success",
  "data": [
    {
      "trade_bot_id": "67175fa4d6f9d6cf92bc5dd8",
      "bot_id": "67175fa4d6f9d6cf92bc5dd7",
      "user_id": "6366170502d5c175fd586fe8",
      "bot_switch": "True",
      "type": "SmartTrade",
      "botname": "示例SmartTrade",
      "code": "00700.HK",
      "stock_name": "腾讯控股",
      "frequency": "60m",
      "trade_mode": "sell_only",
      "price": 380.0,
      "order_size": { "base_order_size": 100 },
      "tp": { "tp_switch": true, "tp_mode": "fix", "fix_tp": 5.0 },
      "sl": { "sl_switch": true, "sl_mode": "fix", "fix_sl": 2.0 },
      "status": 2,
      "position": {
        "opt": "hold",
        "price": 380.0,
        "qty": 100
      }
    }
  ]
}
```

示例中 **`sell_only`** 的 **`price`** 与 **`position.price`** 表示由**账户同步的成本价**，并非用户在创建请求中填写。

---

## 获取 SmartTrade 运行日志

与 Bot 服务 **`POST /getSmartTradeLog`** 语义一致：返回**交易运行快照**（**`log`**）及**当前持仓情况**（**`info`**，来自 **`trade_info.position`**）。MCP 校验 **`bot_id`** 属于当前用户后直读 **`trade_log`**、**`trade_info`**；**不提供日志筛选类请求参数**，查询与返回口径固定为全量（最多 **100** 条 **`log`**，与库查询条件一致）。

### 接口定义

| 项目 | 说明 |
|------|------|
| **URL** | `/getSmartTradeLog` |
| **方法** | `POST` |
| **Content-Type** | `application/json` |
| **认证** | `Authorization: Bearer <API_KEY>` |

### 请求体参数

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| **mcp_token** | string | 是 | 用户 MCP 令牌。 |
| **bot_id** | string | 是 | SmartTrade 机器人 ID（与 **`getAllSmartTrades`** 返回的 **`bot_id`** 一致）。 |

### 响应说明

**成功（HTTP 200）**：`code === 100`，`message` 一般为 `"success"`，业务载荷在 **`data`** 中（与 Bot 服务返回的 **`info` + `log`** 一致，外加 MCP 外层 **`code` / `message`**）。

#### `data` 整体结构

| 字段 | 类型 | 说明 |
|------|------|------|
| **info** | object | **当前持仓情况**：来自 **`trade_info.position`** 的汇总（股数、成本价、浮动盈亏等）；无运行信息或字段缺失时各数值可能为 **`0`**，或整体接近空对象。 |
| **log** | array | 交易运行快照列表，**按 `time` 从新到旧**，**最多 100 条**。仅包含有止盈/止损、跟踪、订单状态或 **`next_opt`** 等有效信息的记录（与库查询条件一致）。 |

#### `data.info`（当前持仓情况）

与 **`getAllSmartTrades`** 列表里合并的 **`position`** 口径一致，为**时点汇总**（非历史序列）：

| 字段 | 类型 | 说明 |
|------|------|------|
| **qty** | number | **持仓头寸**（股数）。 |
| **price** | number | 持仓成本价或与展示一致的价格（**`sell_only`** 下多与账户同步成本一致）。 |
| **pl_val** | number | 浮动盈亏金额。 |
| **pl_ratio** | number | 浮动盈亏比例（百分比数值，如 `-20.42` 表示约 -20.42%）。 |

#### `data.log[]`（单条日志）

每条对应 **`trade_log`** 中满足查询条件的一条文档，字段由该条 **`log`** 子文档与 **`time`** 组装；**`type`** 固定为 **`SmartTrade`**，用于与其它机器人类型区分。

| 字段 | 类型 | 说明 |
|------|------|------|
| **time** | string | 该条日志时间；MCP 可能对 `datetime` 序列化为 **`YYYY-MM-DD HH:MM:SS`** 或 ISO 8601，以实际返回为准。 |
| **next_opt** | string | 本周期策略给出的下一步意图，未写入时默认为 **`hold`**。 |
| **trailing** | object | 跟踪止盈/止损等动态状态（未启用时可能为空对象 **`{}`**）。 |
| **tp_sl** | object | 止盈/止损相关价格与状态（与创建时 **`tp` / `sl`** 配置对应的运行时快照）。 |
| **position** | object | 该快照下的**持仓对象**：见下表。 |
| **type** | string | 固定 **`SmartTrade`**。 |

**`data.log[].position` 常见子字段**（同一对象内一并描述**持仓数量**、**成本价**、**仓位操作**、**订单标识与状态**及策略附加字段）：

| 子字段 | 类型 | 说明 |
|--------|------|------|
| **qty** | number | **持仓头寸**（股数）。 |
| **price** | number | **成本价**（或与该快照一致的持仓均价）。 |
| **opt** | string | **仓位操作**侧状态或意图（如 **`hold`** 等），表示策略对该头寸的操作判定。 |
| **order_id** | string | 当前关联的 **订单 ID**；无订单时可为空字符串。 |
| **order_status** | string | **订单状态**；无在途订单时可为空字符串。 |
| **order_time** | string | **订单时间**（若有）。 |
| **pl_val** / **pl_ratio** | number | 该快照下的浮动盈亏金额 / 比例（若策略写入）。 |
| **can_sell_qty** | number | **当前可卖头寸**（可卖股数）。 |

**说明**：**`info`** 表示**当前**持仓汇总；**`log`** 为**历史快照**，用于复盘各周期的 **`next_opt`**、止盈止损与订单侧状态。相邻两条 **`time`** 的间隔取决于 **`frequency`** 与交易日历。

#### 响应体示例（节选）

```json
{
  "code": 100,
  "message": "success",
  "data": {
    "info": {
      "qty": 100.0,
      "price": 380.0,
      "pl_val": -500.0,
      "pl_ratio": -1.3
    },
    "log": [
      {
        "time": "2026-03-27 16:00:00",
        "next_opt": "hold",
        "trailing": {},
        "tp_sl": {},
        "position": {
          "opt": "hold",
          "qty": 100,
          "price": 380.0,
          "order_id": "",
          "order_status": ""
        },
        "type": "SmartTrade"
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

### 请求示例

```bash
curl -X POST "http://localhost:3120/getSmartTradeLog" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <API_KEY>" \
  -d '{"mcp_token": "mcp_xxx", "bot_id": "<BOT_ID>"}'
```

---

## 数据结构溯源

SmartTrade 参数模板定义在仓库 **`Constants/Basic_bot.py`** 中的 **`SmartTrade`** 与 **`SmartTrade_Info`**。运行时写入集合 **`trade_bot`**（参数）与 **`trade_info`**（状态与持仓）。**修改时的价格/头寸限制**见 **`botAPIServer.editSmartTrade`**。若本文与线上数据或 Bot 服务行为不一致，以 **botAPIServer** 实际逻辑为准。
