# MCP API：SmartReminder 接口说明

## 概述

本文档描述通过 MCP（Skills）对 **Smart 交易提醒机器人**（`bot_type: SmartReminder`）的**创建、修改、删除、获取列表与运行日志**接口；与 **SmartTrade 交易机器人**（`SmartTrade`）为不同产品线。分类与命名见 `[common.md](./common.md)`「机器人分类与命名」。调用方不传 `user_id`，改为传入 `mcp_token`，由服务端根据 `mcp_token` 解析出对应用户后再调用 Bot 服务对应逻辑。

- **基础路径**：GeeGooBot mcp-api 根地址（默认示例：`http://127.0.0.1:3120`）
- **认证方式**：请求头 `Authorization: Bearer <API_KEY>`

**公共约定**：`**mcp_token`**、`**frequency**` 共用枚举、信号与技术分析索引等，见 `[common.md](./common.md)`。

**说明**：SmartReminder 仅支持 **sell_only** 模式（持仓提醒），创建时**必须**提供成本价（`price`）和头寸（`qty`），用于止盈/止损及跟踪提醒。

---

## 止盈止损参数逻辑

SmartReminder 在持仓过程中按配置持续监测价格，达到止盈或止损条件时触发提醒或平仓。`tp`、`sl` 的**定义与设置逻辑**如下。

**默认约定**：动态止盈/止损默认使用 **SAR** 指标（`tp_dynamic_index` / `sl_dynamic_index` 为 `"SAR"`）；止盈跟踪与止损跟踪**默认开启**（`profit_trailing`、`stop_loss_trailing` 为 `true`），跟踪幅度**默认 1%**（`profit_trailing_deviation`、`stop_loss_trailing_deviation` 为 `1`）。

### 止盈（tp）逻辑

- **开关**（`tp_switch`）：为 `true` 时启用止盈；达到止盈条件后触发。
- **模式**：二选一。
  - **动态止盈**（`tp_mode: "dynamic"`）：以**高于当前价格**的指标曲线为触发价，价格上穿该曲线即触发。需指定 `tp_dynamic_index`，主要枚举值为 `"SAR"`、`"BBAND"`（布林线）；`tp_dynamic_factor` 为在指标价之上的线性系数，系数越大止盈价越高，止盈点随行情动态变化。
  - **固定止盈**（`tp_mode: "fix"`）：以成本价为基准，涨幅达到 `fix_tp`（百分比）即触发。例如 `fix_tp: 5` 表示涨 5% 止盈。
- **止盈跟踪**（`profit_trailing: true`）：触发止盈后不立即卖出，继续持有；从**触发后的最高价**起算，当回落幅度达到 `profit_trailing_deviation`（百分比，如 1）时再卖出。用于在趋势行情中争取更高收益，但会承受自高点回落的利润回吐。

### 止损（sl）逻辑

- **开关**（`sl_switch`）：为 `true` 时启用止损；达到止损条件后触发。
- **模式**：二选一。
  - **动态止损**（`sl_mode: "dynamic"`）：以**低于当前价格**的指标曲线为触发价，价格下穿该曲线即触发。需指定 `sl_dynamic_index`，止损点随行情动态变化。
  - **固定止损**（`sl_mode: "fix"`）：以成本价为基准，跌幅达到 `fix_sl`（百分比）即触发。例如 `fix_sl: 3` 表示跌 3% 止损。
- **止损后动作**（`stop_loss_action`）：触发止损并卖出后的行为，如：仅关闭当前交易（机器人继续运行、可再次买入）或关闭当前机器人（不再交易）。具体取值以服务端约定为准。
- **止损跟踪**（`stop_loss_trailing: true`）：触发止损后不立即卖出，从**触发后的最高价**起算，当回落幅度达到 `stop_loss_trailing_deviation`（百分比）时再卖出。用于在下跌中保留反弹空间，但会承受继续下跌带来的更大亏损风险。

---

## 创建 SmartReminder

### 接口定义

| 项目               | 说明                                               |
| ---------------- | ------------------------------------------------ |
| **URL**          | `/createSmartReminder`                           |
| **方法**           | `POST`                                           |
| **Content-Type** | `application/json`                               |
| **认证**           | 需要在 Header 中携带 `Authorization: Bearer <API_KEY>` |

### 请求体参数

请求体为 JSON，字段说明如下（与 `Constants/Basic_reminder.py` 中 **Smart_Reminder** 结构一致）：

| 参数名                  | 类型      | 必填  | Basic_reminder 对应                 | 说明                                                                           |
| -------------------- | ------- | --- | --------------------------------- | ---------------------------------------------------------------------------- |
| **mcp_token**        | string  | 是   | —（仅 MCP 层）                        | 用户 MCP 令牌，用于在服务端解析 `user_id`，不传则返回 400。                                      |
| **botname**          | string  | 是   | `Smart_Reminder.botname`          | 机器人名称，用于展示与唯一性校验（同用户下不可重复）。                                                  |
| **stock_name**       | string  | 否   | `Smart_Reminder.stock_name`       | 标的名称。                                                                        |
| **code**             | string  | 否   | `Smart_Reminder.code`             | 标的代码，如 `518880.SH`、`000858.SZ`。                                              |
| **frequency**        | string  | 否   | `Smart_Reminder.frequency`        | 检查频率，不传时默认为 `60m`；共用枚举见 `[common.md](./common.md)`。          |
| **tp**               | object  | 否   | `Smart_Reminder.tp`               | 止盈配置，见下方 **tp 结构说明**。                                                        |
| **sl**               | object  | 否   | `Smart_Reminder.sl`               | 止损配置，见下方 **sl 结构说明**。                                                        |
| **attitude**         | object  | 否   | `Smart_Reminder.attitude`         | 态度/分析配置：`analysis_prompt_list`、`analysis_period`、`switch`、`controll_switch`。 |
| **binding_bot_id**   | string  | 否   | `Smart_Reminder.binding_bot_id`   | 关联的 Bot ID。                                                                  |
| **binding_bot_name** | string  | 否   | `Smart_Reminder.binding_bot_name` | 关联的 Bot 名称。                                                                  |
| **price**            | number  | 是   | —（创建时必填）                          | 成本价（持仓成本），必须大于 0。                                                            |
| **qty**              | integer | 是   | —（创建时必填）                          | 头寸（持仓数量），必须大于 0。                                                             |

**说明**：创建接口仅接受上表参数。`type`（固定为 `SmartReminder`）、`reminder_id`（即 `_id`）、`user_id` 由服务端生成或填充，请勿在请求体中传入。**创建时必须提供 `price` 和 `qty`**，否则返回 101 业务错误。**SmartReminder 默认检查频率为 60 分钟**（`frequency` 未传时使用 `60m`）。

#### tp 结构说明（止盈）

与 `Basic_reminder.py` 中 Smart_Reminder 的 tp 一致：

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

| 字段                            | 类型      | 说明                                                                                                 |
| ----------------------------- | ------- | -------------------------------------------------------------------------------------------------- |
| **tp_switch**                 | boolean | **止盈开关**。为 `true` 时开启止盈，达到止盈条件后触发止盈（自动卖出或提醒）。                                                      |
| **tp_mode**                   | string  | 止盈模式：`"dynamic"` 动态止盈（默认），`"fix"` 固定止盈。                                                            |
| **tp_dynamic_index**          | string  | **动态止盈**时使用的指标，主要枚举：`"SAR"`（默认）、`"BBAND"`（布林线）；价格触及该指标价即触发。空字符串表示未选或使用固定止盈。                        |
| **tp_dynamic_factor**         | number  | **动态止盈系数**。在标准指标止盈价之上按该系数线性上移，使止盈点随系数提高。                                                           |
| **fix_tp**                    | number  | **固定止盈**涨幅百分比。例如 5 表示买入后上涨 5% 即触发止盈。                                                               |
| **profit_trailing**           | boolean | **止盈跟踪**开关。为 `true` 时（默认开启），触发止盈后不立即卖出，持续持有直至价格自「触发后的最高点」回落达到 `profit_trailing_deviation` 所设幅度再卖出。 |
| **profit_trailing_deviation** | number  | **止盈跟踪幅度**（百分比）。默认 `1`，表示自最高点回落 1% 时卖出。                                                            |

#### sl 结构说明（止损）

与 `Basic_reminder.py` 中 Smart_Reminder 的 sl 一致：

```json
{
  "sl_switch": true,
  "sl_mode": "dynamic",
  "sl_dynamic_index": "SAR",
  "fix_sl": 3.0,
  "stop_loss_trailing": true,
  "stop_loss_trailing_deviation": 1,
  "stop_loss_action": ""
}
```

| 字段                               | 类型      | 说明                                                                                                            |
| -------------------------------- | ------- | ------------------------------------------------------------------------------------------------------------- |
| **sl_switch**                    | boolean | **止损开关**。为 `true` 时开启止损，达到止损条件后触发止损（自动卖出或提醒）。                                                                 |
| **sl_mode**                      | string  | 止损模式：`"dynamic"` 动态止损（默认），`"fix"` 固定止损。                                                                       |
| **sl_dynamic_index**             | string  | **动态止损**时使用的指标，主要枚举：`"SAR"`（默认）、`"BBAND"`（布林线）；价格触及该指标价即触发。空字符串表示未选或使用固定止损。                                   |
| **fix_sl**                       | number  | **固定止损**跌幅百分比。例如 3 表示买入后下跌 3% 即触发止损。                                                                          |
| **stop_loss_trailing**           | boolean | **止损跟踪**开关。为 `true` 时（默认开启），触发止损后不立即卖出，持续持有直至价格自「触发后的最高点」回落达到 `stop_loss_trailing_deviation` 所设幅度再卖出，以保留反弹空间。 |
| **stop_loss_trailing_deviation** | number  | **止损跟踪幅度**（百分比）。默认 `1`，表示自最高点回落 1% 时卖出。                                                                       |
| **stop_loss_action**             | string  | **止损后动作**。触发止损并卖出后的行为（如仅关闭当前交易或关闭当前机器人），具体取值以服务端约定为准，可为空字符串。                                                  |

#### 完整 SmartReminder 示例（创建后 / 获取列表单条结构参考）

以下为创建成功后或「获取所有 SmartReminder」列表中单条记录的形态（含服务端生成字段与 `smart_reminder_info` 合并后的结构），供 Skills 开发时理解各字段含义。**创建时请求体只需传上面表格中的参数，无需传 `reminder_id`、`type`、`user_id` 及只读字段。**

```json
{
  "type": "SmartReminder",
  "user_id": "6366170502d5c175fd586fe8",
  "reminder_id": "6975fc7b1bb491d131ca19b7",
  "bot_id": "6975fc7b1bb491d131ca19b7",
  "botname": "腾讯控股提醒机器人",
  "stock_name": "腾讯控股",
  "code": "00700.HK",
  "frequency": "3m",
  "reminder_switch": "False",
  "switch": true,
  "notice_switch": true,
  "status": 2,
  "trailing_active": false,
  "trailing_high_price": 0,
  "trailing_type": "default",
  "binding_bot_id": "",
  "binding_bot_name": "",
  "price": 595.0,
  "qty": 100,
  "tp": {
    "tp_switch": true,
    "tp_mode": "fix",
    "tp_dynamic_index": "SAR",
    "tp_dynamic_factor": 1.0,
    "fix_tp": 5.0,
    "profit_trailing": false,
    "profit_trailing_deviation": 1.0
  },
  "sl": {
    "sl_switch": true,
    "sl_mode": "fix",
    "sl_dynamic_index": "SAR",
    "fix_sl": 3.0,
    "stop_loss_trailing": true,
    "stop_loss_trailing_deviation": 1.0,
    "stop_loss_action": ""
  },
  "position": {
    "opt": "hold",
    "order_id": "",
    "order_status": "",
    "order_time": "2026-01-27 13:09:03",
    "price": 595.0,
    "qty": 100,
    "can_sell_qty": 100,
    "pl_val": -2200.0,
    "pl_ratio": -3.7
  }
}
```

**字段说明**：

- **reminder_id** / **bot_id**：由服务端生成的提醒 ID（对应库中 `smart_reminder._id`），仅响应/查询时返回。
- **type**：固定为 `SmartReminder`，由服务端写入。
- **user_id**：由服务端根据 `mcp_token` 解析后写入。
- **reminder_switch**：来自 `smart_reminder_info.switch`，为字符串 `"True"` / `"False"`，表示提醒总开关。
- **switch**：与 `reminder_switch` 同源（布尔或字符串），内部使用。
- **notice_switch**：是否开启通知（来自 `smart_reminder_info`）。
- **status**：状态（如 2 表示持仓中），来自 `smart_reminder_info`。
- **trailing_active** / **trailing_high_price** / **trailing_type**：跟踪止盈/止损状态，来自 `smart_reminder_info`。
- **price** / **qty**（根级别）：创建时传入的成本价与头寸，获取列表时与 `position` 内一致。
- **position**：持仓信息（来自 `smart_reminder_info.position`），创建时由 `price`、`qty` 初始化；包含 `opt`、`order_id`、`order_status`、`**order_time`（订单时间）**、`price`、`qty`、`**can_sell_qty`（当前可卖头寸）**、`pl_val`、`pl_ratio` 等。

---

### 响应说明

- **成功**：`code === 100`，表示创建 SmartReminder 成功。
- **业务错误**：`code` 为 101、102、103、105 等时表示业务校验失败（如 reminder 已存在、用户不存在、未绑定交易账号、未提供 price/qty、提醒机器人数量不足等），HTTP 状态码为 400。
- **mcp_token 无效**：未找到对应用户时返回 `code: 102`，HTTP 401。
- **调用 Bot 服务失败**：返回 502 及相应错误信息。

响应体示例（成功）：

```json
{
  "code": 100,
  "message": "创建SmartReminder成功，已使用用户提供的持仓信息（成本价: 4.5, 头寸: 1000）"
}
```

响应体示例（reminder 已存在）：

```json
{
  "code": 101,
  "message": "reminder 已存在"
}
```

响应体示例（缺少 price/qty）：

```json
{
  "code": 101,
  "message": "SmartReminder创建失败，必须提供成本价(price)和头寸(qty)参数"
}
```

---

### 请求示例

```bash
curl -X POST "http://localhost:3120/createSmartReminder" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <API_KEY>" \
  -d '{"mcp_token":"mcp_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx","botname":"黄金ETF止盈止损提醒","stock_name":"黄金ETF","code":"518880.SH","frequency":"5m","price":4.5,"qty":1000,"tp":{"tp_switch":true,"tp_mode":"fix","fix_tp":5,"profit_trailing":true,"profit_trailing_deviation":1},"sl":{"sl_switch":true,"sl_mode":"fix","fix_sl":2,"stop_loss_trailing":false,"stop_loss_trailing_deviation":1}}'
```

---

### 与 Bot 服务的关系

- 本接口只做两件事：用 `mcp_token` 解析 `user_id`，并将请求参数转发至 Bot 服务的 `POST /createBot`（`bot_type: "SmartReminder"`）。
- 创建逻辑、数量与权限校验、持仓初始化（price/qty）、调度与通知等均在 Bot 服务（botAPIServer）中完成。
- 默认 Bot 服务地址由 `Config/APIConnection.py` 中 `--bot_server_ip`、`--bot_server_port` 决定（默认 `http://127.0.0.1:3230`）。

---

## 修改 SmartReminder

### 接口定义

| 项目               | 说明                                               |
| ---------------- | ------------------------------------------------ |
| **URL**          | `/updateSmartReminder`                           |
| **方法**           | `POST`                                           |
| **Content-Type** | `application/json`                               |
| **认证**           | 需要在 Header 中携带 `Authorization: Bearer <API_KEY>` |

### 请求体参数

| 参数名                  | 类型             | 必填  | 说明                                                                                                 |
| -------------------- | -------------- | --- | -------------------------------------------------------------------------------------------------- |
| **mcp_token**        | string         | 是   | 用户 MCP 令牌，用于在服务端解析 `user_id`。                                                                      |
| **reminder_id**      | string         | 是   | 要修改的 Smart Reminder ID（即 Bot 侧 `bot_id` / 库中 `smart_reminder._id`），可从「获取所有 SmartReminder」或创建响应中取得。 |
| **botname**          | string         | 否   | 机器人名称。                                                                                             |
| **stock_name**       | string         | 否   | 标的名称。                                                                                              |
| **code**             | string         | 否   | 标的代码。                                                                                              |
| **frequency**        | string         | 否   | 检查频率（如 `3m`、`5m`、`60m`；默认 `60m`）。                                                                  |
| **tp**               | object         | 否   | 止盈配置，结构同创建接口的 **tp 结构说明**。                                                                         |
| **sl**               | object         | 否   | 止损配置，结构同创建接口的 **sl 结构说明**。                                                                         |
| **attitude**         | object         | 否   | 态度/分析配置。可更新 `analysis_prompt_list`、`analysis_period`、`switch`、`controll_switch`。                   |
| **binding_bot_id**   | string         | 否   | 关联的 Bot ID。                                                                                        |
| **binding_bot_name** | string         | 否   | 关联的 Bot 名称。                                                                                        |
| **price**            | number         | 否   | 成本价；传入则更新持仓成本。                                                                                     |
| **qty**              | integer        | 否   | 头寸；传入则更新持仓数量。                                                                                      |
| **switch**           | boolean/string | 否   | 提醒总开关，如 `true` / `"True"`；写入库时为 `smart_reminder_info.switch`。                                      |

**说明**：仅传需要修改的字段即可，未传字段保持原值。

### 响应说明

- **成功**：`code === 100`，表示更新 SmartReminder 配置成功。
- **业务错误**：`code` 为 101、102、103、104 等时表示未找到 Reminder、用户不存在、名称重复等，HTTP 状态码为 400。
- **mcp_token 无效**：未找到对应用户时返回 `code: 102`，HTTP 401。
- **调用 Bot 服务失败**：返回 502 及相应错误信息。

响应体示例（成功）：

```json
{
  "code": 100,
  "message": "更新SmartReminder配置成功"
}
```

### 请求示例

```bash
curl -X POST "http://localhost:3120/updateSmartReminder" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <API_KEY>" \
  -d '{
    "mcp_token": "mcp_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
    "reminder_id": "67175fa4d6f9d6cf92bc5dd8",
    "frequency": "10m",
    "switch": true
  }'
```

### 与 Bot 服务的关系

- 用 `mcp_token` 解析 `user_id`，将 `reminder_id` 及可更新字段转发至 Bot 服务的 `POST /editBot`（`bot_type: "SmartReminder"`，`bot_id` 即 `reminder_id`）。
- 修改逻辑与调度更新在 `botAPIServer` 的 `editSmartReminder(user_id, bot_id)` 中完成；若传 `price`、`qty` 会更新持仓信息。

---

## 删除 SmartReminder

### 接口定义

| 项目               | 说明                                               |
| ---------------- | ------------------------------------------------ |
| **URL**          | `/deleteSmartReminder`                           |
| **方法**           | `POST`                                           |
| **Content-Type** | `application/json`                               |
| **认证**           | 需要在 Header 中携带 `Authorization: Bearer <API_KEY>` |

### 请求体参数

| 参数名             | 类型     | 必填  | 说明                                        |
| --------------- | ------ | --- | ----------------------------------------- |
| **mcp_token**   | string | 是   | 用户 MCP 令牌。                                |
| **reminder_id** | string | 是   | 要删除的 Smart Reminder ID（即 Bot 侧 `bot_id`）。 |

### 响应说明

- **成功**：`code === 100`，表示删除成功。
- **业务错误**：`code` 为 102、103、104 等时表示未找到 Bot/用户或删除失败，HTTP 状态码为 400。
- **mcp_token 无效**：未找到对应用户时返回 `code: 102`，HTTP 401。
- **调用 Bot 服务失败**：返回 502。

响应体示例（成功）：

```json
{
  "code": 100,
  "message": "删除SmartReminder成功"
}
```

### 请求示例

```bash
curl -X POST "http://localhost:3120/deleteSmartReminder" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <API_KEY>" \
  -d '{
    "mcp_token": "mcp_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
    "reminder_id": "67175fa4d6f9d6cf92bc5dd8"
  }'
```

### 与 Bot 服务的关系

- 用 `mcp_token` 解析 `user_id`，将 `reminder_id` 转发至 Bot 服务的 `POST /deleteBot`（`bot_type: "SmartReminder"`）。
- 删除逻辑、调度移除、提醒数量扣减在 `botAPIServer` 的 `deleteSmartReminder(user_id, bot_id)` 中完成。

---

## 获取所有 SmartReminder

### 接口定义

| 项目               | 说明                                               |
| ---------------- | ------------------------------------------------ |
| **URL**          | `/getAllSmartReminders`                          |
| **方法**           | `POST`                                           |
| **Content-Type** | `application/json`                               |
| **认证**           | 需要在 Header 中携带 `Authorization: Bearer <API_KEY>` |

### 请求体参数

| 参数名           | 类型     | 必填  | 说明                                  |
| ------------- | ------ | --- | ----------------------------------- |
| **mcp_token** | string | 是   | 用户 MCP 令牌。                          |
| **code**      | string | 否   | 按标的代码筛选；不传则返回该用户下全部 Smart Reminder。 |

### 响应说明

- **成功**：`code === 100`，`message` 为 `"success"`，`data` 为当前用户（及可选 `code` 筛选下）的 Smart Reminder 列表。列表中每项结构见下方响应体示例（与 Bot 侧 `getUserReminder` 返回的 SmartReminder 一致）。
- **mcp_token 无效**：未找到对应用户时返回 `code: 102`，HTTP 401。

响应体示例（成功）：

```json
{
  "code": 100,
  "message": "success",
  "data": [
    {
      "binding_bot_id": "",
      "binding_bot_name": "",
      "bot_id": "6975fc7b1bb491d131ca19b7",
      "botname": "腾讯控股提醒机器人",
      "code": "00700.HK",
      "frequency": "3m",
      "notice_switch": true,
      "position": {
        "can_sell_qty": 100,
        "opt": "hold",
        "order_id": "",
        "order_status": "",
        "order_time": "2026-01-27 13:09:03",
        "pl_ratio": -3.7,
        "pl_val": -2200.0,
        "price": 595.0,
        "qty": 100
      },
      "price": 595.0,
      "qty": 100,
      "reminder_id": "6975fc7b1bb491d131ca19b7",
      "reminder_switch": "False",
      "sl": {
        "fix_sl": 3.0,
        "sl_dynamic_index": "SAR",
        "sl_mode": "fix",
        "sl_switch": true,
        "stop_loss_action": "",
        "stop_loss_trailing": true,
        "stop_loss_trailing_deviation": 1.0
      },
      "status": 2,
      "stock_name": "腾讯控股",
      "switch": true,
      "tp": {
        "fix_tp": 5.0,
        "profit_trailing": false,
        "profit_trailing_deviation": 1.0,
        "tp_dynamic_factor": 1.0,
        "tp_dynamic_index": "SAR",
        "tp_mode": "fix",
        "tp_switch": true
      },
      "trailing_active": false,
      "trailing_high_price": 0,
      "trailing_type": "default",
      "type": "SmartReminder",
      "user_id": "6366170502d5c175fd586fe8"
    }
  ]
}
```

### 请求示例

```bash
# 获取该用户全部 Smart Reminder
curl -X POST "http://localhost:3120/getAllSmartReminders" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <API_KEY>" \
  -d '{"mcp_token": "mcp_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"}'

# 仅获取指定标的的 Smart Reminder
curl -X POST "http://localhost:3120/getAllSmartReminders" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <API_KEY>" \
  -d '{"mcp_token": "mcp_xxx", "code": "518880.SH"}'
```

### 与 Bot 服务的关系

- 用 `mcp_token` 解析 `user_id`，MCP 层直读 `smart_reminder`、`smart_reminder_info` 及 `attitude_log`，组装逻辑与 Bot 服务 `getUserReminder` 的 **SmartReminder** 部分一致。
- 返回的 `reminder_id` 与 `bot_id` 均为 `smart_reminder._id`，可直接用于 `/updateSmartReminder`、`/deleteSmartReminder`。

---

## 获取 Smart 提醒运行日志（getSmartReminderLog）

按 `reminder_id` 查询该 Smart 提醒最近最多 **100** 条调度记录，并返回当前持仓摘要 `info`（与 Bot 服务 `POST /getSmartReminderLog` 一致）。MCP 层先用 `mcp_token` 解析 `user_id`，并校验该 `reminder_id` 属于当前用户后再读库。**MCP 不提供日志筛选类请求参数**，固定返回全量（最多 **100** 条 `log`）。

### 接口定义

| 项目               | 说明                                |
| ---------------- | --------------------------------- |
| **URL**          | `/getSmartReminderLog`            |
| **方法**           | `POST`                            |
| **Content-Type** | `application/json`                |
| **认证**           | `Authorization: Bearer <API_KEY>` |

### 请求体参数

| 参数名             | 类型     | 必填  | 说明                                                                                       |
| --------------- | ------ | --- | ---------------------------------------------------------------------------------------- |
| **mcp_token**   | string | 是   | 用户 MCP 令牌。                                                                               |
| **reminder_id** | string | 是   | 提醒 ID，即 `smart_reminder._id`，与 `getAllSmartReminders` 返回列表中每条记录的 `**reminder_id`** 字段一致。 |

### 响应说明

**成功（HTTP 200）**：`code === 100`，`message` 一般为 `"success"`，`**data` 为对象**，包含 `**info`** 与 `**log**` 两部分。

#### `info`（当前持仓摘要）

来自**当前** `smart_reminder_info.position` 的摘要，表示**拉取接口时**的持仓状态（不是历史某一时刻；与 `log` 里每条里的 `position` 可能因时间不同而不完全一致）。

| 字段           | 类型     | 说明                                |
| ------------ | ------ | --------------------------------- |
| **qty**      | number | 头寸数量（股）。                          |
| **price**    | number | 成本价 / 持仓参考价。                      |
| **pl_val**   | number | 浮动盈亏金额。                           |
| **pl_ratio** | number | 浮动盈亏比例（百分比数值，如 `-3.7` 表示约 -3.7%）。 |

#### `log`（调度快照列表）

- **条数与顺序**：最多 **100** 条；按 `**time` 从新到旧** 排列。
- **每条记录**含以下字段：

| 字段             | 类型     | 说明                                                                                                                                                                      |
| -------------- | ------ | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **time**       | string | 本条日志时间，常见为 `**YYYY-MM-DD HH:MM:SS`**。                                                                                                                                   |
| **next_opt**   | string | 本周期**策略/提醒**状态（见下方 **「`next_opt` 常见取值」**）。与 `**position.opt`** 不同：前者表示本次检查触发了哪类止盈止损或跟踪逻辑，后者表示持仓结构里的**订单类型**占位。                                                          |
| **position**   | object | 该快照时刻的**持仓与订单相关字段**，常见子字段见下表。                                                                                                                                           |
| **tp_sl**      | object | 止盈止损与现价参考，常见含 `**current_price`**（当时市价）、`**tp_price**`（止盈参考价）、`**sl_price**`（止损参考价）。                                                                                    |
| `**trailing**` | object | 跟踪止盈/止损等扩展状态；**无数据时多为空对象 `{}`**；开启跟踪后可能含 `**trailing_active**`、`**trailing_type**`、`**trailing_high_price**`、`**trailing_deviation**`、`**trailing_deviation_value**` 等。 |

`**next_opt` 常见取值**（与调度里止盈止损、跟踪开关组合有关，无持仓时取值常为 `**"hold"`** 字符串）：

| 取值                    | 含义（概要）                                                                                                                     |
| --------------------- | -------------------------------------------------------------------------------------------------------------------------- |
| `**"hold"**`          | 未触发新的止盈/止损/跟踪提醒，或无持仓跳过检查。                                                                                                  |
| **tp**                | 固定或动态止盈条件满足，且未走「止盈跟踪」分支时的止盈提醒。                                                                                             |
| **sl**                | 固定或动态止损条件满足，且未走「止损跟踪」分支时的止损提醒。                                                                                             |
| **tp_trailing_start** | 价格越过止盈参考后，准备进入止盈跟踪（`**profit_trailing`** 开启且此前未处于跟踪）。                                                                      |
| **sl_trailing_start** | 价格跌破止损参考后，准备进入止损跟踪（`**stop_loss_trailing`** 开启且此前未处于跟踪）。                                                                   |
| **{type}_trailing**   | 已在跟踪过程中，回撤超过允许偏差时触发提醒；`**type`** 与当前 `**trailing_type**` 一致，常见为 `**tp**` → `**tp_trailing**`、`**sl**` → `**sl_trailing**`。 |

`**position` 常见子字段**：

| 子字段                             | 类型     | 说明                                                                                                                                                                                                                               |
| ------------------------------- | ------ | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **qty**                         | number | 头寸数量。                                                                                                                                                                                                                            |
| **price**                       | number | 成本价。                                                                                                                                                                                                                             |
| **pl_val** / **pl_ratio**       | number | 该快照下的浮动盈亏金额 / 比例。                                                                                                                                                                                                                |
| **can_sell_qty**                | number | **当前可卖头寸**（可卖股数）。                                                                                                                                                                                                                |
| **opt**                         | string | 与 `Basic_reminder.Smart_Reminder_Info.position` 一致，表示**持仓结构中的订单/操作类型**字段（与交易类 Bot 的 `position` 同源）。**SmartReminder 仅提醒、不下单**，初始化持仓时写入 `**"hold"`**，正常运行时日志里 `**opt` 一般为 `"hold"**`；**多种策略含义请看同条记录的 `next_opt`**，勿与 `**opt**` 混淆。 |
| **order_id** / **order_status** | string | 委托编号与状态；无委托时常为空字符串。                                                                                                                                                                                                              |
| **order_time**                  | string | **订单时间**（若有）。                                                                                                                                                                                                                    |

**错误码**：`401` 缺参（未传 `mcp_token` 或 `reminder_id`）；`102` 无效令牌（HTTP 401）；`101` 非法 `reminder_id`；`103` 无权限或提醒不存在。

`**data` 结构示例**（节选一条 `log` 项示意）：

```json
{
  "info": {
    "qty": 100,
    "price": 595.0,
    "pl_val": -2200.0,
    "pl_ratio": -3.7
  },
  "log": [
    {
      "time": "2026-02-03 11:06:00",
      "next_opt": "sl_trailing_start",
      "position": {
        "qty": 100,
        "price": 595.0,
        "pl_val": -2200.0,
        "pl_ratio": -3.7,
        "can_sell_qty": 100,
        "opt": "hold",
        "order_id": "",
        "order_status": "",
        "order_time": "2026-01-27 13:09:03"
      },
      "tp_sl": {
        "current_price": 573.0,
        "sl_price": 577.15,
        "tp_price": 624.75
      },
      "trailing": {}
    }
  ]
}
```

### 请求示例

```bash
curl -X POST "http://localhost:3120/getSmartReminderLog" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <API_KEY>" \
  -d '{
    "mcp_token": "mcp_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
    "reminder_id": "6975fc7b1bb491d131ca19b7"
  }'
```

### 与 Bot 服务的关系

- Bot 侧：`POST /getSmartReminderLog`（请求体字段以 Bot 服务为准），返回 `info` + `log`。
- MCP 侧：请求体使用 `**reminder_id**`（含义与 Bot 的 `bot_id` 相同），与 `getAllSmartReminders` 对齐；归属校验后直读 `smart_reminder_log` 与 `smart_reminder_info`；成功响应为 `{ code, message, data }`，其中 `data` 即 `{ info, log }`。

---

## 参考

- **默认结构**：`Constants/Basic_reminder.py` 中的 `Smart_Reminder`（`botname`、`type`、`stock_name`、`code`、`frequency`、`switch`、`tp`、`sl`、`binding_bot_id`、`binding_bot_name`）及 `Smart_Reminder_Info`（`switch`、`position`、`trailing_active`、`trailing_type`、`trailing_high_price`、`notice_switch` 等）。
- **Bot 服务**：
  - 创建：`botAPIServer` 中 `createSmartReminder(user_id)` 及路由 `POST /createBot`（`bot_type: "SmartReminder"`）。
  - 修改：`editSmartReminder(user_id, bot_id)` 及 `POST /editBot`；开关：`POST /switchBot`（`bot_type: "SmartReminder"`）。
  - 删除：`deleteSmartReminder(user_id, bot_id)` 及 `POST /deleteBot`。
  - 获取：MCP 直读 `smart_reminder`、`smart_reminder_info`，与 `getUserReminder` 中 SmartReminder 列表结构一致。
- **MCP 入口**：`mcpAPIServer` 中 `POST /createSmartReminder`、`POST /updateSmartReminder`、`POST /deleteSmartReminder`、`POST /getAllSmartReminders`、`POST /getSmartReminderLog`，均通过 `mcp_token` 解析 `user_id`；创建/修改/删除转发至 Bot，列表与日志为 MCP 直读库。

