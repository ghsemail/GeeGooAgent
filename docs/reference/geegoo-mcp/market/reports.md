# MCP API：报告与 Workflow

## 概述

本文档描述 geegoo mcp 中与**盘前 / 盘中 / 盘后报告 Workflow**相关的接口，包括：

- **待分析标的列表**（`getReportBotCodes`，原 `getUserBotCodes`）  
- 三类报告 CRUD + 按日聚合查询  

服务端使用三个 MongoDB 集合：

- `pre_market_report`（盘前分析报告）
- `intraday_trade_decision_report`（盘中交易决策报告）
- `post_market_report`（盘后分析报告）

**机器人关联字段（三类报告统一）**  

文档中均会写入 **`bot_id`、`bot_name`、`bot_type`** 三个字符串字段；调用创建接口时若未传，则存为空字符串 `""`。  

| 字段 | 说明 |
|------|------|
| **bot_id** | 机器人文档 `_id`（字符串）。**盘中**创建时必填；**盘前、盘后**创建时可选（可为空字符串）。 |
| **bot_name** | 机器人名称（与对应 bot 集合中的 `botname` 含义一致，便于列表展示与筛选）。 |
| **bot_type** | 机器人类型（如 `DCA`、`GRID`、`DCAReminder`、`GRIDReminder` 等）。 |

**基础路径**：`http://<host>:5700`

---

## 报告 Workflow · 待分析标的列表（getReportBotCodes）

> **命名说明**：该接口返回 `attitude.switch=true` 的机器人所覆盖的**待写报告标的**，不是通用的「列出全部 Bot」接口。  
> **推荐路径**：`POST /getReportBotCodes`  
> **兼容路径**：`POST /getUserBotCodes`（Deprecated，仅旧客户端兼容）

返回当前用户在 DCA / GRID / SmartTrade 及 Reminder 集合中，满足 `attitude.switch=true` 的去重标的列表（含 `bot_id` / `bot_name` / `bot_type`，供报告写入时使用）。

### 接口定义

| 项目 | 说明 |
|------|------|
| **URL** | `/getReportBotCodes`（推荐）或 `/getUserBotCodes`（兼容） |
| **方法** | `POST` |
| **认证** | Header：`Authorization: Bearer <API_KEY>`；Body：必填 `mcp_token` |

### 请求体参数

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| **mcp_token** | string | 是 | 用户 MCP 令牌。 |

### 成功响应

HTTP **200**，`data` 为对象数组，每项含 `code`、`stock_name`、`bot_id`、`bot_name`、`bot_type`。

### 请求示例

```bash
curl -X POST "http://<host>:5700/getReportBotCodes" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <API_KEY>" \
  -d '{"mcp_token":"your_mcp_token"}'
```

### 行为说明

- 过滤条件：`attitude.switch === true`  
- 去重维度：`code`  
- 盘前 / 盘后 Workflow 逐步骤遍历 `data`，**禁止硬编码股票代码**

---

## 认证与请求约定

| 项目 | 说明 |
|------|------|
| **HTTP Header** | `Authorization: Bearer <API_KEY>` |
| **Content-Type** | `application/json` |
| **方法** | `POST` |

下列报告 CRUD 接口均要求 Body 含 `mcp_token`，仅访问当前用户数据。

---

## 数据结构（pre_market_report）

单条报告字段如下：

```json
{
  "code": "00700.HK",
  "stock_name": "腾讯控股",
  "bot_id": "6781cb8309a2189f26d8866e",
  "bot_name": "我的DCA",
  "bot_type": "DCA",
  "result": "long",
  "confidence": "high",
  "reason": "判定依据",
  "suggestion": "buy",
  "report": "盘前分析报告原文",
  "summary": "针对盘前分析报告的总结",
  "support": 485.0,
  "resistance": 502.0
}
```

枚举约束：
- `result`: `long` / `short` / `neutral`
- `confidence`: `high` / `medium` / `low`
- `suggestion`: `buy` / `sell` / `hold`

---

## 创建盘前分析报告（createPreMarketReport）

### 接口定义

| 项目 | 说明 |
|------|------|
| **URL** | `/createPreMarketReport` |
| **方法** | `POST` |
| **认证** | Header：`Authorization: Bearer <API_KEY>`；Body：必填 `mcp_token` |

### 请求体参数

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| **mcp_token** | string | 是 | 用户 MCP 令牌。 |
| **code** | string | 是 | 标的代码，如 `00700.HK`。 |
| **stock_name** | string | 是 | 标的名称。 |
| **result** | string | 是 | `long` / `short` / `neutral`。 |
| **confidence** | string | 是 | `high` / `medium` / `low`。 |
| **reason** | string | 是 | 判定依据。 |
| **suggestion** | string | 是 | `buy` / `sell` / `hold`。 |
| **report** | string | 是 | 盘前分析报告原文。 |
| **summary** | string | 否 | 报告总结。 |
| **support** | number/string | 否 | 支撑位。 |
| **resistance** | number/string | 否 | 阻力位。 |
| **bot_id** | string | 否 | 关联机器人 ID；未传则写入 `""`。 |
| **bot_name** | string | 否 | 机器人名称；未传则写入 `""`。 |
| **bot_type** | string | 否 | 机器人类型；未传则写入 `""`。 |

### 成功响应

HTTP `200`：

```json
{
  "code": 100,
  "message": "success",
  "data": {
    "report_id": "680bc8e7f54cf8a14f82a8a2"
  }
}
```

---

## 更新盘前分析报告（updatePreMarketReport）

### 接口定义

| 项目 | 说明 |
|------|------|
| **URL** | `/updatePreMarketReport` |
| **方法** | `POST` |
| **认证** | Header：`Authorization: Bearer <API_KEY>`；Body：必填 `mcp_token` |

### 请求体参数

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| **mcp_token** | string | 是 | 用户 MCP 令牌。 |
| **report_id** | string | 是 | 报告 ID。 |
| **code** | string | 否 | 标的代码。 |
| **stock_name** | string | 否 | 标的名称。 |
| **result** | string | 否 | `long` / `short` / `neutral`。 |
| **confidence** | string | 否 | `high` / `medium` / `low`。 |
| **reason** | string | 否 | 判定依据。 |
| **suggestion** | string | 否 | `buy` / `sell` / `hold`。 |
| **report** | string | 否 | 盘前分析报告原文。 |
| **summary** | string | 否 | 报告总结。 |
| **support** | number/string | 否 | 支撑位。 |
| **resistance** | number/string | 否 | 阻力位。 |
| **bot_id** | string | 否 | 关联机器人 ID。 |
| **bot_name** | string | 否 | 机器人名称。 |
| **bot_type** | string | 否 | 机器人类型。 |

### 成功响应

HTTP `200`：

```json
{
  "code": 100,
  "message": "success"
}
```

---

## 删除盘前分析报告（deletePreMarketReport）

### 接口定义

| 项目 | 说明 |
|------|------|
| **URL** | `/deletePreMarketReport` |
| **方法** | `POST` |
| **认证** | Header：`Authorization: Bearer <API_KEY>`；Body：必填 `mcp_token` |

### 请求体参数

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| **mcp_token** | string | 是 | 用户 MCP 令牌。 |
| **report_id** | string | 是 | 报告 ID。 |

### 成功响应

HTTP `200`：

```json
{
  "code": 100,
  "message": "success"
}
```

---

## 查询盘前分析报告（getPreMarketReports）

### 接口定义

| 项目 | 说明 |
|------|------|
| **URL** | `/getPreMarketReports` |
| **方法** | `POST` |
| **认证** | Header：`Authorization: Bearer <API_KEY>`；Body：必填 `mcp_token` |

### 请求体参数

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| **mcp_token** | string | 是 | 用户 MCP 令牌。 |
| **report_id** | string | 否 | 按单条报告 ID 查询。 |
| **code** | string | 否 | 按标的代码筛选。 |

### 成功响应

HTTP `200`，返回列表：

```json
{
  "code": 100,
  "message": "success",
  "data": [
    {
      "report_id": "680bc8e7f54cf8a14f82a8a2",
      "bot_id": "6781cb8309a2189f26d8866e",
      "bot_name": "我的DCA",
      "bot_type": "DCA",
      "code": "00700.HK",
      "stock_name": "腾讯控股",
      "result": "long",
      "confidence": "high",
      "reason": "判定依据",
      "suggestion": "buy",
      "report": "盘前分析报告原文",
      "summary": "针对盘前分析报告的总结",
      "support": 485.0,
      "resistance": 502.0,
      "created_at": "2026-04-25T23:59:59.000000",
      "updated_at": "2026-04-26T00:05:12.000000"
    }
  ]
}
```

---

## 失败与业务码

| code | HTTP 状态 | 说明 |
|------|------------|------|
| **401** | 400 | 缺少 `mcp_token` 或必填 ID 参数。 |
| **102** | 401 | `mcp_token` 无效或用户不存在。 |
| **101** | 400 | `report_id` 非法。 |
| **103** | 400 | 未找到报告或无权限访问。 |
| **400** | 400 | 参数非法（如枚举值不在允许范围、缺少必填字段）。 |
| **500** | 500 | 服务端异常。 |

---

## 数据结构（intraday_trade_decision_report）

单条盘中交易决策报告字段如下：

```json
{
  "code": "00700.HK",
  "stock_name": "腾讯控股",
  "bot_id": "6781cb8309a2189f26d8866e",
  "bot_name": "我的DCA",
  "bot_type": "DCA",
  "result": "buy",
  "confidence": "high",
  "reason": "判定依据",
  "report": "针对上述分析信息的总结报告",
  "trade_type": "信号买入",
  "price": 500.0,
  "qty": 100.0,
  "position": {
    "opt": "hold",
    "order_id": "",
    "order_status": "",
    "order_time": "2026-04-26 15:30:00",
    "price": 500.0,
    "qty": 100.0,
    "can_sell_qty": 100.0,
    "pl_val": 0.0,
    "pl_ratio": 0.0
  },
  "cash": 100000.0,
  "summary": "盘中交易执行摘要",
  "tags": ["盘中突破", "量能放大"]
}
```

枚举约束：
- `result`: `buy` / `sell` / `hold`
- `confidence`: `high` / `medium` / `low`

与交易 Agent 最小决策输出对齐（用于机器人执行前判定）：

```json
{
  "result": "buy/sell/hold",
  "confidence": "high/medium/low",
  "report_id": "report_id"
}
```

---

## 创建盘中交易决策报告（createIntradayTradeDecisionReport）

### 接口定义

| 项目 | 说明 |
|------|------|
| **URL** | `/createIntradayTradeDecisionReport` |
| **方法** | `POST` |
| **认证** | Header：`Authorization: Bearer <API_KEY>`；Body：必填 `mcp_token` |

### 请求体参数

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| **mcp_token** | string | 是 | 用户 MCP 令牌。 |
| **code** | string | 是 | 标的代码，如 `00700.HK`。 |
| **stock_name** | string | 是 | 标的名称。 |
| **bot_id** | string | 是 | 触发该决策的机器人 ID。 |
| **bot_name** | string | 否 | 机器人名称；未传则写入 `""`。 |
| **bot_type** | string | 否 | 机器人类型；未传则写入 `""`。 |
| **result** | string | 是 | `buy` / `sell` / `hold`。 |
| **confidence** | string | 是 | `high` / `medium` / `low`。 |
| **reason** | string | 是 | 判定依据。 |
| **report** | string | 是 | 决策报告全文。 |
| **trade_type** | string | 否 | 交易类型（如“信号买入”）。 |
| **price** | number/string | 否 | 决策参考价格。 |
| **qty** | number/string | 否 | 下单数量（交易数量）。 |
| **position** | object | 否 | 决策时持仓快照对象（建议与 DCA `position` 结构一致）。 |
| **cash** | number/string | 否 | 决策时可用现金。 |
| **summary** | string | 否 | 决策摘要。 |
| **tags** | array | 否 | 标签数组。未传时默认 `[]`。 |

### 成功响应

HTTP `200`：

```json
{
  "code": 100,
  "message": "success",
  "data": {
    "report_id": "680bc8e7f54cf8a14f82a8a2"
  }
}
```

---

## 更新盘中交易决策报告（updateIntradayTradeDecisionReport）

### 接口定义

| 项目 | 说明 |
|------|------|
| **URL** | `/updateIntradayTradeDecisionReport` |
| **方法** | `POST` |
| **认证** | Header：`Authorization: Bearer <API_KEY>`；Body：必填 `mcp_token` |

### 请求体参数

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| **mcp_token** | string | 是 | 用户 MCP 令牌。 |
| **report_id** | string | 是 | 报告 ID。 |
| **bot_id** | string | 否 | 机器人 ID。 |
| **bot_name** | string | 否 | 机器人名称。 |
| **bot_type** | string | 否 | 机器人类型。 |
| **code** | string | 否 | 标的代码。 |
| **stock_name** | string | 否 | 标的名称。 |
| **result** | string | 否 | `buy` / `sell` / `hold`。 |
| **confidence** | string | 否 | `high` / `medium` / `low`。 |
| **reason** | string | 否 | 判定依据。 |
| **report** | string | 否 | 决策报告全文。 |
| **trade_type** | string | 否 | 交易类型。 |
| **price** | number/string | 否 | 决策参考价格。 |
| **qty** | number/string | 否 | 下单数量（交易数量）。 |
| **position** | object | 否 | 决策时持仓快照对象（建议与 DCA `position` 结构一致）。 |
| **cash** | number/string | 否 | 决策时可用现金。 |
| **summary** | string | 否 | 决策摘要。 |
| **tags** | array | 否 | 标签数组。 |

### 成功响应

HTTP `200`：

```json
{
  "code": 100,
  "message": "success"
}
```

---

## 删除盘中交易决策报告（deleteIntradayTradeDecisionReport）

### 接口定义

| 项目 | 说明 |
|------|------|
| **URL** | `/deleteIntradayTradeDecisionReport` |
| **方法** | `POST` |
| **认证** | Header：`Authorization: Bearer <API_KEY>`；Body：必填 `mcp_token` |

### 请求体参数

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| **mcp_token** | string | 是 | 用户 MCP 令牌。 |
| **report_id** | string | 是 | 报告 ID。 |

### 成功响应

HTTP `200`：

```json
{
  "code": 100,
  "message": "success"
}
```

---

## 查询盘中交易决策报告（getIntradayTradeDecisionReports）

### 接口定义

| 项目 | 说明 |
|------|------|
| **URL** | `/getIntradayTradeDecisionReports` |
| **方法** | `POST` |
| **认证** | Header：`Authorization: Bearer <API_KEY>`；Body：必填 `mcp_token` |

### 请求体参数

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| **mcp_token** | string | 是 | 用户 MCP 令牌。 |
| **report_id** | string | 否 | 按单条报告 ID 查询。 |
| **code** | string | 否 | 按标的代码筛选。 |
| **bot_id** | string | 否 | 按机器人 ID 筛选。 |
| **result** | string | 否 | `buy` / `sell` / `hold`。 |

### 成功响应

HTTP `200`，返回列表：

```json
{
  "code": 100,
  "message": "success",
  "data": [
    {
      "report_id": "680bc8e7f54cf8a14f82a8a2",
      "bot_id": "6781cb8309a2189f26d8866e",
      "bot_name": "我的DCA",
      "bot_type": "DCA",
      "code": "00700.HK",
      "stock_name": "腾讯控股",
      "result": "buy",
      "confidence": "high",
      "reason": "判定依据",
      "report": "针对上述分析信息的总结报告",
      "trade_type": "信号买入",
      "price": 500.0,
      "qty": 100.0,
      "position": {
        "opt": "hold",
        "order_id": "",
        "order_status": "",
        "order_time": "2026-04-26 15:30:00",
        "price": 500.0,
        "qty": 100.0,
        "can_sell_qty": 100.0,
        "pl_val": 0.0,
        "pl_ratio": 0.0
      },
      "cash": 100000.0,
      "summary": "盘中交易执行摘要",
      "tags": ["盘中突破", "量能放大"],
      "created_at": "2026-04-25T23:59:59.000000",
      "updated_at": "2026-04-26T00:05:12.000000"
    }
  ]
}
```

---

## 数据结构（post_market_report）

单条盘后分析报告字段如下（不含 `confidence`、`price`、`position`、`cash`）：

```json
{
  "code": "00700.HK",
  "stock_name": "腾讯控股",
  "session_date": "2026-05-09",
  "session_bias": "bullish",
  "change_pct": 0.42,
  "trade_summary": "今日交易执行与节奏复盘",
  "market_summary": "今日行情走势与量能结构复盘",
  "experience_summary": "可复用的经验与教训",
  "report": "盘后完整正文（可与上述三块呼应）",
  "summary": "一句话总览（可选）",
  "bot_id": "6781cb8309a2189f26d8866e",
  "bot_name": "我的DCA",
  "bot_type": "DCA",
  "vs_pre_market": "aligned",
  "pre_market_report_id": "680bc8e7f54cf8a14f82a8a2",
  "tags": ["缩量", "冲高回落"]
}
```

枚举约束：

- **`session_bias`**（当日涨跌/盘面倾向）：`bullish` / `bearish` / `neutral`
- **`vs_pre_market`**（与盘前观点对照，可选）：`aligned` / `partial` / `contradicted` / `na`

---

## 创建盘后分析报告（createPostMarketReport）

### 接口定义

| 项目 | 说明 |
|------|------|
| **URL** | `/createPostMarketReport` |
| **方法** | `POST` |
| **认证** | Header：`Authorization: Bearer <API_KEY>`；Body：必填 `mcp_token` |

### 请求体参数

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| **mcp_token** | string | 是 | 用户 MCP 令牌。 |
| **code** | string | 是 | 标的代码，如 `00700.HK`。 |
| **stock_name** | string | 是 | 标的名称。 |
| **session_date** | string | 是 | 交易日，建议 `YYYY-MM-DD`。 |
| **session_bias** | string | 是 | `bullish` / `bearish` / `neutral`。 |
| **trade_summary** | string | 是 | 今日交易总结。 |
| **market_summary** | string | 是 | 今日行情总结。 |
| **experience_summary** | string | 是 | 经验总结。 |
| **report** | string | 是 | 盘后报告全文。 |
| **change_pct** | number | 否 | 当日涨跌幅（%）。 |
| **summary** | string | 否 | 一句话摘要。 |
| **bot_id** | string | 否 | 关联机器人 ID；未传则写入 `""`。 |
| **bot_name** | string | 否 | 机器人名称；未传则写入 `""`。 |
| **bot_type** | string | 否 | 机器人类型；未传则写入 `""`。 |
| **vs_pre_market** | string | 否 | `aligned` / `partial` / `contradicted` / `na`。 |
| **pre_market_report_id** | string | 否 | 关联的盘前报告 `report_id`。 |
| **tags** | array | 否 | 标签；未传时默认 `[]`。 |

### 成功响应

HTTP `200`：

```json
{
  "code": 100,
  "message": "success",
  "data": {
    "report_id": "680bc8e7f54cf8a14f82a8a2"
  }
}
```

---

## 更新盘后分析报告（updatePostMarketReport）

### 接口定义

| 项目 | 说明 |
|------|------|
| **URL** | `/updatePostMarketReport` |
| **方法** | `POST` |
| **认证** | Header：`Authorization: Bearer <API_KEY>`；Body：必填 `mcp_token` |

### 请求体参数

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| **mcp_token** | string | 是 | 用户 MCP 令牌。 |
| **report_id** | string | 是 | 报告 ID。 |
| **code** | string | 否 | 标的代码。 |
| **stock_name** | string | 否 | 标的名称。 |
| **session_date** | string | 否 | 交易日。 |
| **session_bias** | string | 否 | `bullish` / `bearish` / `neutral`。 |
| **trade_summary** | string | 否 | 今日交易总结。 |
| **market_summary** | string | 否 | 今日行情总结。 |
| **experience_summary** | string | 否 | 经验总结。 |
| **report** | string | 否 | 盘后报告全文。 |
| **summary** | string | 否 | 一句话摘要。 |
| **change_pct** | number / null | 否 | 涨跌幅；传 `null` 或空字符串可清空已保存值。 |
| **bot_id** | string | 否 | 机器人 ID。 |
| **bot_name** | string | 否 | 机器人名称。 |
| **bot_type** | string | 否 | 机器人类型。 |
| **vs_pre_market** | string / null | 否 | 对照枚举；传 `null` 或空字符串可清空。 |
| **pre_market_report_id** | string | 否 | 关联盘前报告 ID。 |
| **tags** | array | 否 | 标签数组。 |

### 成功响应

HTTP `200`：

```json
{
  "code": 100,
  "message": "success"
}
```

---

## 删除盘后分析报告（deletePostMarketReport）

### 接口定义

| 项目 | 说明 |
|------|------|
| **URL** | `/deletePostMarketReport` |
| **方法** | `POST` |
| **认证** | Header：`Authorization: Bearer <API_KEY>`；Body：必填 `mcp_token` |

### 请求体参数

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| **mcp_token** | string | 是 | 用户 MCP 令牌。 |
| **report_id** | string | 是 | 报告 ID。 |

### 成功响应

HTTP `200`：

```json
{
  "code": 100,
  "message": "success"
}
```

---

## 查询盘后分析报告（getPostMarketReports）

### 接口定义

| 项目 | 说明 |
|------|------|
| **URL** | `/getPostMarketReports` |
| **方法** | `POST` |
| **认证** | Header：`Authorization: Bearer <API_KEY>`；Body：必填 `mcp_token` |

### 请求体参数

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| **mcp_token** | string | 是 | 用户 MCP 令牌。 |
| **report_id** | string | 否 | 按单条报告 ID 查询。 |
| **code** | string | 否 | 按标的代码筛选。 |
| **bot_id** | string | 否 | 按机器人 ID 筛选。 |
| **session_date** | string | 否 | 按交易日筛选。 |
| **session_bias** | string | 否 | `bullish` / `bearish` / `neutral`。 |

### 成功响应

HTTP `200`，返回列表：

```json
{
  "code": 100,
  "message": "success",
  "data": [
    {
      "report_id": "680bc8e7f54cf8a14f82a8a2",
      "code": "00700.HK",
      "stock_name": "腾讯控股",
      "session_date": "2026-05-09",
      "session_bias": "bullish",
      "change_pct": 0.42,
      "trade_summary": "今日交易执行与节奏复盘",
      "market_summary": "今日行情走势与量能结构复盘",
      "experience_summary": "可复用的经验与教训",
      "report": "盘后完整正文",
      "summary": "一句话总览",
      "bot_id": "6781cb8309a2189f26d8866e",
      "bot_name": "我的DCA",
      "bot_type": "DCA",
      "vs_pre_market": "aligned",
      "pre_market_report_id": "680bc8e7f54cf8a14f82a8a2",
      "tags": ["缩量", "冲高回落"],
      "created_at": "2026-05-09T18:30:00.000000",
      "updated_at": "2026-05-09T18:35:12.000000"
    }
  ]
}
```

---

## 失败与业务码（盘中交易决策报告）

| code | HTTP 状态 | 说明 |
|------|------------|------|
| **401** | 400 | 缺少 `mcp_token` 或必填 ID 参数。 |
| **102** | 401 | `mcp_token` 无效或用户不存在。 |
| **101** | 400 | `report_id` 非法。 |
| **103** | 400 | 未找到报告或无权限访问。 |
| **400** | 400 | 参数非法（如枚举值不在允许范围、缺少必填字段）。 |
| **500** | 500 | 服务端异常。 |

盘后分析报告接口（`/createPostMarketReport`、`/updatePostMarketReport`、`/deletePostMarketReport`、`/getPostMarketReports`）使用**相同**业务码约定。

---

## 相关：日报统一查询（reportServer）

本仓库另提供独立服务 **`reportServer.py`**（FastAPI，默认端口 **6100**，可由 `REPORT_SERVER_PORT` 覆盖），用于按用户 **user_id** 一次拉取盘前 / 盘中 / 盘后报告并支持 `bot_name`、`bot_type`、`stock_name`、日期等筛选。与上述 Market 接口**不同机**部署时，请通过 `start.sh` 中 **reportServer** 项启动，并配置 `REPORT_SERVER_API_KEY`。

| 项目 | 说明 |
|------|------|
| **根地址** | `http://<host>:6100`（以实际为准） |
| **鉴权** | `Authorization: Bearer <REPORT_SERVER_API_KEY>` |
| **主接口** | `POST /reports/daily`（Body 含 `user_id` 与筛选条件） |
| **健康检查** | `GET /health` |
| **文档** | `GET /docs`（Swagger）、`GET /openapi.json` |

报告正文仍以 Market API 写入 MongoDB 为准；列表展示优先使用文档中的 **`bot_id` / `bot_name` / `bot_type`**，缺省时由服务端按 `bot_id` 回查机器人集合。

