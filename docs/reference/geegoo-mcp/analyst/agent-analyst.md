# MCP API：AgentAnalyst 接口说明

## 概述

本文档描述通过 MCP（Skills）调用的 **AgentAnalyst 相关接口**：获取单项分析 Prompt 模板列表（**getSinglePromptTemplate**）、**用户自建竞品 / ETF 分析模板**（创建 / 修改 / 删除），以及**直接调用 AIServer 得到分析结果**（**getMCPAnalysis**）。均支持通过 **`mcp_token`** 解析用户身份；前几项由 MCP 将 **`user_id`** 转发至 **Prompt 服务**，getMCPAnalysis 转发至 **AIServer**。**getMCPAnalysis** 要求 **`period`**（枚举见下文）；**`language`** 可选，默认 **`cn`**。

- **基础路径**：GeeGooBot mcp-api 根地址（默认示例：`http://127.0.0.1:3120`）
- **认证方式**：请求头 `Authorization: Bearer <API_KEY>`（MCP 的 API Key，见 `mcpAPIServer` 配置）

**说明**：Prompt 服务地址在 `Config/APIConnection.py` 中配置（`--prompt_server_ip` / `--prompt_server_port`）。AIServer 地址与 API Key 在 `Config/APIConnection.py` 中配置（`--aidata_server_ip` / `--aidata_server_port`、`--aidata_server_api_key`），getMCPAnalysis 由 MCP 转发至 AIServer 并直接返回 LLM 分析结果。

**公共约定**：认证与 **`mcp_token`** 见 [`common.md`](common.md)「认证与用户身份」。下文 **约定与枚举** 给出 **`type`** / **TemplateType** / **`creator`** / **`period`** / **`attitude.analysis_period`**；各接口路径、请求体与示例见后续章节。

---

## 约定与枚举（三类 type · TemplateType · creator · period）

### 三类筛选（`getSinglePromptTemplate` 请求参数 `type`）

| 值 | 说明 |
|----|------|
| **index** | 指数类模板（库内文档字段 **`type`** 见下表 **TemplateType**） |
| **tech** | 技术类模板（机器人 **`attitude.analysis_prompt_list`** / **`analysis_period`** 常用） |
| **fundamental** | 基本面类模板 |

调用 **`POST /getSinglePromptTemplate`** 时需 **`mcp_token`** + 上述 **`type`** 之一；可选 **`period`** 缩小返回范围。

### TemplateType（库内字段 `type`）

每条 Prompt 文档另有字段 **`type`**，表示该条模板的**具体种类**（本文称为 **TemplateType**）。与筛选参数 **`type`**（**`index` / `tech` / `fundamental`**）对应关系如下：

| 筛选参数 **`type`** | 模板文档 **`type`**（TemplateType）取值 |
|---------------------|----------------------------------------|
| **index** | **`index`** |
| **tech** | **`price`**、**`kline`**、**`flag`**、**`industry`**、**`competitor`**、**`etf`**、**`template`** |
| **fundamental** | **`basic`**、**`risk`**、**`financial`**、**`industry`** |

说明：**`industry`** 在技术类与基本面类中均可出现，以文档实际 **`type`** 及所选筛选大类为准。

### creator（官方模板 / 用户自建）

| 值 | 说明 |
|----|------|
| **admin** | 官方预制模板 |
| **user** | 用户自建（如 **`/createCompetitorPromptTemplate`**、**`/createEtfPromptTemplate`** 等写入） |

列表通常按 **`creator`** 与 **`user_id`** 做作用域过滤；管理员可见更全量，普通用户可见 **`admin`** 或本人 **`user`** 模板（以实现为准）。

### period（筛选模板与分析请求共用）

**`getSinglePromptTemplate`**、模板文档、**`attitude.analysis_period`**、**getMCPAnalysis** / **getSingleAnalysis** 请求中的 **`period`** 等共用下列字符串：

| 值 | 说明 |
|----|------|
| **no_period** | 不按周期档位归类 / 无周期维度 |
| **minutes** | 分钟级 |
| **hourly** | 小时级 |
| **daily** | 日线 |
| **weekly** | 周线 |
| **monthly** | 月线 |
| **quarterly** | 季线 |
| **yearly** | 年线 |
| **longterm** | 长期 |

调用 **`getSinglePromptTemplate`** 时可不传 **`period`**（不按周期收窄，仍受 **`type`** 与作用域约束），也可传上表任一值。

### attitude.analysis_period（机器人 `attitude`）

DCA / GRID **Bot** 与 **Reminder** 的 **`attitude`** 中 **`analysis_period`**：定时刷新态度、单项分析时采用的周期，须为 **period** 表中之一种；默认 **`daily`**（与 **`Constants/Basic_bot.py`、`Basic_reminder.py`** 一致）。从 **`getSinglePromptTemplate`** 取 **`prompt_id`** 时，建议请求的 **`period`** 与本字段一致。

### 竞品 / ETF 自建接口（索引）

MCP 将 **`mcp_token`** 解析为 **`user_id`** 后转发 Prompt 服务同名路径（详见下文 **竞品分析 / ETF 用户 Prompt 模板**）。

| 竞品 | ETF |
|------|-----|
| **`/createCompetitorPromptTemplate`** | **`/createEtfPromptTemplate`** |
| **`/editCompetitorPromptTemplate`** | **`/editEtfPromptTemplate`** |
| **`/deleteCompetitorPromptTemplate`** | **`/deleteEtfPromptTemplate`** |

---

## getSinglePromptTemplate（获取单项分析 Prompt 模板列表）

返回 **switch 为 true** 的 Prompt 模板列表（按类型与用户作用域过滤）。MCP 层使用 **`mcp_token`** 解析 **`user_id`**，再转发至 Prompt 服务 **`/getSinglePromptTemplate`**（与 trading_operation、trading_app、SKILLServer 同路径语义；此处仅需 MCP API Key）。

### 接口定义

| 项目 | 说明 |
|------|------|
| **URL** | `/getSinglePromptTemplate` |
| **方法** | `POST` |
| **Content-Type** | `application/json` |
| **认证** | `Authorization: Bearer <API_KEY>`（MCP API Key） |

### 请求体参数（MCP）

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| **mcp_token** | string | 是 | 用户 MCP 令牌；MCP 解析为 **`user_id`** 后写入上游请求体。 |
| **type** | string | 是 | **`index`** / **`tech`** / **`fundamental`**，含义见上文 **约定与枚举 · 三类筛选**。 |
| **period** | string | 否 | 选填，取值见上文 **约定与枚举 · period**（如 **`no_period`**、**`daily`**、**`monthly`**）。不传则只按 **`type`** 与作用域返回、不按周期收窄。 |

配置机器人 **`attitude.analysis_prompt_list`** 时通常取 **`type`: `tech`**，并按产品要求传 **`period`**（如 **`monthly`**）。**`attitude.analysis_period`** 与 **getMCPAnalysis** / **getSingleAnalysis** 的 **`period`** 应与上文 **约定与枚举** 一致。

### 响应说明

- **成功**：HTTP 状态码与 Prompt 服务一致；响应体一般为 **JSON 数组**，每项通常含 **`prompt_id`**、**`type`**（TemplateType，如 **`competitor`**、**`etf`**）、**`creator`**（**`admin`** / **`user`**）、**`period`**、**`template`** 等（以实际返回为准）。
- **MCP 层错误**：缺少 **`mcp_token`** → 400；无效 **`mcp_token`** → 401；无效 **`type`** → 400；Prompt 服务不可用 → 502。

### 请求示例

```bash
# 技术类模板，并按 monthly 过滤（示例：与态度分析常用的月度日线级 Prompt 对齐）
curl -X POST "http://localhost:3120/getSinglePromptTemplate" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <API_KEY>" \
  -d '{"mcp_token": "your_mcp_token", "type": "tech", "period": "monthly"}'

# 不传 period，仅按 type 与用户作用域返回
curl -X POST "http://localhost:3120/getSinglePromptTemplate" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <API_KEY>" \
  -d '{"mcp_token": "your_mcp_token", "type": "tech"}'
```

### 与上游服务的关系

- MCP 请求体：**`mcp_token`** + **`type`** + 可选 **`period`**。
- 转发至 Prompt 服务：**`POST {PROMPT_SERVER_IP}/getSinglePromptTemplate`**，请求体为 **`{"user_id": "<解析得到的用户ID>", "type": "<index|tech|fundamental>", ...}`**，若调用方传了 **`period`** 则附带 **`period`**；请求头携带 Prompt 服务 API Key（由 `mcpAPIServer` 配置）。
- 管理员 **`user_id`** 在上游可得全量；普通用户按 **`creator`** 等作用域过滤（详见 Prompt 服务实现）。

---

## 竞品分析 / ETF 用户 Prompt 模板（创建 · 修改 · 删除）

用户可自建 **竞品分析**、**ETF 分析**两类 Prompt（**`creator`** 一般为 **`user`**，**TemplateType** 为 **`competitor`**、**`etf`**，见上文 **约定与枚举**）。

MCP 仅做代理：请求体 **`mcp_token`** → **`user_id`** 后 **`POST`** 转发 **Prompt 服务**；模板生成、ETF 基准模板、**`stock_db`** 补全等在 **Prompt 服务**。

### 接口一览

| URL | 方法 | 说明 |
|-----|------|------|
| **`/createCompetitorPromptTemplate`** | POST | 创建竞品分析模板 |
| **`/editCompetitorPromptTemplate`** | POST | 按 **`list`** 重写模板内容与 **`variable`** |
| **`/deleteCompetitorPromptTemplate`** | POST | 删除本人竞品模板 |
| **`/createEtfPromptTemplate`** | POST | 创建 ETF 分析模板（基于管理员基准模板生成用户副本） |
| **`/editEtfPromptTemplate`** | POST | 修改 ETF 模板 |
| **`/deleteEtfPromptTemplate`** | POST | 删除本人 ETF 模板 |

**认证**：Header **`Authorization: Bearer <API_KEY>`**（MCP API Key）。

### 请求体参数（MCP；均需 `mcp_token`）

### `list` 结构（竞品创建/修改与 ETF 创建/修改共用）

**`list`** 为 JSON **数组**，**每一项**均为对象 **`{ "name": "<字符串>", "code": "<字符串>" }`**，例如：

```json
[
  { "name": "腾讯控股", "code": "00700.HK" },
  { "name": "阿里巴巴", "code": "09988.HK" }
]
```

竞品场景：每一项表示一只**竞品**的名称与代码。ETF 场景：每一项表示一只 **ETF**（或成份）的名称与代码，结构与上相同。

**竞品**

| 接口 | 参数 |
|------|------|
| **创建** | **`list`**（必填）：见上文 **`list` 结构**。**`language`** 选填：**`cn`** / **`en`** / **`hk`**，默认 **`cn`**（用户传入 **`name`** 的主语言，用于与 **`stock_db`** 补全互补）。 |
| **修改** | **`id`**（必填）：模板 Mongo **`_id`** 字符串。**`list`**、**`language`** 同创建。 |
| **删除** | **`id`**（必填）。 |

**ETF**

| 接口 | 参数 |
|------|------|
| **创建** | **`list`**（必填）：与竞品相同，见上文 **`list` 结构**。**`language`** 选填，默认 **`cn`**。 |
| **修改** | **`id`**、**`list`**（结构同上）；**`language`** 选填。 |
| **删除** | **`id`**（必填）。 |

### 响应说明

- **成功**：HTTP 状态码与 Prompt 服务一致。**创建**接口常见：**`code`: 100**、**`prompt_id`**（新建文档 **`_id`**）、**`variable`** 等。**修改 / 删除**常见 **`code`: 100** 与 **`message`**（以实际返回为准）。
- **失败**：无效 **`mcp_token`**（MCP 层 **401** / **`code` 102**）；缺少必填字段（MCP 层 **400**）；模板不存在或无权限（上游 **404** 等）；Prompt 服务不可用（MCP **502**）。

### 请求示例

```bash
# 创建竞品分析模板
curl -X POST "http://localhost:3120/createCompetitorPromptTemplate" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <API_KEY>" \
  -d '{"mcp_token":"your_mcp_token","list":[{"name":"某公司","code":"00700.HK"},{"name":"竞品B","code":"09988.HK"}],"language":"cn"}'

# 修改竞品模板
curl -X POST "http://localhost:3120/editCompetitorPromptTemplate" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <API_KEY>" \
  -d '{"mcp_token":"your_mcp_token","id":"<prompt_object_id>","list":[{"name":"某公司","code":"00700.HK"}],"language":"cn"}'

# 删除竞品模板
curl -X POST "http://localhost:3120/deleteCompetitorPromptTemplate" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <API_KEY>" \
  -d '{"mcp_token":"your_mcp_token","id":"<prompt_object_id>"}'

# 创建 ETF 分析模板（list 与竞品同为 [{name,code}, ...]）
curl -X POST "http://localhost:3120/createEtfPromptTemplate" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <API_KEY>" \
  -d '{"mcp_token":"your_mcp_token","list":[{"name":"黄金ETF","code":"518880.SH"}],"language":"cn"}'

# 修改 / 删除 ETF：POST /editEtfPromptTemplate、/deleteEtfPromptTemplate，参数与竞品 edit/delete 相同（含 id、list 等）
```

创建成功后可将返回的 **`prompt_id`** 用于 **getMCPAnalysis** / 上游单项分析接口（与官方模板用法相同）。

---

## getMCPAnalysis（获取 MCP 分析结果）

根据 **`prompt_id`**、股票代码等选用对应单项分析模板，调用 LLM 并返回分析结果（不限于技术面模板）。

**MCP 层约定**：

- **`period`**：**必填**，须为 **约定与枚举 · period** 表中之一；非法值返回 **400**，响应体可含 **`allowed_period`**（排序后的允许列表）。
- **`language`**：**选填**；仅允许 **`cn`**、**`en`**、**`hk`**。不传、字段缺失或仅空白时，MCP 按 **`cn`** 转发；若显式传入非法值，返回 **400**，响应体可含 **`allowed_language`**。
- 转发至 AIServer 的请求体中 **`language`** 恒为上述三者之一（默认已解析为 **`cn`**）。

### 接口定义

| 项目 | 说明 |
|------|------|
| **路由** | `@app.route('/getMCPAnalysis', methods=['POST'])` |
| **URL** | `/getMCPAnalysis` |
| **方法** | `POST` |
| **Content-Type** | `application/json` |
| **认证** | `@require_api_key`，Header 中携带 `Authorization: Bearer <API_KEY>`（MCP API Key） |

### 请求体参数

**通过 MCP 调用时**（请求体传 `mcp_token`，MCP 解析为 `user_id` 后转发）：

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| **mcp_token** | string | 是 | 用户 MCP 令牌，MCP 层解析为 `user_id` 后转发至 AIServer。 |
| **name** | string | 是 | 股票名称，如 `腾讯控股`。 |
| **code** | string | 是 | 股票代码，如 `00700.HK`、`518880.SH`。 |
| **prompt_id** | string | 是 | 单项分析模板 ID（一般为 Mongo **`_id`**）。来源：**getSinglePromptTemplate** 列表项，或 **createCompetitorPromptTemplate** / **createEtfPromptTemplate** 等返回的 **`prompt_id`**。 |
| **language** | string | 否 | 仅允许 **`cn`**、**`en`**、**`hk`**（MCP 会转小写后转发）；不传或空字符串时默认 **`cn`**。 |
| **period** | string | 是 | 仅允许 **`no_period`**、**`minutes`**、**`hourly`**、**`daily`**、**`weekly`**、**`monthly`**、**`quarterly`**、**`yearly`**、**`longterm`**（与 **约定与枚举 · period** 一致）。 |

**AIServer 上游接口**（MCP 转发后的请求体）：

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| **user_id** | string | 是 | 用户 ID（由 MCP 根据 `mcp_token` 解析填入）。 |
| **name** | string | 是 | 股票名称。 |
| **code** | string | 是 | 股票代码。 |
| **prompt_id** | string | 是 | 同上。 |
| **language** | string | 是 | 恒为 **`cn`** / **`en`** / **`hk`**（MCP 将客户端未传或空白规范为 **`cn`**，否则 **`strip().lower()` 后校验**）。 |
| **period** | string | 是 | 同上枚举（已由 MCP **`strip()` 后校验**）。 |

### 响应说明

- **成功**：HTTP 200，返回 `code: 100`，以及 `data`（`create_date`、`analysis_result`、`model`）。
- **失败**：
  - 无效 **`mcp_token` 或用户不存在**（MCP 层）：**401**，`code: 102`，`message` 含「无效」或「不存在」等（与实现一致）。
  - 未找到用户（AIServer 等业务层）：`code: 102`，如 **`未找到用户`**。
  - 缺少或非法枚举（MCP 层）：**400**；缺 **`period`**，或 **`period`** / **`language`**（显式传入且非空但非法）不在允许列表时，响应体可能含 **`allowed_period`** / **`allowed_language`**。
  - AIServer 不可用：MCP 返回 **`code: 502`**。

成功时响应体结构（与 AIServer 一致）：

| 字段 | 类型 | 说明 |
|------|------|------|
| **code** | number | 状态码，100 表示成功。 |
| **data** | object | 分析结果数据，见下方。 |

**data 对象结构**：

| 字段 | 类型 | 说明 |
|------|------|------|
| **create_date** | string | 分析创建时间（服务端时间，如 `"Sun, 15 Mar 2026 19:00:30 GMT"`）。 |
| **analysis_result** | string | LLM 生成的分析报告正文，为 Markdown 格式（语种由请求中的 **`language`**：`cn` / `en` / `hk` 决定）。 |
| **model** | string | 实际使用的模型名称，如 `MiniMax-M2.5`。 |

**响应示例**：

```json
{
  "code": 100,
  "data": {
    "analysis_result": "## 腾讯控股（00700.HK）技术分析报告\n\n### 一、数据概述\n本次分析基于...\n\n### 二、技术指标分析\n...",
    "create_date": "Sun, 15 Mar 2026 19:00:30 GMT",
    "model": "MiniMax-M2.5"
  }
}
```

失败示例（用户不存在）：

```json
{
  "code": 102,
  "message": "未找到用户"
}
```

失败示例（MCP 层 **`period` 非法**）：

```json
{
  "code": 400,
  "message": "无效的 period",
  "allowed_period": [
    "daily", "hourly", "longterm", "minutes", "monthly",
    "no_period", "quarterly", "weekly", "yearly"
  ]
}
```

### 请求示例

```bash
# 通过 MCP 调用：language 省略时默认 cn；period 必填
curl -X POST "http://localhost:3120/getMCPAnalysis" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <API_KEY>" \
  -d '{"mcp_token": "your_mcp_token", "name": "腾讯控股", "code": "00700.HK", "prompt_id": "66494754fbe37cd6846ebd89", "period": "monthly"}'

# 显式指定 language（如英文）
curl -X POST "http://localhost:3120/getMCPAnalysis" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <API_KEY>" \
  -d '{"mcp_token": "your_mcp_token", "name": "腾讯控股", "code": "00700.HK", "prompt_id": "66494754fbe37cd6846ebd89", "period": "daily", "language": "en"}'
```

---

## getStockDailyReports（按股票+日期聚合盘前/盘中/盘后报告）

一次查询某只股票在某一天的三类报告：**盘前分析**、**盘中交易决策**、**盘后复盘**。  
该接口由 MCP 直接查询报告集合并聚合返回，适合 Agent 在单次调用中拿到完整日内报告上下文。

### 接口定义

| 项目 | 说明 |
|------|------|
| **URL** | `/getStockDailyReports` |
| **方法** | `POST` |
| **Content-Type** | `application/json` |
| **认证** | `Authorization: Bearer <API_KEY>`（MCP API Key） |

### 请求体参数

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| **mcp_token** | string | 是 | 用户 MCP 令牌，MCP 解析为 `user_id`。 |
| **code** | string | 是 | 股票代码，如 `00700.HK`、`AAPL.US`。 |
| **report_date** | string | 是 | 目标日期，格式 `YYYY-MM-DD`。 |

### 查询逻辑说明

- **盘前 / 盘中**：按 `code` 且 `created_at` 或 `updated_at` 落在 `report_date` 当天筛选。
- **盘后**：优先匹配 `session_date == report_date`，并兼容 `created_at` / `updated_at` 在当日的记录。
- 结果按 `updated_at` 倒序返回，每个阶段为一个列表（可能为空）。

### 响应说明

成功返回（HTTP `200`）：

```json
{
  "code": 100,
  "message": "success",
  "data": {
    "code": "00700.HK",
    "report_date": "2026-05-15",
    "pre_market": [
      {
        "phase": "pre_market",
        "report_id": "6825e65e2d7f2a005b6f8ca1",
        "code": "00700.HK",
        "stock_name": "腾讯控股",
        "result": "long",
        "confidence": "high",
        "reason": "判定依据",
        "suggestion": "buy",
        "summary": "盘前摘要",
        "report": "盘前完整报告",
        "created_at": "2026-05-15T09:02:10",
        "updated_at": "2026-05-15T09:05:22"
      }
    ],
    "intraday": [],
    "post_market": []
  }
}
```

### 失败场景

- 缺少 `mcp_token`：HTTP `400`，`code: 401`
- `mcp_token` 无效或用户不存在：HTTP `401`，`code: 102`
- 缺少 `code` / `report_date`，或 `report_date` 非 `YYYY-MM-DD`：HTTP `400`，`code: 400`
- 服务端异常：HTTP `500`，`code: 500`

### 请求示例

```bash
curl -X POST "http://localhost:3120/getStockDailyReports" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <API_KEY>" \
  -d '{"mcp_token":"your_mcp_token","code":"00700.HK","report_date":"2026-05-15"}'
```
