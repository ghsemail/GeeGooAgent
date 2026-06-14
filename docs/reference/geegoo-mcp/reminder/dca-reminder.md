# MCP API：DCAReminder 接口说明

## 概述

本文档描述通过 MCP（Skills）对 **DCA 信号提醒机器人**（`bot_type: DCAReminder`）的**创建、修改、删除、获取列表与运行日志**接口。分类与命名见 [`common.md`](common.md)「机器人分类与命名」。调用方不传 `user_id`，改为传入 `mcp_token`，由服务端根据 `mcp_token` 解析出对应用户后再调用 Bot 服务对应逻辑。

- **基础路径**：geegoo mcp 根地址（默认示例：`http://0.0.0.0:5700`）
- **认证方式**：请求头 `Authorization: Bearer <API_KEY>`

**公共约定**：**`mcp_token`**、**`frequency`**、信号与 **`signal`** 见 [`common.md`](common.md)；技术分析 **`prompt_id` / `period`**、**`attitude`** 见 [`agent-analyst.md`](../analyst/agent-analyst.md)。

---

## 创建 DCAReminder

### 接口定义

| 项目 | 说明 |
|------|------|
| **URL** | `/createDCAReminder` |
| **方法** | `POST` |
| **Content-Type** | `application/json` |
| **认证** | 需要在 Header 中携带 `Authorization: Bearer <API_KEY>` |

### 请求体参数

请求体为 JSON，字段说明如下（与 `Constants/Basic_reminder.py` 中 **DCA_Reminder** 结构一致）：

| 参数名 | 类型 | 必填 | Basic_reminder 对应 | 说明 |
|--------|------|------|---------------------|------|
| **mcp_token** | string | 是 | —（仅 MCP 层） | 用户 MCP 令牌，用于在服务端解析 `user_id`，不传则返回 400。 |
| **botname** | string | 是 | `DCA_Reminder.botname` | 机器人名称，用于展示与唯一性校验（同用户下不可重复）。 |
| **stock_name** | string | 否 | `DCA_Reminder.stock_name` | 标的名称，如「黄金ETF」。 |
| **code** | string | 否 | `DCA_Reminder.code` | 标的代码，如 `518880.SH`。 |
| **frequency** | string | 否 | `DCA_Reminder.frequency` | 检查频率，如 `60m`、`5m` 等；共用枚举见 [`common.md`](common.md)。 |
| **signal** | object | 否 | `DCA_Reminder.signal` | 买卖信号配置，见下方 **signal 结构说明**。 |
| **attitude** | object | 否 | `DCA_Reminder.attitude` | 态度/分析配置，见下方 **attitude 结构说明**。 |
| **reminder_switch** | string | 否 | `DCA_Reminder.switch` | 提醒总开关，如 `"True"` / `"False"`；写入库时为布尔 `switch`。 |

**说明**：创建接口仅接受上表参数。`type`（固定为 `DCAReminder`）、`reminder_id`（即 `_id`）、`user_id` 由服务端生成或填充，请勿在请求体中传入。

#### signal 结构说明

与 DCA 策略的买卖信号一致：

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

- **buy_signal**：买入信号列表。每项包含：
  - **index**：指标名，如 `SAR`、`MACD`、`EMA` 等。
  - **type**：指标在组合中的角色；**`signal`** 表示**信号**（参与买卖触发），**`flag`** 表示**趋势**（辅助过滤/确认）；无卖出信号占位时可为空字符串。
  - **param**：该指标参数字典（键值均为字符串），如 SAR 的 `acceleration`、`maximum`，MACD 的 `fastPeriod`、`signalPeriod`、`slowPeriod` 等。
- **sell_signal**：卖出信号列表，结构同上；无信号时可使用 `index: "nosignal"`、`param: {}`。

#### attitude 结构说明

与 `Basic_reminder.py` 中 DCA_Reminder 的 attitude 一致：

```json
{
  "analysis_prompt_list": ["66436c877c670036b234fd40"],
  "analysis_period": "daily",
  "controll_switch": false,
  "switch": false
}
```

- **analysis_prompt_list**：分析用 prompt 的 ID 列表（字符串 ID）。
- **analysis_period**：字符串，单项分析 / 态度刷新采用的周期；未传时默认 **`daily`**（与 **`Constants/Basic_reminder.py`** 模板一致）。含义见 [`agent-analyst.md`](../analyst/agent-analyst.md) **约定与枚举 · attitude.analysis_period**。
- **controll_switch**：是否开启态度控制开关。
- **switch**：态度功能总开关。

#### 完整 DCAReminder 示例（创建后 Bot 侧结构参考）

以下为创建成功后，Bot 服务中一条 DCAReminder 的完整形态（含服务端生成的字段），供 Skills 开发时理解各字段含义。**请求体只需传上面表格中的参数，无需传 `reminder_id`、`type`、`user_id`。**

```json
{
  "attitude": {
    "analysis_prompt_list": ["66436c877c670036b234fd40"],
    "analysis_period": "daily",
    "controll_switch": false,
    "switch": false
  },
  "botname": "黄金ETF提醒器",
  "code": "518880.SH",
  "frequency": "60m",
  "reminder_id": "67175fa4d6f9d6cf92bc5dd8",
  "reminder_switch": "True",
  "signal": {
    "buy_signal": [
      { "index": "SAR", "param": { "acceleration": "0.02", "maximum": "0.2" }, "type": "signal" },
      { "index": "MACD", "param": { "fastPeriod": "12", "signalPeriod": "9", "slowPeriod": "26" }, "type": "flag" },
      { "index": "EMA", "param": { "fastPeriod": "25", "mediumPeriod": "50", "slowPeriod": "120" }, "type": "flag" }
    ],
    "sell_signal": [
      { "index": "nosignal", "param": {}, "type": "" }
    ]
  },
  "stock_name": "黄金ETF",
  "type": "DCAReminder",
  "user_id": "6366170502d5c175fd586fe8"
}
```

- **reminder_id**：由服务端生成的提醒 ID（对应库中 `_id`），仅响应/查询时返回。
- **type**：固定为 `DCAReminder`，由服务端写入。
- **user_id**：由服务端根据 `mcp_token` 解析后写入。

---

### 响应说明

- **成功**：`code === 100`，表示创建 DCA Reminder 成功。
- **业务错误**：`code` 为 101、102、103、105 等时表示业务校验失败（如 reminder 已存在、用户不存在、未绑定交易账号、提醒机器人数量不足等），HTTP 状态码为 400。
- **mcp_token 无效**：未找到对应用户时返回 `code: 102`，HTTP 401。
- **调用 Bot 服务失败**：返回 502 及相应错误信息。

响应体示例（成功）：

```json
{
  "code": 100,
  "message": "创建DCA Reminder成功"
}
```

响应体示例（reminder 已存在）：

```json
{
  "code": 101,
  "message": "reminder 已存在"
}
```

---

### 请求示例

```bash
curl -X POST "http://localhost:5700/createDCAReminder" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <API_KEY>" \
  -d '{"mcp_token":"mcp_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx","botname":"黄金ETF提醒器","stock_name":"黄金ETF","code":"518880.SH","frequency":"60m","reminder_switch":"True","signal":{"buy_signal":[{"index":"SAR","type":"signal","param":{"acceleration":"0.02","maximum":"0.2"}},{"index":"MACD","type":"flag","param":{"fastPeriod":"12","signalPeriod":"9","slowPeriod":"26"}}],"sell_signal":[{"index":"nosignal","type":"","param":{}}]},"attitude":{"analysis_prompt_list":[],"analysis_period":"daily","switch":false,"controll_switch":false}}'
```

---

### 与 Bot 服务的关系

- 本接口只做两件事：用 `mcp_token` 解析 `user_id`，并将请求参数转发至 Bot 服务的 `POST /createBot`（`bot_type: "DCAReminder"`）。
- 创建逻辑、数量与权限校验、调度与通知等均在 Bot 服务（botAPIServer）中完成。
- 默认 Bot 服务地址由 `Config/APIConnection.py` 中 `--bot_server_ip`、`--bot_server_port` 决定（默认 `http://127.0.0.1:5600`）。

---

## 修改 DCAReminder

### 接口定义

| 项目 | 说明 |
|------|------|
| **URL** | `/updateDCAReminder` |
| **方法** | `POST` |
| **Content-Type** | `application/json` |
| **认证** | 需要在 Header 中携带 `Authorization: Bearer <API_KEY>` |

### 请求体参数

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| **mcp_token** | string | 是 | 用户 MCP 令牌，用于在服务端解析 `user_id`。 |
| **reminder_id** | string | 是 | 要修改的 DCA Reminder ID（即 Bot 侧 `bot_id` / 库中 `_id`），可从「获取所有 DCAReminder」或创建响应中取得。 |
| **botname** | string | 否 | 机器人名称。 |
| **stock_name** | string | 否 | 标的名称。 |
| **code** | string | 否 | 标的代码。 |
| **frequency** | string | 否 | 检查频率。 |
| **signal** | object | 否 | 买卖信号配置，结构同创建接口的 **signal 结构说明**。 |
| **attitude** | object | 否 | 态度/分析配置，结构同创建接口的 **attitude 结构说明**。 |
| **reminder_switch** | string/boolean | 否 | 提醒总开关，如 `"True"` / `"False"`；写入库时为 `switch`。 |

**说明**：仅传需要修改的字段即可，未传字段保持原值。

### 响应说明

- **成功**：`code === 100`，表示更新 DCA Reminder 配置成功。
- **业务错误**：`code` 为 101、102、103 等时表示未找到 Reminder、用户不存在等，HTTP 状态码为 400。
- **mcp_token 无效**：未找到对应用户时返回 `code: 102`，HTTP 401。
- **调用 Bot 服务失败**：返回 502 及相应错误信息。

响应体示例（成功）：

```json
{
  "code": 100,
  "message": "更新DCA Reminder配置成功"
}
```

### 请求示例

```bash
curl -X POST "http://localhost:5700/updateDCAReminder" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <API_KEY>" \
  -d '{
    "mcp_token": "mcp_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
    "reminder_id": "67175fa4d6f9d6cf92bc5dd8",
    "frequency": "30m",
    "reminder_switch": "True"
  }'
```

### 与 Bot 服务的关系

- 用 `mcp_token` 解析 `user_id`，将 `reminder_id` 及可更新字段转发至 Bot 服务的 `POST /editBot`（`bot_type: "DCAReminder"`，`bot_id` 即 `reminder_id`）。
- 修改逻辑与调度更新在 `botAPIServer` 的 `editDCAReminder(user_id, bot_id)` 中完成。

---

## 删除 DCAReminder

### 接口定义

| 项目 | 说明 |
|------|------|
| **URL** | `/deleteDCAReminder` |
| **方法** | `POST` |
| **Content-Type** | `application/json` |
| **认证** | 需要在 Header 中携带 `Authorization: Bearer <API_KEY>` |

### 请求体参数

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| **mcp_token** | string | 是 | 用户 MCP 令牌。 |
| **reminder_id** | string | 是 | 要删除的 DCA Reminder ID（即 Bot 侧 `bot_id`）。 |

### 响应说明

- **成功**：`code === 100`，表示删除成功。
- **业务错误**：`code` 为 102、103、104 等时表示未找到 Bot/用户或删除失败，HTTP 状态码为 400。
- **mcp_token 无效**：未找到对应用户时返回 `code: 102`，HTTP 401。
- **调用 Bot 服务失败**：返回 502。

响应体示例（成功）：

```json
{
  "code": 100,
  "message": "删除bot成功"
}
```

### 请求示例

```bash
curl -X POST "http://localhost:5700/deleteDCAReminder" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <API_KEY>" \
  -d '{
    "mcp_token": "mcp_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
    "reminder_id": "67175fa4d6f9d6cf92bc5dd8"
  }'
```

### 与 Bot 服务的关系

- 用 `mcp_token` 解析 `user_id`，将 `reminder_id` 转发至 Bot 服务的 `POST /deleteBot`（`bot_type: "DCAReminder"`）。
- 删除逻辑、调度移除、提醒数量扣减在 `botAPIServer` 的 `deleteDCAReminder(user_id, bot_id)` 中完成。

---

## 获取所有 DCAReminder

### 接口定义

| 项目 | 说明 |
|------|------|
| **URL** | `/getAllDCAReminders` |
| **方法** | `POST` |
| **Content-Type** | `application/json` |
| **认证** | 需要在 Header 中携带 `Authorization: Bearer <API_KEY>` |

### 请求体参数

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| **mcp_token** | string | 是 | 用户 MCP 令牌。 |
| **code** | string | 否 | 按标的代码筛选；不传则返回该用户下全部 DCA Reminder。 |

### 响应说明

- **成功**：`code === 100`，`data` 为当前用户（及可选 `code` 筛选下）的 DCA Reminder 列表。列表中每项包含 `reminder_id`、`botname`、`stock_name`、`code`、`frequency`、`reminder_switch`、`signal`、`attitude`、`user_id` 等（与 Bot 侧 `getUserReminder` 返回的 DCA 结构一致）。
- **mcp_token 无效**：未找到对应用户时返回 `code: 102`，HTTP 401。
- **调用 Bot 服务失败**：返回 502。

响应体示例（成功）：

```json
{
  "code": 100,
  "message": "success",
  "data": [
    {
      "user_id": "6366170502d5c175fd586fe8",
      "reminder_id": "67175fa4d6f9d6cf92bc5dd8",
      "botname": "黄金ETF提醒器",
      "stock_name": "黄金ETF",
      "code": "518880.SH",
      "frequency": "60m",
      "reminder_switch": "True",
      "signal": { "buy_signal": [], "sell_signal": [] },
      "attitude": { "analysis_prompt_list": [], "analysis_period": "daily", "switch": false, "controll_switch": false }
    }
  ]
}
```

### 请求示例

```bash
# 获取该用户全部 DCA Reminder
curl -X POST "http://localhost:5700/getAllDCAReminders" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <API_KEY>" \
  -d '{"mcp_token": "mcp_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"}'

# 仅获取指定标的的 DCA Reminder
curl -X POST "http://localhost:5700/getAllDCAReminders" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <API_KEY>" \
  -d '{"mcp_token": "mcp_xxx", "code": "518880.SH"}'
```

### 与 Bot 服务的关系

- 用 `mcp_token` 解析 `user_id`，转发至 Bot 服务的 `POST /getUserReminder`（传 `user_id` 及可选 `code`）。
- MCP 层仅返回 Bot 响应中的 **DCA** 列表，不返回 GRID、SmartReminder 等其它类型。

---

## 获取 DCA 提醒运行日志（getDCAReminderLog）

按 `reminder_id` 查询该 DCA 提醒最近最多 **100** 条调度记录（与 Bot 服务 `POST /getDCAReminderLog` 数据源与字段一致）。MCP 层先用 `mcp_token` 解析 `user_id`，并校验该 `reminder_id` 属于当前用户后再读库，**不可**跨用户访问他人提醒日志。**MCP 不提供日志筛选类请求参数**，固定返回全量（最多 **100** 条）。

### 接口定义

| 项目 | 说明 |
|------|------|
| **URL** | `/getDCAReminderLog` |
| **方法** | `POST` |
| **Content-Type** | `application/json` |
| **认证** | `Authorization: Bearer <API_KEY>` |

### 请求体参数

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| **mcp_token** | string | 是 | 用户 MCP 令牌。 |
| **reminder_id** | string | 是 | 提醒 ID，即 `dca_reminder._id`，与 `getAllDCAReminders` 返回列表中每条记录的 **`reminder_id`** 字段一致。 |

### 响应说明

**成功（HTTP 200）**：`code === 100`，`message` 一般为 `"success"`，**`data` 为数组**：

- **条数与顺序**：最多 **100** 条；按日志时间 **从新到旧** 排列（与库中 `dca_reminder_log` 查询一致）。
- **每条记录**（对应一次调度写入的快照）包含以下顶层字段：

| 字段 | 类型 | 说明 |
|------|------|------|
| **time** | string | 该条日志对应的时间点，常见为 **`YYYY-MM-DD HH:MM:SS`** 格式字符串（由服务端将 `time` 序列化得到；若运行环境不同，也可能为带 `T` 的 ISO 8601）。 |
| **next_opt** | string | 本周期**综合**后的操作建议，常见取值为 **`hold`**（持有）、**`buy`**（买入）、**`sell`**（卖出）；若策略或引擎扩展，也可能出现其它字符串，以实际返回为准。 |
| **buy_signal** | object | **买入侧**信号评估结果，与 `DCA_Reminder.signal.buy_signal` 配置的指标链对应，结构见下表。 |
| **sell_signal** | object | **卖出侧**信号评估结果，与 `sell_signal` 配置对应，结构同 `buy_signal`。 |

**`buy_signal` / `sell_signal` 对象结构**（二者字段相同，仅语义上一侧为买、一侧为卖）：

| 字段 | 类型 | 说明 |
|------|------|------|
| **name** | array | 参与组合计算的**指标/条件列表**（顺序与配置一致）。每项为对象，常见子字段：`index`（指标名，如 `SAR`、`MACD`、`EMA`；无有效卖出信号时可能出现 `nosignal`）、`param`（该指标参数字典，键值多为字符串）、`type`（**`signal`** = **信号**；**`flag`** = **趋势**；占位项可能为空字符串）。 |
| **value** | object | 该侧信号在**本周期**的汇总子结果，来自 Signal 服务 **`/getBotSignal`** 返回体中的 `buy_signal` / `sell_signal`（与 `DCA/DCAReminderTask.py` 写入日志的 `value` 一致）。通常含：`next_opt`（该侧局部文案）、**`signal`**（整数，含义见下表）。 |

**`value.signal` 整数约定**（与 Signal 服务及任务逻辑一致）：

| 取值 | 含义 |
|------|------|
| **0** | 持有 / 中性，该侧未触发方向指令 |
| **1** | 买入方向触发（出现在买入侧 `buy_signal.value.signal`） |
| **-1** | 卖出方向触发（出现在卖出侧 `sell_signal.value.signal`） |

**顶层 `next_opt` 与两侧 `signal` 的关系**（见 `DCA/DCAReminderTask.py`）：先取整型 `buy_signal = signal['buy_signal']['signal']`、`sell_signal = signal['sell_signal']['signal']`，再合成：

- `buy_signal == 1` 且 `sell_signal != -1` → 顶层 **`buy`**
- `sell_signal == -1` 且 `buy_signal != 1` → 顶层 **`sell`**
- 其余情况（含两侧均为 `0`，或同时为 **`1`** 与 **`-1`** 等）→ 顶层 **`hold`**

**说明**：`log` 中的 `buy_signal` / `sell_signal` 为**嵌套对象**；解读时可结合 `name` 中各 `index`/`param` 与 **`value.signal`**。列表中相邻两条的 `time` 间隔取决于提醒的 **`frequency`**（如 60m 则约 60 分钟一条，交易日历与调度可能跳过非交易时段）。

**错误响应**：

- **缺少参数**：`code === 401`（未传 `mcp_token` 或 `reminder_id`）。
- **无效令牌**：`code === 102`，HTTP 401。
- **ID 非法**：`code === 101`（非合法 ObjectId 字符串）。
- **无权限或不存在**：`code === 103`（该 ID 不属于当前用户或记录不存在）。

**`data` 单条结构示例**（字段与真实环境一致，仅作结构参考）：

```json
{
  "time": "2026-03-27 15:00:00",
  "next_opt": "hold",
  "buy_signal": {
    "name": [
      { "index": "SAR", "param": { "acceleration": "0.02", "maximum": "0.2" }, "type": "signal" },
      { "index": "MACD", "param": { "fastPeriod": "12", "signalPeriod": "9", "slowPeriod": "26" }, "type": "flag" },
      { "index": "EMA", "param": { "fastPeriod": "25", "mediumPeriod": "50", "slowPeriod": "120" }, "type": "flag" }
    ],
    "value": { "next_opt": "hold", "signal": 0 }
  },
  "sell_signal": {
    "name": [ { "index": "nosignal", "param": {}, "type": "" } ],
    "value": { "next_opt": "hold", "signal": 0 }
  }
}
```

### 请求示例

```bash
curl -X POST "http://localhost:5700/getDCAReminderLog" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <API_KEY>" \
  -d '{
    "mcp_token": "mcp_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
    "reminder_id": "67175fa4d6f9d6cf92bc5dd8"
  }'
```

### 与 Bot 服务的关系

- Bot 侧：`POST /getDCAReminderLog`（请求体字段以 Bot 服务为准），不经 `mcp_token` 校验归属。
- MCP 侧：请求体使用 **`reminder_id`**（含义与 Bot 的 `bot_id` 相同，即库中 `_id`），与 `getAllDCAReminders` 对齐；归属校验后在 `mcpAPIServer` 直读 `dca_reminder_log`，字段与 Bot 一致；成功响应统一为 `{ code, message, data }`。

---

## 参考

- **默认结构**：`Constants/Basic_reminder.py` 中的 `DCA_Reminder`（`botname`、`type`、`stock_name`、`code`、`frequency`、`switch`、`signal`、`attitude`）。
- **Bot 服务**：
  - 创建：`botAPIServer` 中 `createDCAReminder(user_id)` 及路由 `POST /createBot`（`bot_type: "DCAReminder"`）。
  - 修改：`editDCAReminder(user_id, bot_id)` 及 `POST /editBot`。
  - 删除：`deleteDCAReminder(user_id, bot_id)` 及 `POST /deleteBot`。
  - 获取：`POST /getUserReminder`，MCP 仅使用返回中的 `DCA` 列表。
- **MCP 入口**：`mcpAPIServer` 中 `POST /createDCAReminder`、`POST /updateDCAReminder`、`POST /deleteDCAReminder`、`POST /getAllDCAReminders`、`POST /getDCAReminderLog`，均通过 `mcp_token` 解析 `user_id`；创建/修改/删除转发至 Bot，日志为 MCP 直读库。
