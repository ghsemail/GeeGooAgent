# MCP API：策略回测接口说明（loopBackStrategy）

本文档描述通过 MCP（Skills）调用的 **策略回测接口**：**loopBackStrategy**。MCP 层校验必填参数后转发至 Signal Server，并统一返回格式（成功为 `code: 100` + `data`，失败为 `code: 101` 或 `502` + `message`）。

---

## 0. 使用场景

用户或 Skill 已确定策略类型（DCA 定投 或 GRID 网格）及对应参数（如 DCA 的买卖信号、止盈止损，或 GRID 的网格参数）后，需要**基于历史行情对该策略进行回测**，得到期末资产、盈亏、收益率等结果，用于评估策略表现或对比不同参数。

- **DCA 回测**：传入 `strategy_type="dca"`，以及 `signal`（买入信号配置）、`sl_tp`（止盈止损配置）、标的 `code`、`frequency`、初始资金 `fund`、回测月数 `months_back` 等。
- **GRID 回测**：传入 `strategy_type="grid"`，以及 `grid_param`（网格参数）、标的与资金等。

因此，调用方（如 Skills）可在生成或选定策略参数后，调用本接口执行回测，并将返回的 `data`（如 `finalValue`、`profit`、`profit_rate`）展示给用户或用于后续逻辑。

**公共约定**：认证、**`frequency`**、与 **`signal`** 相关的 Admin 接口索引等，见 [`common.md`](common.md)。

---

## 1. 通用说明

### 1.1 基础路径与认证

- **基础路径**：geegoo mcp 根地址（默认示例：`http://0.0.0.0:5700`）
- **认证**：请求头 `Authorization: Bearer <API_KEY>`（MCP 的 API Key，见 `mcpAPIServer` 配置）
  - 缺少或格式错误返回 `401`；Key 无效返回 `401`。

MCP 转发至 Signal Server 时使用 `Config/APIConnection.py` 中的 `--signal_server_ip` / `--signal_server_port`、`--signal_server_api_key`。

### 1.2 响应格式

- **HTTP 状态码**：成功为 `200`，业务错误也为 `200`，通过 body 中的 `code` 区分；上游不可用时 MCP 返回 `502`。
- **成功**：`code: 100`，`message` 为提示文案（如「回测完成」），`data` 为 Signal Server 返回的回测结果（通常含 `finalValue`、`profit`、`profit_rate` 等，具体由上游决定）。
- **失败**：`code: 101` 表示参数错误或上游返回业务错误，`message` 为错误说明；`code: 502` 表示调用 Signal Server 失败或返回空；`code: 500` 表示 MCP 内部异常。

---

## 2. 接口一览

| 接口 | 方法 | 说明 |
|------|------|------|
| `/loopBackStrategy` | POST | DCA 或 GRID 策略历史回测 |

---

## 3. loopBackStrategy（策略回测）

### 3.1 请求

- **URL**：`/loopBackStrategy`
- **方法**：`POST`
- **Content-Type**：`application/json`
- **Headers**：`Authorization: Bearer <API_KEY>`（MCP API Key）

**请求体：**

| 字段 | 类型 | 必填 | 默认 | 说明 |
|------|------|------|------|------|
| strategy_type 或 type | string | 是 | - | 策略类型（必填）：`dca` 定投策略，`grid` 网格策略 |
| code | string | 是 | - | 股票代码（必填），如 `00700.HK`、`518880.SH`、`AAPL.US` |
| frequency | string | 是 | - | 时间频率（必填），如 `5m`、`60m`、`daily`；常用取值见 [`common.md`](common.md) |
| fund | number | 是 | - | 初始资金（必填） |
| months_back | number | 是 | - | 回测月数（必填） |
| base_order_size | number | 否 | 100 | 基础订单数量（可选） |
| signal | array | 条件必填 | - | DCA 买入信号配置（当 type=dca 时必填），结构见下文 |
| sl_tp | object | 条件必填 | - | DCA 止盈止损配置（当 type=dca 时必填），结构见下文 |
| grid_param | object | 条件必填 | - | Grid 网格参数（当 type=grid 时必填），结构见下文 |

**signal（DCA 买入信号）**：数组，每项为指标配置对象。

| 字段 | 类型 | 说明 |
|------|------|------|
| index | string | 指标名，如 `SAR`、`MACD`、`BBAND` |
| type | string | **`signal`**：信号；**`flag`**：趋势 |
| param | object | 指标参数，如 SAR 的 `acceleration`、`maximum`；MACD 的 `fastPeriod`、`signalPeriod`、`slowPeriod` |

**sl_tp（DCA 止盈止损）**：对象，**顶层需带 `type`**：`fix`（固定比例）或 `dynamic`（动态指标）。

| 字段 | 类型 | 说明 |
|------|------|------|
| type | string | `fix` 固定止盈止损，或 `dynamic` 动态止盈止损 |
| tp | object | 止盈配置，见下 |
| sl | object | 止损配置，见下 |

- **当 type=`fix`**：`tp` 为 `{ "fix_tp": number }`（固定止盈比例，如 5 表示 5%），`sl` 为 `{ "fix_sl": number }`（固定止损比例，如 3 表示 3%）。
- **当 type=`dynamic`**：`tp`/`sl` 为 `{ "index": "BBAND"|"SAR" 等, "type": "signal", "param": {...} }`。

**grid_param（GRID 网格参数）**：对象。

| 字段 | 类型 | 说明 |
|------|------|------|
| upper_limit_price | number | 网格上限价格 |
| lower_limit_price | number | 网格下限价格 |
| grid_num | number | 网格数量 |

**DCA 请求体示例（固定止盈止损）：**

```json
{
  "type": "dca",
  "code": "00700.HK",
  "frequency": "60m",
  "fund": 100000,
  "months_back": 3,
  "base_order_size": 100,
  "signal": [
    { "index": "SAR", "param": { "acceleration": "0.02", "maximum": "0.2" }, "type": "signal" }
  ],
  "sl_tp": {
    "type": "fix",
    "sl": { "fix_sl": 3 },
    "tp": { "fix_tp": 5 }
  }
}
```

**DCA 请求体示例（动态止盈止损）：**

```json
{
  "type": "dca",
  "code": "518880.SH",
  "frequency": "60m",
  "fund": 100000,
  "months_back": 3,
  "signal": [
    { "index": "SAR", "param": { "acceleration": "0.02", "maximum": "0.2" }, "type": "signal" },
    { "index": "MACD", "param": { "fastPeriod": "12", "signalPeriod": "9", "slowPeriod": "26" }, "type": "flag" }
  ],
  "sl_tp": {
    "type": "dynamic",
    "tp": { "index": "BBAND", "type": "signal", "param": {} },
    "sl": { "index": "SAR", "type": "signal", "param": {} }
  }
}
```

**GRID 请求体示例：**

```json
{
  "type": "grid",
  "code": "00700.HK",
  "frequency": "5m",
  "fund": 50000,
  "months_back": 6,
  "base_order_size": 100,
  "grid_param": {
    "upper_limit_price": 650,
    "lower_limit_price": 550,
    "grid_num": 7
  }
}
```

### 3.2 响应

**成功（code: 100）**

- `data` 为上游回测结果，结构如下：

| 字段 | 类型 | 说明 |
|------|------|------|
| code | string | 股票代码 |
| frequency | string | 时间频率 |
| cash | number | 初始资金 |
| base_order_size | number | 基础订单数量 |
| finalValue | number | 最终资产 |
| profit | number | 盈利金额 |
| profit_rate | number | 回报率（百分比） |
| drawdown | number | 最大回撤（百分比） |
| moneydown | number | 最大回撤金额 |
| annualized_return | number | 年化收益率（百分比） |

**成功响应示例：**

```json
{
  "code": 100,
  "message": "回测完成",
  "data": {
    "code": "00700.HK",
    "frequency": "60m",
    "cash": 100000,
    "base_order_size": 100,
    "finalValue": 105000.00,
    "profit": 5000.00,
    "profit_rate": 5.00,
    "drawdown": 2.50,
    "moneydown": 2500.00,
    "annualized_return": 20.00
  }
}
```

**失败（code: 101）**

- `message` 可能取值示例：
  - `"缺少 strategy_type（或 type）"`
  - `"缺少 code"`
  - `"缺少 frequency"`
  - `"缺少 fund"`
  - `"缺少 months_back"`
  - `"DCA 策略缺少必填参数: signal"` / `"DCA 策略缺少必填参数: sl_tp"`
  - `"Grid 策略缺少必填参数: grid_param"`
  - `"不支持的策略类型: xxx，仅支持 dca 或 grid"`
  - 或上游返回的 `error` 文案（如「回测执行失败: ...」）

**MCP 特有**：Signal Server 不可用或返回空时，MCP 返回 HTTP 502，`code: 502`。

### 3.3 请求示例

```bash
# DCA 回测（固定止盈止损）
curl -X POST "http://localhost:5700/loopBackStrategy" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <API_KEY>" \
  -d '{
    "type": "dca",
    "code": "00700.HK",
    "frequency": "60m",
    "fund": 100000,
    "months_back": 3,
    "signal": [{ "index": "SAR", "type": "signal", "param": {} }],
    "sl_tp": { "type": "fix", "sl": { "fix_sl": 3 }, "tp": { "fix_tp": 5 } }
  }'

# GRID 回测
curl -X POST "http://localhost:5700/loopBackStrategy" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <API_KEY>" \
  -d '{
    "type": "grid",
    "code": "00700.HK",
    "frequency": "5m",
    "fund": 100000,
    "months_back": 3,
    "grid_param": { "upper_limit_price": 650, "lower_limit_price": 550, "grid_num": 7 }
  }'
```

### 3.4 与上游（Signal Server）的关系

- 使用入口：回测系统 - 策略回测。上游根据 `type`（dca 或 grid）执行对应回测并返回结果。
- MCP 校验 `strategy_type`（或 `type`）、`code`、`frequency`、`fund`、`months_back`，以及按策略类型校验 `signal`+`sl_tp`（dca）或 `grid_param`（grid）。
- 将参数组装为 JSON（字段 `type` 取 `strategy_type` 的值），转发至 Signal Server 的 `POST {Signal_Server}/loopBackStrategy`。
- 若上游返回体中含有 `error` 字段，MCP 将其转为 `code: 101` 与 `message`；否则将上游返回体整体放入 `data`，并返回 `code: 100`。

---

## 4. 依赖与错误来源（Signal Server 侧）

- **行情数据**：拉取历史 K 线失败可能导致回测失败或上游返回 `error`。
- **参数格式**：`signal`、`sl_tp`、`grid_param` 格式不符合上游约定时，由 Signal Server 返回错误，MCP 原样转成 `code: 101`。
- **超时**：回测计算时间较长时，MCP 对 Signal Server 的请求超时时间为 120 秒，超时则返回 `502`。

---

## 5. 参考

| 说明 | 路径 |
|------|------|
| MCP 接口定义 | `mcpAPIServer.py`：`/loopBackStrategy`、`_forward_to_signal_server` |
| 上游服务配置 | `Config/APIConnection.py`：`Signal_Server`、`Signal_Server_API_Key` |
