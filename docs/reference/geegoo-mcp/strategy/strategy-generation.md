# MCP API：策略生成接口说明（GRID / DCA）

本文档描述通过 MCP（Skills）调用的 **策略生成接口**：**generateGridStrategy**（GRID 网格策略生成）与 **generateDCAStrategy**（DCA 定投策略生成）。MCP 层校验必填参数后转发至 AIServer，并原样返回其结构化 JSON。

---

## 0. 使用场景

两个接口的共同场景为：**基于用户提供的股票，进行 GRID（网格）与 DCA（定投）的策略设置，并基于回测数据给出最佳参数**。

- **generateGridStrategy**：用户传入股票代码与名称后，系统使用指定月数的历史行情进行回测，结合 LLM 分析给出该标的是否适合做网格、以及推荐的网格参数（如上下限价、网格数量等）。
- **generateDCAStrategy**：用户传入股票代码、名称及选定的信号 ID 后，系统基于回测数据评估该信号在该标的上的适用性，并给出动态/固定止盈止损的最佳参数建议，用于 DCA 定投策略配置。

因此，调用方（如 Skills）可先让用户选择标的，再按需调用 GRID 或 DCA 策略生成接口，用返回的最佳参数完成策略创建或推荐。

**公共约定**：认证、标的搜索 **`/searchCode`**、**`signal_id`** 来源（**getIndexSignalForSkill** / **getSignalCombinationForSkill**）等，见 [`common.md`](common.md)。

---

## 1. 通用说明

### 1.1 基础路径与认证

- **基础路径**：geegoo mcp 根地址（默认示例：`http://0.0.0.0:5700`）
- **认证**：请求头 `Authorization: Bearer <API_KEY>`（MCP 的 API Key，见 `mcpAPIServer` 配置）
  - 缺少或格式错误返回 `401`；Key 无效返回 `401`。

MCP 转发至 AIServer 时使用 `Config/APIConnection.py` 中的 `--aidata_server_ip` / `--aidata_server_port`、`--aidata_server_api_key`。

### 1.2 响应格式

- **HTTP 状态码**：成功为 `200`，业务错误也为 `200`，通过 body 中的 `code` 区分；上游不可用时 MCP 返回 `502`。
- **成功**：`code: 100`，`message` 为提示文案，`data` 为策略结果对象（结构见各接口下文）。
- **失败**：`code: 101`，`message` 为错误说明，无 `data` 或 `data` 为空；上游调用失败时 `code: 502`。

---

## 2. 接口一览

| 接口 | 方法 | 说明 |
|------|------|------|
| `/generateGridStrategy` | POST | GRID 网格策略生成 |
| `/generateDCAStrategy`  | POST | DCA 定投策略生成（含信号评估与止盈止损） |

返回 `data` 结构由 AIServer 决定，见下文各接口说明。

---

## 3. generateGridStrategy（GRID 策略生成）

### 3.1 请求

- **URL**：`/generateGridStrategy`
- **方法**：`POST`
- **Content-Type**：`application/json`
- **Headers**：`Authorization: Bearer <API_KEY>`（MCP API Key）

**请求体：**

| 字段 | 类型 | 必填 | 默认 | 说明 |
|------|------|------|------|------|
| code | string | 是 | - | 股票代码，如 `00700.HK`、`518880.SH`、`AAPL.US`；可先通过 MCP **`/searchCode`** 解析（见 [`common.md`](common.md)） |
| name | string | 是 | - | 股票名称（展示与提示词用） |
| months_back | number | 否 | 1 | 回测月数，代表使用该策略回测的行情月份时长 |
| language | string | 否 | cn | 返回语言：`cn`（简体）、`en`（英文）、`hk`（繁体）时 reason 为对应语言单字符串；`all` 或其它时保留多语言对象 `{ cn, en, hk }` |

**请求体示例：**

```json
{
  "code": "00700.HK",
  "name": "腾讯控股",
  "months_back": 3,
  "language": "cn"
}
```

### 3.2 响应

**成功（code: 100）**

- `data` 结构：

| 字段 | 类型 | 说明 |
|------|------|------|
| create_time | string | 生成时间（如 `Sun, 15 Mar 2026 20:58:09 GMT`） |
| model_name | string | 使用的 LLM 模型名 |
| suitable | boolean | 是否适合做网格 |
| param | object | 网格参数，见下表 |
| param.upper_limit_price | number | 网格上限价格 |
| param.lower_limit_price | number | 网格下限价格 |
| param.grid_num | number | 网格数量 |
| reason | string 或 object | 分析理由。当请求 `language=cn`/`en`/`hk` 时为**对应语言单字符串**；当 `language=all` 等时为对象 `{ cn, en, hk }` |
| data | array | 回测用的 K 线/行情数组，每项见下表 |

**data.data 数组中每项（K 线）：**

| 字段 | 类型 | 说明 |
|------|------|------|
| trade_date | string | 交易时间，如 `2025/12/15 11:30` |
| open | number | 开盘价 |
| high | number | 最高价 |
| low | number | 最低价 |
| close | number | 收盘价 |
| change | number | 涨跌额 |
| change_pct | number | 涨跌幅（%） |
| volume | number | 成交量 |

**成功响应示例（language=cn 时，reason 为中文单字符串）：**

```json
{
  "code": 100,
  "data": {
    "create_time": "Sun, 15 Mar 2026 20:58:09 GMT",
    "data": [{"trade_date": "2025/12/15 11:30", "open": 606.0, "high": 608.5, "low": 604.5, "close": 606.5, "change": 1.0, "change_pct": 0.16, "volume": 1656800}, "..."],
    "model_name": "MiniMax-M2.5",
    "param": {"grid_num": 7, "lower_limit_price": 500, "upper_limit_price": 640},
    "reason": "经过对腾讯控股（00700.HK）近期股价数据的分析，我认为该股票目前适合实施网格交易策略。原因如下：...",
    "suitable": true
  },
  "message": "GRID策略生成完成"
}
```

**失败（code: 101）**

- `message` 可能取值示例：
  - `"缺少 code 或 name"`
  - `"无法获取股票 {code} ({name}) 的Grid策略数据"`
  - `"无法获取股票 {code} ({name}) 的Grid策略提示词或数据"`
  - 或 LLM/解析异常信息（如 JSON 解析错误、缺少必填字段等）

**MCP 特有**：AIServer 不可用时，MCP 返回 HTTP 502，`code: 502`。

### 3.3 请求示例

```bash
# 默认中文（language=cn）
curl -X POST "http://localhost:5700/generateGridStrategy" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <API_KEY>" \
  -d '{"code": "00700.HK", "name": "腾讯控股", "months_back": 3}'

# 指定英文或繁体：language=en / language=hk
# 需要多语言对象时传 language=all
curl -X POST "http://localhost:5700/generateGridStrategy" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <API_KEY>" \
  -d '{"code": "00700.HK", "name": "腾讯控股", "months_back": 3, "language": "all"}'
```

### 3.4 与上游（AIServer）的关系

- MCP 校验 `code`、`name` 后，将 `code`、`name`、`months_back` 组装为 JSON 转发至 AIServer 的 `POST {AIData_Server}/generateGridStrategy`。
- 当请求 `language=cn`（默认）、`en` 或 `hk` 时，MCP 在收到 AIServer 返回后，将 `data.reason` 收敛为对应语言的单字符串再返回；`language=all` 或其它值时原样返回多语言 `reason` 对象。

---

## 4. generateDCAStrategy（DCA 策略生成）

### 4.1 请求

- **URL**：`/generateDCAStrategy`
- **方法**：`POST`
- **Content-Type**：`application/json`
- **Headers**：`Authorization: Bearer <API_KEY>`（MCP API Key）

**请求体：**

| 字段 | 类型 | 必填 | 默认 | 说明 |
|------|------|------|------|------|
| code | string | 是 | - | 股票代码，如 `00700.HK`；可先通过 MCP **`/searchCode`** 解析（见 [`common.md`](common.md)） |
| name | string | 是 | - | 股票名称（展示与提示词用） |
| months_back | number | 否 | 1 | 回测月数，代表使用该策略回测的行情月份时长 |
| signal_id | string | 是 | - | 信号 ID，可来源于 **getIndexSignalForSkill** 或 **getSignalCombinationForSkill**（见 [`common.md`](common.md)「信号查询」）返回的某一信号 |
| language | string | 否 | cn | 返回语言：`cn`（简体）、`en`（英文）、`hk`（繁体）时，多语言字段收敛为对应语言字符串；`all` 或其它时保留多语言对象 `{ cn, en, hk }` |

**请求体示例：**

```json
{
  "code": "00700.HK",
  "name": "腾讯控股",
  "months_back": 3,
  "signal_id": "662d0424c4cee7ffb800d0af",
  "language": "cn"
}
```

### 4.2 响应

**成功（code: 100）**

- `data` 结构（与 StrategyServer /getDCAAnalysis 一致）：

| 字段 | 类型 | 说明 |
|------|------|------|
| create_time | string | 生成时间（如 `Sun, 15 Mar 2026 21:18:34 GMT`） |
| model_name | string | 使用的 LLM 模型名 |
| signal_id | string | 请求中的 signal_id |
| signal_name | string | 信号名称（如「SAR信号配套MACD直方图趋势」） |
| trend_conclusion | object | 趋势结论 |
| trend_conclusion.suitable | boolean | 当前行情是否适合趋势交易 |
| trend_conclusion.reason | string 或 object | 理由。`language=cn`/`en`/`hk` 时为**单语言字符串**；`language=all` 时为 `{ cn, en, hk }` |
| signal_evaluation | object | 信号评估 |
| signal_evaluation.suitable | boolean | 当前信号是否适合该标的 |
| signal_evaluation.reason | string 或 object | 理由，同上 |
| dynamicParam | object | 动态止盈止损参数（BBAND 止盈、SAR 止损等） |
| dynamicParam.tp | object | 动态止盈配置（如 `index`、`param`、`type`） |
| dynamicParam.sl | object | 动态止损配置 |
| dynamicParam.reason | string 或 object | 选择理由，同上 |
| fixedParam | object | 固定止盈止损参数（百分比） |
| fixedParam.tp | object | 固定止盈（如 `fix_tp`） |
| fixedParam.sl | object | 固定止损（如 `fix_sl`） |
| fixedParam.reason | string 或 object | 选择理由，同上 |
| comparison | string 或 object | 动态 vs 固定策略对比说明，同上 |
| signal | object | 信号定义：`name`（多语言时可收敛）、`info`、`buy_signal` 等 |
| data | object | 内部含 `data`（策略提示模板字符串）与 `signal`（K 线/指标数组），供参考 |

**data.data.signal 数组中每项（K 线/指标）：**

| 字段 | 类型 | 说明 |
|------|------|------|
| trade_date | string | 交易时间（GMT） |
| open, high, low, close | number | 开/高/低/收 |
| change, change_pct | number | 涨跌额、涨跌幅 |
| volume | number | 成交量 |
| MACD, MACD_Hist, MACD_S, SAR 等 | number | 指标值（依信号而定） |

**成功响应示例（language=cn 时，多语言字段为单字符串）：**

```json
{
  "code": 100,
  "data": {
    "comparison": "动态策略优势：BBAND止盈能够根据市场波动自动调整...",
    "create_time": "Sun, 15 Mar 2026 21:18:34 GMT",
    "data": { "data": "...", "signal": [...] },
    "dynamicParam": {
      "reason": "从输入数据中分析，腾讯控股（00700.HK）在2025年12月至2026年3月期间...",
      "sl": { "index": "SAR", "param": { "acceleration": 0.02, "maximum": 0.2 }, "type": "signal" },
      "tp": { "index": "BBAND", "param": { "matype": 2, "period": 20 }, "type": "signal" }
    },
    "fixedParam": {
      "reason": "基于输入数据的历史波动率分析...",
      "sl": { "fix_sl": 4.5 },
      "tp": { "fix_tp": 7.5 }
    },
    "model_name": "MiniMax-M2.5",
    "signal": {
      "buy_signal": [...],
      "info": "指标组合\n1. SAR抛物线信号...",
      "name": "SAR信号配套MACD直方图趋势"
    },
    "signal_evaluation": { "suitable": false, "reason": "根据SAR和MACD指标分析..." },
    "signal_id": "662d0424c4cee7ffb800d0af",
    "signal_name": "SAR信号配套MACD直方图趋势",
    "trend_conclusion": { "suitable": false, "reason": "从输入数据来看，股价在2025年12月15日至2026年3月13日期间..." }
  },
  "message": "DCA策略生成完成"
}
```

**失败（code: 101）**

- `message` 可能取值示例：
  - `"缺少 code 或 name"`
  - `"缺少 signal_id"`
  - `"无法获取股票 {code} ({name}) 的DCA策略数据"`
  - `"无法获取股票 {code} ({name}) 的DCA策略提示词"`
  - 或 LLM/解析异常（如信号分析或止盈止损 JSON 缺少必填字段等）。上游 AIServer 不可用时，同样以 `code: 101` 及相应错误信息返回。

### 4.3 请求示例

```bash
# 默认仅返回中文（language=cn）
curl -X POST "http://localhost:5700/generateDCAStrategy" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <API_KEY>" \
  -d '{"code": "00700.HK", "name": "腾讯控股", "months_back": 3, "signal_id": "662d0424c4cee7ffb800d0af"}'

# 指定英文返回
curl -X POST "http://localhost:5700/generateDCAStrategy" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <API_KEY>" \
  -d '{"code": "00700.HK", "name": "腾讯控股", "months_back": 3, "signal_id": "662d0424c4cee7ffb800d0af", "language": "en"}'

# 保留多语言对象（language=all）
curl -X POST "http://localhost:5700/generateDCAStrategy" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <API_KEY>" \
  -d '{"code": "00700.HK", "name": "腾讯控股", "months_back": 3, "signal_id": "662d0424c4cee7ffb800d0af", "language": "all"}'
```

### 4.4 与上游（AIServer）的关系

- MCP 校验 `code`、`name`、`signal_id` 后，将 `code`、`name`、`months_back`、`signal_id` 组装为 JSON 转发至 AIServer 的 `POST {AIData_Server}/generateDCAStrategy`。
- 当请求 `language=cn`（默认）、`en` 或 `hk` 时，MCP 在收到 AIServer 返回后，将 `data` 中的多语言字段（`comparison`、`dynamicParam.reason`、`fixedParam.reason`、`signal_evaluation.reason`、`trend_conclusion.reason`、`signal.name`）收敛为对应语言的单字符串再返回；`language=all` 或其它值时原样返回多语言对象。

---

## 5. 依赖与错误来源（AIServer 侧）

- **Prompt/数据服务**：拉取模板或价格数据失败会导致“无法获取…数据”或“无法获取…提示词或数据”。
- **LLM**：调用失败或返回非预期格式会以 `code: 101` 及异常信息返回。
- **解析**：若 LLM 输出不符合约定结构，会以 `code: 101` 返回。

---

## 6. 参考

| 说明 | 路径 |
|------|------|
| MCP 接口定义 | `mcpAPIServer.py`：`/generateGridStrategy`、`/generateDCAStrategy` |
