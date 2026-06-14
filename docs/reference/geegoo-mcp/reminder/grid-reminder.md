# MCP API：GRIDReminder 接口说明

## 概述

本文档描述通过 MCP（Skills）对 **GRID 网格提醒机器人**（`bot_type: GRIDReminder`）的**创建、修改、删除、获取列表与运行日志**接口。分类与命名见 [`common.md`](common.md)「机器人分类与命名」。调用方不传 `user_id`，改为传入 `mcp_token`，由服务端根据 `mcp_token` 解析出对应用户后再调用 Bot 服务对应逻辑。

- **基础路径**：geegoo mcp 根地址（默认示例：`http://0.0.0.0:5700`）
- **认证方式**：请求头 `Authorization: Bearer <API_KEY>`

**公共约定**：**`mcp_token`**、**`frequency`**（共用枚举见 [`common.md`](common.md)，GRID 专用取值见本文）；技术分析 **`prompt_id` / `period`**、**`attitude`** 见 [`agent-analyst.md`](../analyst/agent-analyst.md)。

---

## 创建 GRIDReminder

### 接口定义

| 项目 | 说明 |
|------|------|
| **URL** | `/createGRIDReminder` |
| **方法** | `POST` |
| **Content-Type** | `application/json` |
| **认证** | 需要在 Header 中携带 `Authorization: Bearer <API_KEY>` |

### 请求体参数

请求体为 JSON，字段说明如下（与 `Constants/Basic_reminder.py` 中 **Grid_Reminder** 结构一致）：

| 参数名 | 类型 | 必填 | Basic_reminder 对应 | 说明 |
|--------|------|------|---------------------|------|
| **mcp_token** | string | 是 | —（仅 MCP 层） | 用户 MCP 令牌，用于在服务端解析 `user_id`，不传则返回 400。 |
| **botname** | string | 是 | `Grid_Reminder.botname` | 机器人名称，用于展示与唯一性校验（同用户下不可重复）。 |
| **stock_name** | string | 否 | `Grid_Reminder.stock_name` | 标的名称。 |
| **code** | string | 否 | `Grid_Reminder.code` | 标的代码，如 `518880.SH`。 |
| **frequency** | string | 否 | `Grid_Reminder.frequency` | 检查频率，默认 `5m`。共用枚举见 [`common.md`](common.md)；GRID 调度专用取值见下方 **frequency 取值说明**。 |
| **grid** | object | 否 | `Grid_Reminder.grid` | 网格配置，见下方 **grid 结构说明**。 |
| **attitude** | object | 否 | `Grid_Reminder.attitude` | 态度/分析配置，见下方 **attitude 结构说明**。 |
| **name** | string | 否 | — | 可选名称字段，部分逻辑会使用。 |

**说明**：创建接口仅接受上表参数。`type`（固定为 `GRIDReminder`）、`reminder_id`（即 `_id`）、`user_id` 由服务端生成或填充，请勿在请求体中传入。**GRIDReminder 默认创建的时间周期为 5 分钟**（即 `frequency` 未传时默认为 `5m`）。创建后提醒开关默认开启（`Grid_Reminder_Info.switch` 默认为 `True`）。

#### frequency 取值说明

`frequency` 表示检查周期，调度逻辑见 `GRID/BasicGridBot.py` 中的 `getFrequency()`。支持取值如下：

| 取值 | 含义 | 调度间隔（分钟） |
|------|------|------------------|
| **5m** | 5 分钟 | 5 |
| **60m** | 60 分钟 | 60 |
| **daily** | 每日 | 1440 |

未在上述取值中或未传时：未传则使用默认 `5m`；若传入其它字符串，服务端会按 1 分钟间隔处理（不推荐）。

#### grid 结构说明

与 Grid 策略的网格参数一致：

```json
{
  "upper_limit_price": 180.0,
  "lower_limit_price": 100.0,
  "grid_num": 9
}
```

- **upper_limit_price**：网格上限价格（数值，可为整数或浮点数）。
- **lower_limit_price**：网格下限价格（数值，可为整数或浮点数）。
- **grid_num**：网格数量（整数）。

#### attitude 结构说明

与 `Basic_reminder.py` 中 Grid_Reminder 的 attitude 一致：

```json
{
  "analysis_prompt_list": ["664f5e464883b402278dc92c"],
  "analysis_period": "daily",
  "controll_switch": false,
  "switch": false
}
```

- **analysis_prompt_list**：分析用 prompt 的 ID 列表（字符串 ID 数组），可为空数组 `[]`。
- **analysis_period**：字符串，单项分析 / 态度刷新采用的周期；未传时默认 **`daily`**（与 **`Constants/Basic_reminder.py`** 模板一致）。含义见 [`agent-analyst.md`](../analyst/agent-analyst.md) **约定与枚举 · attitude.analysis_period**。
- **controll_switch**：是否开启态度控制开关。
- **switch**：态度功能总开关。

#### 完整 GRIDReminder 示例（创建后 / 获取列表单条结构参考）

以下为创建成功后或「获取所有 GRIDReminder」列表中单条记录的形态（含服务端生成字段与运行状态），供 Skills 开发时理解各字段含义。**创建时请求体只需传上面表格中的参数，无需传 `reminder_id`、`bot_id`、`type`、`user_id`、`buy_grid`、`sell_grid`、`current_grid`、`reminder_switch`。**

```json
{
  "attitude": {
    "analysis_prompt_list": ["664f5e464883b402278dc92c"],
    "analysis_period": "daily",
    "controll_switch": false,
    "switch": false
  },
  "bot_id": "671affe2d6f9d6cf92bc6120",
  "botname": "五粮液提醒器",
  "buy_grid": [100.0],
  "code": "000858.SZ",
  "current_grid": 110.0,
  "frequency": "5m",
  "grid": {
    "grid_num": 9,
    "lower_limit_price": 100.0,
    "upper_limit_price": 180.0
  },
  "reminder_id": "671affe2d6f9d6cf92bc6120",
  "reminder_switch": "True",
  "sell_grid": [180.0, 170.0, 160.0, 150.0, 140.0, 130.0, 120.0],
  "stock_name": "五粮液",
  "type": "GRIDReminder",
  "user_id": "6366170502d5c175fd586fe8"
}
```

- **reminder_id** / **bot_id**：由服务端生成的提醒 ID（对应库中 `grid_reminder._id`），仅响应/查询时返回；MCP 返回中二者一致，可直接用于 `/updateGRIDReminder`、`/deleteGRIDReminder`。
- **type**：固定为 `GRIDReminder`，由服务端写入。
- **user_id**：由服务端根据 `mcp_token` 解析后写入。
- **reminder_switch**：来自 `grid_reminder_info.switch`，获取列表或状态时返回。
- **buy_grid**：当前买入网格价位列表（来自 `grid_reminder_info`），运行时更新。
- **sell_grid**：当前卖出网格价位列表（来自 `grid_reminder_info`），运行时更新。
- **current_grid**：当前价格所在网格价位（来自 `grid_reminder_info`），运行时更新。

---

### 响应说明

- **成功**：`code === 100`，表示创建 GRID Reminder 成功。
- **业务错误**：`code` 为 101、102、103、105 等时表示业务校验失败（如 reminder 已存在、用户不存在、未绑定交易账号、提醒机器人数量不足等），HTTP 状态码为 400。
- **mcp_token 无效**：未找到对应用户时返回 `code: 102`，HTTP 401。
- **调用 Bot 服务失败**：返回 502 及相应错误信息。

响应体示例（成功）：

```json
{
  "code": 100,
  "message": "创建GridReminder成功"
}
```

响应体示例（reminder 已存在）：

```json
{
  "code": 101,
  "message": "reminder已存在"
}
```

---

### 请求示例

```bash
curl -X POST "http://localhost:5700/createGRIDReminder" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <API_KEY>" \
  -d '{"mcp_token":"mcp_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx","botname":"黄金网格提醒","stock_name":"黄金ETF","code":"518880.SH","frequency":"5m","grid":{"upper_limit_price":350,"lower_limit_price":300,"grid_num":6},"attitude":{"analysis_prompt_list":[],"analysis_period":"daily","switch":false,"controll_switch":false}}'
```

---

### 与 Bot 服务的关系

- 本接口只做两件事：用 `mcp_token` 解析 `user_id`，并将请求参数转发至 Bot 服务的 `POST /createBot`（`bot_type: "GRIDReminder"`）。
- 创建逻辑、数量与权限校验、调度与通知等均在 Bot 服务（botAPIServer）中完成。
- 默认 Bot 服务地址由 `Config/APIConnection.py` 中 `--bot_server_ip`、`--bot_server_port` 决定（默认 `http://127.0.0.1:5600`）。

---

## 修改 GRIDReminder

### 接口定义

| 项目 | 说明 |
|------|------|
| **URL** | `/updateGRIDReminder` |
| **方法** | `POST` |
| **Content-Type** | `application/json` |
| **认证** | 需要在 Header 中携带 `Authorization: Bearer <API_KEY>` |

### 请求体参数

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| **mcp_token** | string | 是 | 用户 MCP 令牌，用于在服务端解析 `user_id`。 |
| **reminder_id** | string | 是 | 要修改的 GRID Reminder ID（即 Bot 侧 `bot_id` / 库中 `grid_reminder._id`），可从「获取所有 GRIDReminder」或创建响应中取得。 |
| **botname** | string | 否 | 机器人名称。 |
| **name** | string | 否 | 可选名称。 |
| **stock_name** | string | 否 | 标的名称。 |
| **code** | string | 否 | 标的代码。 |
| **frequency** | string | 否 | 检查频率，取值同创建接口的 **frequency 取值说明**（如 `3m`、`5m`、`15m`、`30m`、`60m`、`daily`）。 |
| **grid** | object | 否 | 网格配置，结构同创建接口的 **grid 结构说明**。 |
| **attitude** | object | 否 | 态度/分析配置，结构同创建接口的 **attitude 结构说明**。 |

**说明**：仅传需要修改的字段即可，未传字段保持原值。提醒总开关（`reminder_switch`）不在本接口中修改，请使用后续提供的专用开关接口。

### 响应说明

- **成功**：`code === 100`，表示更新 GRID Reminder 配置成功。
- **业务错误**：`code` 为 101、102、103 等时表示未找到 Reminder、用户不存在等，HTTP 状态码为 400。
- **mcp_token 无效**：未找到对应用户时返回 `code: 102`，HTTP 401。
- **调用 Bot 服务失败**：返回 502 及相应错误信息。

响应体示例（成功）：

```json
{
  "code": 100,
  "message": "更新GRID Reminder配置成功"
}
```

### 请求示例

```bash
curl -X POST "http://localhost:5700/updateGRIDReminder" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <API_KEY>" \
  -d '{
    "mcp_token": "mcp_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
    "reminder_id": "67175fa4d6f9d6cf92bc5dd9",
    "frequency": "10m"
  }'
```

### 与 Bot 服务的关系

- 用 `mcp_token` 解析 `user_id`，将 `reminder_id` 及可更新字段转发至 Bot 服务的 `POST /editBot`（`bot_type: "GRIDReminder"`，`bot_id` 即 `reminder_id`）。
- 本接口不处理提醒开关；开关由专用接口控制。
- 修改逻辑与调度更新在 `botAPIServer` 的 `editGRIDReminder(user_id, bot_id)` 中完成。

---

## 删除 GRIDReminder

### 接口定义

| 项目 | 说明 |
|------|------|
| **URL** | `/deleteGRIDReminder` |
| **方法** | `POST` |
| **Content-Type** | `application/json` |
| **认证** | 需要在 Header 中携带 `Authorization: Bearer <API_KEY>` |

### 请求体参数

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| **mcp_token** | string | 是 | 用户 MCP 令牌。 |
| **reminder_id** | string | 是 | 要删除的 GRID Reminder ID（即 Bot 侧 `bot_id`）。 |

### 响应说明

- **成功**：`code === 100`，表示删除成功。
- **业务错误**：`code` 为 102、103、104 等时表示未找到 Bot/用户或删除失败，HTTP 状态码为 400。
- **mcp_token 无效**：未找到对应用户时返回 `code: 102`，HTTP 401。
- **调用 Bot 服务失败**：返回 502。

响应体示例（成功）：

```json
{
  "code": 100,
  "message": "删除Grid Reminder成功"
}
```

### 请求示例

```bash
curl -X POST "http://localhost:5700/deleteGRIDReminder" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <API_KEY>" \
  -d '{
    "mcp_token": "mcp_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
    "reminder_id": "67175fa4d6f9d6cf92bc5dd9"
  }'
```

### 与 Bot 服务的关系

- 用 `mcp_token` 解析 `user_id`，将 `reminder_id` 转发至 Bot 服务的 `POST /deleteBot`（`bot_type: "GRIDReminder"`）。
- 删除逻辑、调度移除、提醒数量扣减在 `botAPIServer` 的 `deleteGRIDReminder(user_id, bot_id)` 中完成。

---

## 获取所有 GRIDReminder

### 接口定义

| 项目 | 说明 |
|------|------|
| **URL** | `/getAllGRIDReminders` |
| **方法** | `POST` |
| **Content-Type** | `application/json` |
| **认证** | 需要在 Header 中携带 `Authorization: Bearer <API_KEY>` |

### 请求体参数

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| **mcp_token** | string | 是 | 用户 MCP 令牌。 |
| **code** | string | 否 | 按标的代码筛选；不传则返回该用户下全部 GRID Reminder。 |

### 响应说明

- **成功**：`code === 100`，`message` 为 `"success"`，`data` 为当前用户（及可选 `code` 筛选下）的 GRID Reminder 列表。列表中每项包含 `reminder_id`、`bot_id`、`botname`、`stock_name`、`code`、`frequency`、`reminder_switch`、`grid`、`attitude`、`user_id`、`type`（固定为 `"GRIDReminder"`）以及状态字段 `buy_grid`、`sell_grid`、`current_grid`（与 Bot 侧 `getUserReminder` 返回的 GRID 结构一致）。
- **mcp_token 无效**：未找到对应用户时返回 `code: 102`，HTTP 401。

响应体示例（成功）：

```json
{
    "code": 100,
    "data": [
        {
            "attitude": {
                "analysis_prompt_list": [
                    "664f5e464883b402278dc92c"
                ],
                "analysis_period": "daily",
                "controll_switch": false,
                "switch": false
            },
            "bot_id": "671affe2d6f9d6cf92bc6120",
            "botname": "五粮液提醒器",
            "buy_grid": [
                100.0
            ],
            "code": "000858.SZ",
            "current_grid": 110.0,
            "frequency": "5m",
            "grid": {
                "grid_num": 9,
                "lower_limit_price": 100.0,
                "upper_limit_price": 180.0
            },
            "reminder_id": "671affe2d6f9d6cf92bc6120",
            "reminder_switch": "True",
            "sell_grid": [
                180.0,
                170.0,
                160.0,
                150.0,
                140.0,
                130.0,
                120.0
            ],
            "stock_name": "五粮液",
            "type": "GRIDReminder",
            "user_id": "6366170502d5c175fd586fe8"
        },
        {
            "attitude": {
                "analysis_prompt_list": [],
                "analysis_period": "daily",
                "controll_switch": false,
                "switch": false
            },
            "bot_id": "69b62dc1bc9920e4d09c735d",
            "botname": "泸州老窖中长线网格",
            "buy_grid": [
                90.0,
                97.0,
                104.0
            ],
            "code": "000568.SZ",
            "current_grid": 111.0,
            "frequency": "60m",
            "grid": {
                "grid_num": 6,
                "lower_limit_price": 90.0,
                "upper_limit_price": 125.0
            },
            "reminder_id": "69b62dc1bc9920e4d09c735d",
            "reminder_switch": "True",
            "sell_grid": [
                125.0,
                118.0
            ],
            "stock_name": "泸州老窖",
            "type": "GRIDReminder",
            "user_id": "6366170502d5c175fd586fe8"
        }
    ],
    "message": "success"
}
```

### 请求示例

```bash
# 获取该用户全部 GRID Reminder
curl -X POST "http://localhost:5700/getAllGRIDReminders" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <API_KEY>" \
  -d '{"mcp_token": "mcp_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"}'

# 仅获取指定标的的 GRID Reminder
curl -X POST "http://localhost:5700/getAllGRIDReminders" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <API_KEY>" \
  -d '{"mcp_token": "mcp_xxx", "code": "518880.SH"}'
```

### 与 Bot 服务的关系

- 用 `mcp_token` 解析 `user_id`，MCP 层直读 `grid_reminder`、`grid_reminder_info` 及 `attitude_log`，组装逻辑与 Bot 服务 `getUserReminder` 的 **GRID** 部分一致。
- 返回的 `reminder_id` 与 `bot_id` 均为 `grid_reminder._id`，可直接用于 `/updateGRIDReminder`、`/deleteGRIDReminder`。

---

## 获取 GRID 提醒运行日志（getGRIDReminderLog）

按 `reminder_id` 查询该 GRID 提醒最近最多 **100** 条调度记录（与 Bot 服务 `POST /getGRIDReminderLog` 数据源与字段一致）。MCP 层先用 `mcp_token` 解析 `user_id`，并校验该 `reminder_id` 属于当前用户后再读库。**MCP 不提供日志筛选类请求参数**，固定返回全量（最多 **100** 条）。

### 接口定义

| 项目 | 说明 |
|------|------|
| **URL** | `/getGRIDReminderLog` |
| **方法** | `POST` |
| **Content-Type** | `application/json` |
| **认证** | `Authorization: Bearer <API_KEY>` |

### 请求体参数

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| **mcp_token** | string | 是 | 用户 MCP 令牌。 |
| **reminder_id** | string | 是 | 提醒 ID，即 `grid_reminder._id`，与 `getAllGRIDReminders` 返回列表中每条记录的 **`reminder_id`** 字段一致。 |

### 响应说明

**成功（HTTP 200）**：`code === 100`，`message` 一般为 `"success"`，**`data` 为数组**：

- **条数与顺序**：最多 **100** 条；按日志时间 **从新到旧** 排列。
- **数据来源**：每条对应一次 GRID 提醒调度任务结束时写入 `grid_reminder_log` 的快照（`log` 字段）；MCP 与 Bot `getGRIDReminderLog` 返回的字段一致。库内完整 `log` 另含 **`current_price`**（当时行情价），**本接口不返回**该字段，仅返回下表所列五项。

**`data[]` 每条记录的字段**：

| 字段 | 类型 | 说明 |
|------|------|------|
| **time** | string | 本条日志生成时间，常见为 **`YYYY-MM-DD HH:MM:SS`**（与任务内 `strftime` 一致）。 |
| **buy_grid** | number[] | **买入侧**待触发的网格价位列表（从下方向上扫档时使用的档位集合快照）。为空数组 `[]` 表示当前没有更下方的待买档（例如价格已在网格下沿附近）。 |
| **sell_grid** | number[] | **卖出侧**待触发的网格价位列表（从上方向下扫档）。一般为从网格上沿向下排列的多个价位；条数随成交与档位迁移而变化。 |
| **current_grid** | number | **当前网格**价位（机器人当前所处的网格价格档；与 `grid_reminder_info.current_grid` 一致，为本条日志写入时的快照）。 |
| **next_opt** | string | 本周期根据现价与档位关系给出的操作提示：**`hold`**（未触发买卖档）、**`buy`**（现价不高于 `buy_grid` 最后一个档位且 `buy_grid` 非空）、**`sell`**（现价不低于 `sell_grid` 最后一个档位且 `sell_grid` 非空）；二者皆不满足时为 **`hold`**。 |

**阅读示例**：若长时间为 `next_opt: "hold"` 且 `buy_grid`、`sell_grid` 数值不变，表示行情未触及下一档买卖线；若出现 `buy`/`sell`，下一跳日志里通常会看到 `buy_grid` / `sell_grid` / `current_grid` 随档位迁移而更新（例如从下方买回后 `buy_grid` 可能出现新档、`current_grid` 上移）。

**错误响应**：

- **缺少参数**：`code === 401`（未传 `mcp_token` 或 `reminder_id`）。
- **无效令牌**：`code === 102`，HTTP 401。
- **ID 非法**：`code === 101`。
- **无权限或不存在**：`code === 103`。

**`data` 单条结构示例**：

```json
{
  "time": "2026-03-27 15:00:00",
  "buy_grid": [],
  "sell_grid": [180.0, 170.0, 160.0, 150.0, 140.0, 130.0, 120.0, 110.0],
  "current_grid": 100.0,
  "next_opt": "hold"
}
```

### 请求示例

```bash
curl -X POST "http://localhost:5700/getGRIDReminderLog" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <API_KEY>" \
  -d '{
    "mcp_token": "mcp_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
    "reminder_id": "671affe2d6f9d6cf92bc6120"
  }'
```

### 与 Bot 服务的关系

- Bot 侧：`POST /getGRIDReminderLog`（请求体字段以 Bot 服务为准），返回字段与上表一致（不含库内 `log.current_price`）。
- MCP 侧：请求体使用 **`reminder_id`**（含义与 Bot 的 `bot_id` 相同），与 `getAllGRIDReminders` 对齐；归属校验后直读 `grid_reminder_log`；成功响应为 `{ code, message, data }`。

---

## 参考

- **默认结构**：`Constants/Basic_reminder.py` 中的 `Grid_Reminder`（`botname`、`type`、`stock_name`、`code`、`frequency`、`grid`、`attitude`）及 `Grid_Reminder_Info`（`switch`、`buy_grid`、`sell_grid`、`current_grid`）。
- **Bot 服务**：
  - 创建：`botAPIServer` 中 `createGRIDReminder(user_id)` 及路由 `POST /createBot`（`bot_type: "GRIDReminder"`）。
  - 修改：`editGRIDReminder(user_id, bot_id)` 及 `POST /editBot`；开关：`POST /switchBot`（`bot_type: "GRIDReminder"`）。
  - 删除：`deleteGRIDReminder(user_id, bot_id)` 及 `POST /deleteBot`。
  - 获取：MCP 直读 `grid_reminder`、`grid_reminder_info`，与 `getUserReminder` 中 GRID 列表结构一致。
- **MCP 入口**：`mcpAPIServer` 中 `POST /createGRIDReminder`、`POST /updateGRIDReminder`、`POST /deleteGRIDReminder`、`POST /getAllGRIDReminders`、`POST /getGRIDReminderLog`，均通过 `mcp_token` 解析 `user_id`；创建/修改/删除转发至 Bot，列表与日志为 MCP 直读库。
