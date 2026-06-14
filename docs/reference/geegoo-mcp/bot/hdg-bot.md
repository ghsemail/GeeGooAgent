# MCP API：HDGBot 接口说明

## 概述

本文档描述通过 MCP（Skills）对 **HDG 对冲交易机器人**（`bot_type: HDG`）的**创建、修改、删除、列表与运行日志**接口。分类与命名见 [`common.md`](common.md)「机器人分类与命名」。调用方不传 `user_id`，改为传入 `mcp_token`，由服务端根据 `mcp_token` 解析出对应用户后再调用 Bot 服务对应逻辑（列表与日志在 MCP 进程内直读数据库）。

HDG 用于在独立标的上对冲已绑定的**主策略交易机器人**；**`binding` 可指向 DCA 信号交易机器人、GRID 网格交易机器人或 SmartTrade 交易机器人**（见 `binding`、`direction`，代码侧对应 **`DCA` / `GRID` / `SmartTrade`**）。主策略信息通过 MCP 获取：**`POST /getAllDCABots`**、**`POST /getAllGRIDBots`**、**`POST /getAllSmartTrades`**，从响应 **`data`** 中取得 **`bot_id`**、`botname`、`code` 等再填入 `binding`。

- **基础路径**：geegoo mcp 根地址（默认示例：`http://0.0.0.0:5700`）
- **认证方式**：请求头 `Authorization: Bearer <API_KEY>`；缺少或错误的 API Key 时 HTTP **401**（响应体为 `error` 字段说明，非下文 `code` 体系）。
- **缺少 `mcp_token` 或必填业务 ID**：未传 `mcp_token`，或更新/删除时未传 `bot_id`，HTTP 为 **400**，响应 JSON 中 **`code` 为 401**（`message` 提示缺少的字段）。这与 **无效 `mcp_token`**（找不到用户）时的 **`code` 102**、HTTP **401** 不同。

**公共约定**：认证与 **`mcp_token`**、MCP **`/searchCode`** 等共用说明，见 [`common.md`](common.md)。

---

## 创建 HDGBot

### 接口定义

| 项目 | 说明 |
|------|------|
| **URL** | `/createHDGBot` |
| **方法** | `POST` |
| **Content-Type** | `application/json` |
| **认证** | 需要在 Header 中携带 `Authorization: Bearer <API_KEY>` |

### 请求体参数

请求体为 JSON。下表列出创建时可传入的字段；未传入的项由 Bot 服务端按 HDG 默认参数补齐。

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| **mcp_token** | string | 是 | 用户 MCP 令牌，用于在服务端解析 `user_id`。 |
| **botname** | string | 是* | 机器人名称；全局 `botname` 不可重复。 |
| **stock_name** | string | 否 | 对冲标的名称。 |
| **code** | string | 否 | 对冲标的代码，如 `00700.HK`、`518880.SH`。 |
| **frequency** | string | 否 | K 线/检查频率（与主策略、调度一致即可）。 |
| **direction** | string | 否 | **对冲方向**：取 **`bullish`**（**同向对冲**）或 **`bearish`**（**反向对冲**）；与主策略槽位、成交联动的完整说明见下文 **direction 取值与联动逻辑**。 |
| **binding** | object | 否 | 绑定主机器人信息，见下文 **binding 结构说明**。创建时若服务端要求绑定，需与 `direction` 等一并满足校验。 |
| **order_size** | number | 否 | 单笔/头寸数量基数（整数）；未传时由服务端默认。 |
| **tp** | object | 否 | 止盈配置，字段与系数含义见下文 **tp / sl 结构与系数**。 |
| **sl** | object | 否 | 止损配置，字段与系数含义见下文 **tp / sl 结构与系数**。 |

#### tp / sl 结构与系数

HDG 在对冲标的上持仓后，由调度按持仓**成本价**计算止盈价、止损价；**`fix_tp`、`fix_sl`、`*_trailing_deviation` 均为百分比数值**（如 `5.0` 表示 5%），与 DCA/SmartTrade 的固定止盈止损含义一致。

**`tp`（止盈）**

| 字段 | 类型/含义 | 说明 |
|------|-----------|------|
| **fix_tp** | 百分比 | **固定止盈比例**（相对持仓**成本价**向上）。止盈参考价 **`tp_price = 成本价 × (1 + fix_tp ÷ 100)`**；例如 `5.0` 表示涨到「成本上 5%」触发止盈或进入止盈跟踪。 |
| **profit_trailing** | 布尔 | 为 `true` 时：现价**首次高于** `tp_price` 后，不立即全部卖出，而是进入**止盈跟踪**，记录阶段最高价。 |
| **profit_trailing_deviation** | 百分比 | **止盈跟踪回落阈值**：自阶段最高价下跌超过该**百分比**时触发卖出（例如 `1.0` 表示回落 1%）。仅在 **`profit_trailing` 为 `true`** 时参与判定。 |

**`sl`（止损）**

| 字段 | 类型/含义 | 说明 |
|------|-----------|------|
| **fix_sl** | 百分比 | **固定止损比例**（相对持仓**成本价**向下）。止损参考价 **`sl_price = 成本价 × (1 − fix_sl ÷ 100)`**；例如 `2.0` 表示跌破「成本下 2%」触发止损或进入止损跟踪。 |
| **stop_loss_trailing** | 布尔 | 为 `true` 时：现价**首次低于** `sl_price` 后，进入**止损跟踪**（记录阶段最高价，再按回落比例卖出）。为 `false` 时：跌破 `sl_price` 则直接按止损逻辑卖出。 |
| **stop_loss_trailing_deviation** | 百分比 | **止损跟踪回落阈值**：在止损跟踪阶段，自阶段最高价回落超过该**百分比**时卖出（与止盈跟踪的偏离量计算方式同类）。仅在 **`stop_loss_trailing` 为 `true`** 时参与判定。 |

\* 业务上应提供有效 `botname`；服务端对重名返回 `code: 101`。

**说明**：不要传 `type`（固定为 HDG）、`bot_id` / `_id`；`user_id` 由 MCP 根据 `mcp_token` 填入转发请求。

#### binding 结构说明

`binding` 为对象，用于指向已存在的 **DCABot**、**GRIDBot** 或 **SmartTrade** 主机器人：

```json
{
  "binding_bot_id": "",
  "binding_bot_name": "",
  "binding_bot_type": "DCA",
  "binding_code": "",
  "binding_stock_name": ""
}
```

- **binding_bot_type**：主策略类型，取值为 **`DCA`**、**`GRID`** 或 **`SmartTrade`**（须与真实主机器人一致）。
- **binding_bot_id**：主策略在各自列表接口中的 **`bot_id`**（即该主策略参数库文档 `_id` 的字符串形式），见下表。
- **binding_bot_name** / **binding_code** / **binding_stock_name**：建议与列表接口返回的 **`botname`**、**`code`**、**`stock_name`** 保持一致，便于展示与排查。

同一主机器人、同一 **direction** 下重复绑定会触发 Bot 层校验错误（消息大意：该方向已绑定对冲机器人）。

#### `direction` 取值与联动逻辑

**对冲方向（语义）**：`direction` 表示对冲方向——**`bearish`** 为**反向对冲**（对冲腿与主仓典型风险暴露方向相反，用于对冲不利波动等）；**`bullish`** 为**同向对冲**（对冲腿与主策略在该侧槽位上的联动/暴露为同向）。以下表格与联动说明在此基础上描述与主策略文档槽位及 DCA 止盈/止损触发的对应关系。

| 取值 | 含义（与主策略文档的槽位） |
|------|---------------------------|
| **`bullish`** | 对应主策略 **`hedging.bullish`**（看涨侧对冲槽位）。 |
| **`bearish`** | 对应主策略 **`hedging.bearish`**（看跌侧对冲槽位）。 |

须使用**小写**英文（`bullish` / `bearish`），与服务端校验及运行逻辑一致。

**创建时的登记**：当 **`binding.binding_bot_type`** 为 **`DCA`**、**`GRID`** 或 **`SmartTrade`** 时，创建 HDG 会在主策略文档上打开对应槽位的 **`hedging.<bullish|bearish>.switch`**，并写入对冲标的代码/名称；删除 HDG 时会复位该槽位。**`direction`** 参与「同主同向」重复绑定校验（同一 `binding_bot_id` + `direction` 不可重复）。

**与主策略成交事件的联动**

- **主类型为 DCA**  
  - **`bullish`**：主 DCA 发生**止盈类卖出**（成交原因含 `profit_take_sell`、`profit_take_tailing_sell`）并进入对冲逻辑时，若绑定的 HDG **`direction == 'bullish'`**，则在该 HDG 的对冲标的上执行对冲下单。  
  - **`bearish`**：主 DCA 发生**止损类卖出**（`stop_loss_sell`、`stop_loss_tailing_sell`）时，若绑定的 HDG **`direction == 'bearish'`**，则触发对冲下单。  
  即：**bullish 对齐「止盈平仓」侧对冲，bearish 对齐「止损平仓」侧对冲**（适用于用反向 ETF/杠杆品种对冲主仓风险暴露的典型用法）。

- **主类型为 GRID**  
  网格策略在部分卖出或突破网格等场景下会单独触发对冲下单，**与 DCA 的止盈/止损分流不是同一套条件**；是否与 `hedging.bullish` 等开关、网格位置有关，以实际运行为准。若同一主机器人仅配置一个 HDG，行为最清晰；若多 HDG 并存，以运行时绑定关系为准。

- **主类型为 SmartTrade**  
  在**止盈类 / 止损类卖出**成交结算时，与 DCA 相同：根据本次平仓原因（如止盈或止损）与 HDG 的 **`direction`** 决定是否在对冲侧下单；网格类独立触发逻辑不适用 SmartTrade。

**唯一性**：同一 **`binding.binding_bot_id`** 下，**`direction`** 不能重复（已存在同主同向 HDG 则创建失败）。

##### 如何查询主机器人并填写 binding

创建 HDG 前，应用 **`mcp_token`** 调用下表 **POST** 接口，从返回的 **`data`** 中取对应主机器人的 **`bot_id`**、名称与代码，填入 `binding`。**不要猜测 `bot_id`**，应以查询结果为准。

| 主机器人类型 | MCP 列表接口（均为 `POST`，请求体需 `mcp_token`） |
|--------------|---------------------------------------------------|
| DCABot | `/getAllDCABots` |
| GRIDBot | `/getAllGRIDBots` |
| SmartTrade | `/getAllSmartTrades` |

列表项中均可选用 **`bot_id`** 作为 **`binding_bot_id`**；**`binding_bot_type`** 分别填 **`DCA`**、**`GRID`**、**`SmartTrade`**。可选在请求体中传 **`code`**（与主策略标的代码一致），以缩小结果集。

**与主策略 `hedging` 回写**：创建/删除 HDG 时，对绑定类型为 **DCA**、**GRID**、**SmartTrade** 的主机器人，Bot 服务会同步更新其文档中的 **`hedging`** 字段（与 DCA/GRID 同一套槽位语义）。

### 响应说明

- **成功**：`code === 100`，`message` 含「创建HDG bot成功」。
- **业务错误**：`code` 为 101（名称冲突、绑定冲突等）、102、103、105 等时 HTTP 一般为 **400**。
- **mcp_token 无效**：`code: 102`，HTTP **401**。
- **调用 Bot 服务失败**：HTTP **502**。

### 请求示例

```bash
curl -X POST "http://localhost:5700/createHDGBot" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <API_KEY>" \
  -d '{
    "mcp_token": "mcp_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
    "botname": "腾讯-HDG-对冲",
    "stock_name": "南方恒生科技",
    "code": "03033.HK",
    "direction": "bearish",
    "binding": {
      "binding_bot_id": "67175fa4d6f9d6cf92bc5dd8",
      "binding_bot_name": "腾讯-DCA",
      "binding_bot_type": "DCA",
      "binding_code": "00700.HK",
      "binding_stock_name": "腾讯控股"
    },
    "order_size": 100,
    "frequency": "60m"
  }'
```

### 与 Bot 服务的关系

MCP 将请求转发至 Bot 服务的 `POST /createBot`，请求体中带 `bot_type: "HDG"` 及解析得到的 `user_id`。

---

## 修改 HDGBot

### 接口定义

| 项目 | 说明 |
|------|------|
| **URL** | `/updateHDGBot` |
| **方法** | `POST` |
| **Content-Type** | `application/json` |
| **认证** | `Authorization: Bearer <API_KEY>` |

### 请求体参数

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| **mcp_token** | string | 是 | 用户 MCP 令牌。 |
| **bot_id** | string | 是 | 要修改的 HDG 的策略主文档 ID（与列表返回的 **`bot_id`** 相同）。 |
| **botname** | string | 否 | 机器人名称。 |
| **stock_name** | string | 否 | 标的名称。 |
| **code** | string | 否 | 标的代码。 |
| **frequency** | string | 否 | 检查频率。 |
| **order_size** | number | 否 | 头寸数量基数。 |
| **tp** | object | 否 | 同创建接口，字段含义见上文 **tp / sl 结构与系数**。 |
| **sl** | object | 否 | 同创建接口，字段含义见上文 **tp / sl 结构与系数**。 |

**说明**：修改 HDG 时**不能**通过本接口变更 `binding`、`direction`；仅传需要修改的其它字段即可。

### 响应说明

- **成功**：`code === 100`，「更新HDG Bot配置成功」。
- **业务错误**：`code` 为 101、102、103 等，HTTP **400**。
- **mcp_token 无效**：`code: 102`，HTTP **401**。

### 与 Bot 服务的关系

转发至 `POST /editBot`，`bot_type: "HDG"`。

---

## 删除 HDGBot

### 接口定义

| 项目 | 说明 |
|------|------|
| **URL** | `/deleteHDGBot` |
| **方法** | `POST` |
| **Content-Type** | `application/json` |
| **认证** | `Authorization: Bearer <API_KEY>` |

### 请求体参数

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| **mcp_token** | string | 是 | 用户 MCP 令牌。 |
| **bot_id** | string | 是 | 要删除的 HDG Bot ID。 |

### 响应说明

- **成功**：`code === 100`，`message` 随持仓状态可能为「删除bot成功」「已撤单，删除bot成功」等。
- **业务错误**：`code` 为 102、103、104 等，HTTP **400**。

### 与 Bot 服务的关系

转发至 `POST /deleteBot`，`bot_type: "HDG"`；必要时撤单，并清理该 HDG 的运行信息、日志与盈亏记录，以及主策略上的对冲标记。

---

## 获取所有 HDGBot

### 接口定义

| 项目 | 说明 |
|------|------|
| **URL** | `/getAllHDGBots` |
| **方法** | `POST` |
| **Content-Type** | `application/json` |
| **认证** | `Authorization: Bearer <API_KEY>` |

### 请求体参数

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| **mcp_token** | string | 是 | 用户 MCP 令牌。 |
| **code** | string | 否 | 按对冲标的代码筛选。 |
| **binding_bot_id** | string | 否 | 按绑定主策略 `bot_id` 筛选。 |

### 响应说明

- **成功**：`code === 100`，`data` 为 HDG 列表。每条记录在策略参数文档上合并运行信息（含开关等）；字段含义与「通过 user_id 拉取用户 HDG 列表」的 Bot 接口返回一致（本 MCP 接口在进程内直读库组装，不依赖该 HTTP 调用）。
- **mcp_token 无效**：`code: 102`，HTTP **401**。

#### 单条数据结构说明

| 字段 | 含义 |
|------|------|
| **bot_id** | 对冲策略**参数主文档** ID；创建 / 修改 / 删除时请求体中的 **`bot_id`**。 |
| **hdg_bot_id** | **运行信息**行 ID（与参数主文档配套的状态行主键；与其它机器人类型列表里「运行信息 id」类字段作用相同）。 |
| **bot_switch** | 运行开关，接口中为字符串 **`"True"`** / **`"False"`**。 |
| **type** | 固定 **`"HDG"`**。 |
| **user_id** | 所属用户 ID（字符串）。 |
| **botname** / **code** / **stock_name** | 对冲侧展示名与标的（例如反向/杠杆 ETF）。 |
| **binding** | 绑定的主机器人：`binding_bot_id`、`binding_bot_type`（如 **`GRID`**）、`binding_code`、`binding_stock_name` 等。 |
| **direction** | **`bullish`**（同向对冲）或 **`bearish`**（反向对冲）；含义与触发逻辑见上文 **direction 取值与联动逻辑**。 |
| **frequency** | K 线 / 检查频率。 |
| **order_size** | 数值型头寸基数。 |
| **tp** / **sl** | 止盈、止损配置；各字段与系数含义见上文 **tp / sl 结构与系数**。 |

响应示例（与线上一致）：

```json
{
  "code": 100,
  "message": "success",
  "data": [
    {
      "binding": {
        "binding_bot_id": "6781cb8309a2189f26d8866e",
        "binding_bot_name": "腾讯控股机器人",
        "binding_bot_type": "GRID",
        "binding_code": "00700.HK",
        "binding_stock_name": "腾讯控股"
      },
      "bot_id": "6781cc64acff90570c6a73b6",
      "bot_switch": "True",
      "botname": "腾讯控股反向对冲机器人",
      "code": "07552.HK",
      "hdg_bot_id": "6781cc64acff90570c6a73b7",
      "direction": "bearish",
      "frequency": "3m",
      "order_size": 100,
      "sl": {
        "fix_sl": 2.0,
        "stop_loss_trailing": false,
        "stop_loss_trailing_deviation": 1.0
      },
      "stock_name": "南方两倍做空恒科",
      "tp": {
        "fix_tp": 5.0,
        "profit_trailing": false,
        "profit_trailing_deviation": 1.0
      },
      "type": "HDG",
      "user_id": "6366170502d5c175fd586fe8"
    }
  ]
}
```

### 与 Bot 服务的关系

`getAllHDGBots` 在 MCP 进程内直读持久化中的策略参数与运行信息并合并，**不**再转发其它 HTTP；返回字段与上述 Bot 侧列表语义对齐（运行信息 id 为 **`hdg_bot_id`**）。

---

## 获取 HDGBot 运行日志

与 Bot 服务 **`POST /getHDGBotLog`** 语义一致：返回**交易运行快照**（**`log`**）及**当前持仓情况**（**`info`**，来自 **`hdg_info.position`**）。MCP 校验 **`bot_id`** 属于当前用户后直读 **`hdg_log`**、**`hdg_info`**。

### 接口定义

| 项目 | 说明 |
|------|------|
| **URL** | `/getHDGBotLog` |
| **方法** | `POST` |
| **Content-Type** | `application/json` |
| **认证** | `Authorization: Bearer <API_KEY>` |

### 请求体参数

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| **mcp_token** | string | 是 | 用户 MCP 令牌。 |
| **bot_id** | string | 是 | HDG 机器人 ID（与 **`getAllHDGBots`** 返回的 **`bot_id`** 一致）。 |

### 响应说明

**成功（HTTP 200）**：`code === 100`，`message` 一般为 `"success"`，业务载荷在 **`data`** 中（与 Bot 服务返回的 **`info` + `log`** 一致，外加 MCP 外层 **`code` / `message`**）。

#### `data` 整体结构

| 字段 | 类型 | 说明 |
|------|------|------|
| **info** | object | **当前持仓情况**：来自 **`hdg_info.position`** 的汇总（股数、成本价、浮动盈亏等）；无运行信息或字段缺失时各数值可能为 **`0`**，或整体接近空对象。 |
| **log** | array | 运行快照列表，**按 `time` 从新到旧**，**最多 100 条**。仅包含有止盈/止损价、跟踪状态或非空订单状态等信息的记录（与库查询条件一致）。 |

#### `data.info`（当前持仓情况）

| 字段 | 类型 | 说明 |
|------|------|------|
| **qty** | number | **持仓头寸**（股数）。 |
| **price** | number | 持仓成本价或与展示一致的价格。 |
| **pl_val** | number | 浮动盈亏金额。 |
| **pl_ratio** | number | 浮动盈亏比例（百分比数值）。 |

#### `data.log[]`（单条日志）

每条对应 **`hdg_log`** 中满足查询条件的一条文档，字段由该条 **`log`** 子文档与 **`time`** 组装。

| 字段 | 类型 | 说明 |
|------|------|------|
| **time** | string | 该条日志时间；MCP 可能对 `datetime` 序列化为 **`YYYY-MM-DD HH:MM:SS`** 或 ISO 8601，以实际返回为准。 |
| **next_opt** | string | 本周期策略给出的下一步意图，未写入时默认为 **`hold`**。 |
| **trailing** | object | 跟踪止盈/止损等动态状态（未启用时可能为空对象 **`{}`**）。 |
| **tp_sl** | object | 止盈/止损相关价格与状态（与创建时 **`tp` / `sl`** 配置对应的运行时快照）。 |
| **position** | object | 该快照下的**持仓概要**：见下表。 |

**`data.log[].position` 常见子字段**：

| 子字段 | 类型 | 说明 |
|--------|------|------|
| **qty** | number | **持仓头寸**（股数）。 |
| **price** | number | **成本价**（或与该快照一致的持仓均价）。 |
| **pl_val** / **pl_ratio** | number | 该快照下的浮动盈亏金额 / 比例（若策略写入）。 |
| **opt** | string | **仓位操作**侧状态或意图（如 **`hold`** 等）。 |
| **order_id** | string | 关联 **订单 ID**；无订单时可为空字符串。 |
| **order_status** | string | **订单状态**；无在途订单时可为空字符串。 |
| **order_time** | string | **订单时间**（若有）。 |
| **can_sell_qty** | number | **当前可卖头寸**（可卖股数）。 |

**说明**：**`info`** 为**当前**持仓汇总；**`log`** 为**历史快照**，用于复盘止盈止损与订单侧状态。

**错误响应**：缺少 **`mcp_token`** 或 **`bot_id`** → **`401`**；无效 **`bot_id`** → **`101`**；无权限 → **`103`**；无效 token → **`102`**（HTTP 含义与本文档其它 MCP 接口一致）。

### 请求示例

```bash
curl -X POST "http://localhost:5700/getHDGBotLog" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <API_KEY>" \
  -d '{"mcp_token": "mcp_xxx", "bot_id": "<BOT_ID>"}'
```

### 与 Bot 服务的关系

- Bot 侧：**`POST /getHDGBotLog`**（请求体为 **`bot_id`**），不经 **`mcp_token`** 校验归属。
- MCP 侧：归属校验后直读 **`hdg_log`** / **`hdg_info`**；成功响应统一为 **`{ code, message, data }`**。

---

## 行为摘要

| 操作 | MCP 路径 | 是否转发 Bot HTTP |
|------|-----------|-------------------|
| 创建 | `POST /createHDGBot` | 是 → `/createBot` |
| 修改 | `POST /updateHDGBot` | 是 → `/editBot` |
| 删除 | `POST /deleteHDGBot` | 是 → `/deleteBot` |
| 列表 | `POST /getAllHDGBots` | 否（直读库） |
| 运行日志 | `POST /getHDGBotLog` | 否（直读库） |

以上路径均需 **`Authorization: Bearer <API_KEY>`**，且请求体带 **`mcp_token`**（列表与日志接口同样为 **POST**）。
