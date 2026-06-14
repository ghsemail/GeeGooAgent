# MCP API：行情与资金

## 概述

本文档描述 geegoo mcp 上与**行情、资金、信号列表**相关的接口，包括：

- 标的搜索、最新价、逐笔、经纪队列  
- 指标 / 组合信号列表（Admin 转发）  
- 交易日校验、资金流向与分布、机器人态度（Workflow 感知）

**账户持仓**、**Bot 运行日志**（`/getPosition`、`/getBotLogByType`）见 [common.md](../common.md)。**报告 Workflow**（待分析标的列表、三类报告 CRUD）见 [reports.md](./reports.md)。

**基础路径**：`http://<host>:5700`

---

## 认证与请求约定

| 项目 | 说明 |
|------|------|
| **HTTP Header** | `Authorization: Bearer <API_KEY>`，与 geegoo mcp 进程内配置的 API Key 一致。缺少、格式错误或 Key 不匹配时返回 HTTP **401**，响应体为 JSON，含 **`error`** 字段，**不使用**下文业务码 `code`。 |
| **Content-Type** | `application/json` |
| **方法** | `POST` |

---

## 标的搜索（searchCode）

与 **`Utility.searchCode`** 相同实现：按**代码或名称**模糊搜索，调用 Signal **`POST /searchCode`**。**不需要** `mcp_token`。

### 接口定义

| 项目 | 说明 |
|------|------|
| **URL** | `/searchCode` |
| **方法** | `POST` |
| **认证** | Header：`Authorization: Bearer <API_KEY>` |

### 请求体参数

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| **regex** | string | 是 | 代码或名称关键词（不可为空字符串）。 |
| **market** | string[] | 否 | 市场过滤，如 `["HK","US"]`；不传则不限市场。 |

### 成功响应

HTTP **200**，body 为 **JSON 数组**；每项可含 `code`、`name`、`lot_size`、`market`、`stock_type` 等（以 Signal 返回为准）。无结果时为 `[]`。

### 失败与业务码

| code | HTTP 状态 | 说明 |
|------|------------|------|
| **400** | 400 | 缺少或空白 `regex`。 |
| **500** | 500 | 服务端异常。 |

### 请求示例

```bash
curl -X POST "http://<host>:5700/searchCode" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <API_KEY>" \
  -d '{"regex":"腾讯","market":["HK","US"]}'
```

---

## 校验是否为交易日（checkTradingDay）

### 接口定义

| 项目 | 说明 |
|------|------|
| **URL** | `/checkTradingDay` |
| **方法** | `POST` |
| **认证** | Header：`Authorization: Bearer <API_KEY>`；Body：必填 **`mcp_token`** |

### 请求体参数

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| **mcp_token** | string | 是 | 用户 MCP 令牌（`mcp.mcp_token`）；无效或查无用户时返回业务码 **102**，HTTP **401**。 |
| **code** | string | 是 | 标的代码，如 `00700.HK`、`AAPL.US`、`600519.SH`；用于按标的所在市场参与交易日判断（格式与机器人侧一致）。 |

**默认规则**：
- `market` 不作为请求参数传入，由服务端根据 `code` 自动推断（`HK`/`US`/`SH`/`SZ` 等后缀）。
- `date` 不作为请求参数传入，由服务端固定取「今天」（`US` 按美东日期，其他按服务器本地日期）。

### 成功响应

HTTP **200**，JSON 示例：

```json
{
  "code": 100,
  "message": "success",
  "data": {
    "is_trading_day": true,
    "date": "2026-04-25",
    "market": "HK",
    "code": "00700.HK"
  }
}
```

| 字段 | 说明 |
|------|------|
| **code** | 业务码，成功为 **100**。 |
| **data.is_trading_day** | **boolean**：该 **`date`** 在对应 **`market`** 下是否为交易日。 |
| **data.date** | 实际参与判断的日期字符串 **`YYYY-MM-DD`**。 |
| **data.market** | **`HK`** / **`US`** / **`CN`**。 |
| **data.code** | 请求中的标的代码（去首尾空格后）。 |

### 失败与业务码

| code | HTTP 状态 | 说明 |
|------|------------|------|
| **401** | 400 | 缺少 **`mcp_token`**。 |
| **102** | 401 | **`mcp_token`** 无效或用户不存在。 |
| **101** | 400 | 用户文档不存在（极少见于已通过 token 解析到 `user_id` 后）。 |
| **103** | 400 | 用户未配置 **`trade.bot_host`** 或 **`trade.bot_port`**，无法建立行情连接。 |
| **400** | 400 | 缺少 **`code`** 或无法从 **`code`** 推断 **`market`**（**`message`** 含具体原因）。 |
| **500** | 500 | 服务端异常。 |

### 行为说明

- **连接参数**：使用 **`user.trade.bot_host`、`user.trade.bot_port`**，与依赖同一组交易/行情配置的其它 MCP 接口一致；须保证 MCP 进程到该地址的网络可达，且对端服务已就绪、账号与权限满足查询要求。  
- **错误与 `false`**：远端行情链路异常或查询失败时，当前实现可能将 **`is_trading_day`** 置为 **`false`**，且不单独返回「查询失败」类业务码；若结果与预期不符，请结合对端服务状态与项目侧运行日志排查。  
- **`code` 与 `market`**：判断按**市场**维度；`market` 由 `code` 自动推断，`code` 须为可识别的标准证券代码。

---

## 获取股票资金流向（getCapitalFlow）

获取个股资金流向数据。调用方传 `mcp_token`，服务端解析用户后，使用用户绑定的 `trade.bot_host`、`trade.bot_port` 连接行情服务进行查询。

### 接口定义

| 项目 | 说明 |
|------|------|
| **URL** | `/getCapitalFlow` |
| **方法** | `POST` |
| **Content-Type** | `application/json` |
| **认证** | Header：`Authorization: Bearer <API_KEY>` |

### 请求体参数

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| **mcp_token** | string | 是 | 用户 MCP 令牌。 |
| **code** | string | 是 | 标的代码，如 `00700.HK`、`AAPL.US`、`600519.SH`。 |
| **period** | string | 是 | 周期：`INTRADAY` / `DAY` / `WEEK` / `MONTH`。 |
| **start** | string | 否 | 开始日期，格式 `YYYY-MM-DD`。不传时按 Futu 默认规则处理。 |

> 说明：该接口**不需要 `end` 参数**。若传 `start`，结束日期由上游默认逻辑推导。当前实现不支持 `YEAR`，传入会导致服务端报错（`PeriodType.YEAR` 不存在）。

### 成功响应

HTTP **200**，JSON 示例：

```json
{
  "code": 100,
  "message": "success",
  "data": [
    {
      "in_flow": -185791500.0,
      "main_in_flow": -106682800.0,
      "super_in_flow": -32509240.0,
      "big_in_flow": -3675200.0,
      "mid_in_flow": -7317740.0,
      "sml_in_flow": -4846380.0,
      "capital_flow_item_time": "2026-04-25 09:31:00",
      "last_valid_time": "N/A"
    }
  ]
}
```

### 失败与业务码

| code | HTTP 状态 | 说明 |
|------|------------|------|
| **401** | 400 | 缺少 `mcp_token`。 |
| **102** | 401 | `mcp_token` 无效或用户不存在。 |
| **103** | 400 | 用户未配置 `trade.bot_host` 或 `trade.bot_port`。 |
| **400** | 400 | 缺少 `code` 或 `period`，或参数格式非法。 |
| **500** | 500 | 服务端异常。 |

### 请求示例

```bash
curl -X POST "http://<host>:5700/getCapitalFlow" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <API_KEY>" \
  -d '{"mcp_token":"your_mcp_token","code":"00700.HK","period":"DAY","start":"2026-01-01"}'
```

---

## 获取股票资金分布（getCapitalDistribution）

获取个股资金分布（超大单/大单/中单/小单的流入与流出）。调用方传 `mcp_token`，服务端解析用户后，使用用户绑定的 `trade.bot_host`、`trade.bot_port` 连接行情服务进行查询。

### 接口定义

| 项目 | 说明 |
|------|------|
| **URL** | `/getCapitalDistribution` |
| **方法** | `POST` |
| **Content-Type** | `application/json` |
| **认证** | Header：`Authorization: Bearer <API_KEY>` |

### 请求体参数

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| **mcp_token** | string | 是 | 用户 MCP 令牌。 |
| **code** | string | 是 | 标的代码，如 `00700.HK`、`AAPL.US`、`600519.SH`。 |

### 成功响应

HTTP **200**，JSON 示例：

```json
{
  "code": 100,
  "message": "success",
  "data": {
    "capital_in_super": 1000000.0,
    "capital_in_big": 520000.0,
    "capital_in_mid": 310000.0,
    "capital_in_small": 180000.0,
    "capital_out_super": 860000.0,
    "capital_out_big": 470000.0,
    "capital_out_mid": 260000.0,
    "capital_out_small": 150000.0,
    "update_time": "2026-04-27 09:31:00"
  }
}
```

### 失败与业务码

| code | HTTP 状态 | 说明 |
|------|------------|------|
| **401** | 400 | 缺少 `mcp_token`。 |
| **102** | 401 | `mcp_token` 无效或用户不存在。 |
| **101** | 400 | 用户不存在。 |
| **103** | 400 | 用户未配置 `trade.bot_host` 或 `trade.bot_port`。 |
| **400** | 400 | 缺少 `code` 或参数格式非法。 |
| **500** | 500 | 服务端异常。 |

### 请求示例

```bash
curl -X POST "http://<host>:5700/getCapitalDistribution" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <API_KEY>" \
  -d '{"mcp_token":"your_mcp_token","code":"00700.HK"}'
```

---

## 获取实时逐笔（getTicker）

获取已订阅股票的实时逐笔数据。调用方传 `mcp_token`，服务端解析用户后，使用用户绑定的 `trade.bot_host`、`trade.bot_port` 查询。

### 接口定义

| 项目 | 说明 |
|------|------|
| **URL** | `/getTicker` |
| **方法** | `POST` |
| **Content-Type** | `application/json` |
| **认证** | Header：`Authorization: Bearer <API_KEY>` |

### 请求体参数

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| **mcp_token** | string | 是 | 用户 MCP 令牌。 |
| **code** | string | 是 | 标的代码，如 `00700.HK`、`AAPL.US`。 |
| **num** | int | 否 | 最近逐笔条数，默认 `500`，范围 `1~1000`。 |

### 成功响应

HTTP **200**，JSON 示例：

```json
{
  "code": 100,
  "message": "success",
  "data": [
    {
      "code": "00700.HK",
      "name": "腾讯控股",
      "sequence": 7490506385373790208,
      "time": "2026-05-16 10:20:24.170",
      "price": 181.73,
      "volume": 1.0,
      "turnover": 181.73,
      "ticker_direction": "NEUTRAL",
      "type": "ODD_LOT"
    }
  ]
}
```

### 失败与业务码

| code | HTTP 状态 | 说明 |
|------|------------|------|
| **401** | 400 | 缺少 `mcp_token`。 |
| **102** | 401 | `mcp_token` 无效或用户不存在。 |
| **101** | 400 | 用户不存在。 |
| **103** | 400 | 用户未配置 `trade.bot_host` 或 `trade.bot_port`。 |
| **400** | 400 | 缺少 `code`，或 `num` 非整数/不在 `1~1000`。 |
| **500** | 500 | 服务端异常。 |

### 请求示例

```bash
curl -X POST "http://<host>:5700/getTicker" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <API_KEY>" \
  -d '{"mcp_token":"your_mcp_token","code":"00700.HK","num":200}'
```

---

## 获取实时经纪队列（getBroker）

获取已订阅股票的实时经纪队列（买盘/卖盘）。调用方传 `mcp_token`，服务端解析用户后，使用用户绑定的 `trade.bot_host`、`trade.bot_port` 查询。

### 接口定义

| 项目 | 说明 |
|------|------|
| **URL** | `/getBroker` |
| **方法** | `POST` |
| **Content-Type** | `application/json` |
| **认证** | Header：`Authorization: Bearer <API_KEY>` |

### 请求体参数

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| **mcp_token** | string | 是 | 用户 MCP 令牌。 |
| **code** | string | 是 | 标的代码，如 `00700.HK`。 |

### 成功响应

HTTP **200**，JSON 示例：

```json
{
  "code": 100,
  "message": "success",
  "data": {
    "bid_list": [
      {
        "code": "00700.HK",
        "name": "腾讯控股",
        "bid_broker_id": 5338,
        "bid_broker_name": "J.P.摩根",
        "bid_broker_pos": 1,
        "order_id": "N/A",
        "order_volume": "N/A"
      }
    ],
    "ask_list": [
      {
        "code": "00700.HK",
        "name": "腾讯控股",
        "ask_broker_id": 8305,
        "ask_broker_name": "富途证券国际(香港)有限公司",
        "ask_broker_pos": 4,
        "order_id": "N/A",
        "order_volume": "N/A"
      }
    ]
  }
}
```

### 失败与业务码

| code | HTTP 状态 | 说明 |
|------|------------|------|
| **401** | 400 | 缺少 `mcp_token`。 |
| **102** | 401 | `mcp_token` 无效或用户不存在。 |
| **101** | 400 | 用户不存在。 |
| **103** | 400 | 用户未配置 `trade.bot_host` 或 `trade.bot_port`。 |
| **400** | 400 | 缺少 `code` 或参数格式非法。 |
| **500** | 500 | 服务端异常。 |

### 请求示例

```bash
curl -X POST "http://<host>:5700/getBroker" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <API_KEY>" \
  -d '{"mcp_token":"your_mcp_token","code":"00700.HK"}'
```

---

## 获取机器人昨日态度日志（getBotYesterdayAttitude）

根据 `bot_id` 读取该机器人“昨天”的 attitude 记录，并按请求参数 `language` 返回对应语言的 `analysis_report`。

### 接口定义

| 项目 | 说明 |
|------|------|
| **URL** | `/getBotYesterdayAttitude` |
| **方法** | `POST` |
| **Content-Type** | `application/json` |
| **认证** | Header：`Authorization: Bearer <API_KEY>` |

### 请求体参数

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| **mcp_token** | string | 是 | 用户 MCP 令牌。 |
| **bot_id** | string | 是 | 机器人 `_id`（字符串）。 |
| **language** | string | 否 | `cn` / `en` / `hk`，不传默认 `cn`。 |

### 成功响应

HTTP **200**，JSON 示例：

```json
{
  "code": 100,
  "message": "success",
  "data": {
    "attitude_id": "6631a2b3c4d5e6f7890ab123",
    "bot_id": "662f3e12ab45cd7890ef1234",
    "code": "00700.HK",
    "stock_name": "腾讯控股",
    "date": "2026-04-27",
    "attitude": "neutral",
    "analysis_report": "昨日市场波动加大，建议控制仓位并关注支撑位。",
    "language": "cn"
  }
}
```

### 失败与业务码

| code | HTTP 状态 | 说明 |
|------|------------|------|
| **401** | 400 | 缺少 `mcp_token`。 |
| **102** | 401 | `mcp_token` 无效或用户不存在。 |
| **400** | 400 | 缺少 `bot_id`、`language` 非法，或参数格式非法。 |
| **101** | 400 | `bot_id` 非法（不是有效 ObjectId）。 |
| **103** | 404 | 未找到该机器人或无权限访问。 |
| **104** | 400 | 该机器人未配置 `code`。 |
| **105** | 404 | 未找到该机器人昨天的 attitude 记录。 |
| **500** | 500 | 服务端异常。 |

### 行为说明

- 仅允许访问当前 `mcp_token` 对应用户自己的机器人。
- 机器人来源支持：`dca_bot`、`grid_bot`、`trade_bot`。
- “昨天”按服务端本地时间计算，格式为 `YYYY-MM-DD`。
- 态度日志仅按 `bot_id + date` 精确查询。
- `attitude` 为三态枚举：`bearish` / `bullish` / `neutral`。
- 当 `analysis_report` 为多语言对象时，按 `language` 返回对应字符串；若缺少对应语言，回退 `cn`。

### 请求示例

```bash
curl -X POST "http://<host>:5700/getBotYesterdayAttitude" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <API_KEY>" \
  -d '{"mcp_token":"your_mcp_token","bot_id":"662f3e12ab45cd7890ef1234","language":"en"}'
```

---

## 获取单项分析模板（getSinglePromptTemplate）

用于获取单项分析 Prompt 模板列表（仅返回 `switch=true` 条目）。调用方传 `mcp_token`，服务端解析为 `user_id` 后转发到上游模板服务。

### 接口定义

| 项目 | 说明 |
|------|------|
| **URL** | `/getSinglePromptTemplate` |
| **方法** | `POST` |
| **Content-Type** | `application/json` |
| **认证** | Header：`Authorization: Bearer <API_KEY>` |

### 请求体参数

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| **mcp_token** | string | 是 | 用户 MCP 令牌。 |
| **type** | string | 是 | `index` / `tech` / `fundamental`。 |
| **period** | string | 否 | 可选：`no_period`、`minutes`、`hourly`、`daily`、`weekly`、`monthly`、`quarterly`、`yearly`、`longterm`。不传则不按周期收窄。 |

### 响应说明

- 成功时返回模板数组（具体字段以实际返回为准，常见含 `prompt_id`、`type`、`creator`、`period`、`template`）。
- 常见错误：
  - 缺少 `mcp_token`：HTTP 400
  - `mcp_token` 无效：HTTP 401，`code=102`
  - `type` 非法：HTTP 400
  - 上游服务不可用：HTTP 502

### 请求示例

```bash
curl -X POST "http://<host>:5700/getSinglePromptTemplate" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <API_KEY>" \
  -d '{"mcp_token":"your_mcp_token","type":"tech","period":"monthly"}'
```

---

## 获取 MCP 分析结果（getMCPAnalysis）

根据 `prompt_id`、标的代码、周期等参数执行分析并返回结果。调用方传 `mcp_token`，服务端解析为 `user_id` 后转发到分析服务。

### 接口定义

| 项目 | 说明 |
|------|------|
| **URL** | `/getMCPAnalysis` |
| **方法** | `POST` |
| **Content-Type** | `application/json` |
| **认证** | Header：`Authorization: Bearer <API_KEY>` |

### 请求体参数

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| **mcp_token** | string | 是 | 用户 MCP 令牌。 |
| **name** | string | 是 | 标的名称，如 `腾讯控股`。 |
| **code** | string | 是 | 标的代码，如 `00700.HK`。 |
| **prompt_id** | string | 是 | 模板 ID（通常为 Mongo `_id`）。 |
| **period** | string | 是 | `no_period`、`minutes`、`hourly`、`daily`、`weekly`、`monthly`、`quarterly`、`yearly`、`longterm`。 |
| **language** | string | 否 | `cn` / `en` / `hk`，不传默认 `cn`。 |

### 响应说明

- 成功：HTTP 200，常见为 `code=100`，`data` 中包含分析结果（如 `analysis_result`、`model`、`create_date`）。
- 失败：
  - `mcp_token` 无效：HTTP 401，`code=102`
  - `period` 缺失或非法：HTTP 400（可能返回 `allowed_period`）
  - `language` 非法：HTTP 400（可能返回 `allowed_language`）
  - 上游服务不可用：HTTP 502

### 请求示例

```bash
curl -X POST "http://<host>:5700/getMCPAnalysis" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <API_KEY>" \
  -d '{"mcp_token":"your_mcp_token","name":"腾讯控股","code":"00700.HK","prompt_id":"66494754fbe37cd6846ebd89","period":"monthly"}'
```

