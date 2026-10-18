# GeeGooAgent Lab 模块

## 定位

`internal/lab` 承载 **实验能力**，与现网 `diagnose` / `strategydev` / worker 路径 **完全隔离**：

- 不修改现网信号诊断行为；现网 diagnose **不暴露** 实验 PA。
- 现网 worker **不执行** lab 任务。
- Lab 取数 **仅** 调用 GeeGooSignal **lab-api**（默认 `:3250`），不经过 `probe_bot_signal_series`、不调用 `/getSupportingPrice` 或 probe `key_levels`。

首个实验：**多周期价格分析（price-analysis）**，自归档 `archive/price-analysis-2026-10-17`（`6d3cc3e`）迁入，算法包括摆动 HH/HL/LH/LL、BOS/CHOCH、区间位置、Kaufman ER（无硬阈值）、RVOL 量能确认、多周期综合分（周线 0.45 + 日线 0.35 + 60m 0.20，约 −100～+100）。**默认只展示，不硬拦截**；对外术语统一为 **「分析」**，不叫「打分」。

## 取数窗 vs 分析窗

| 概念 | 说明 |
| --- | --- |
| 取数窗 | 经 lab-api 一次拉长，覆盖最早信号锚点；同批 K 可复用 |
| 分析窗 | 按每条信号锚点截断后取尾：5m≈330、60m≈130、日线≈60、周线≈26；只向上取周期；分析不超过半年；预热 K 不计入分析 |

## Agent 开关与配置

| 项 | 环境变量 | 配置 JSON | 默认 |
| --- | --- | --- | --- |
| Lab HTTP 路由 | `GEEGOO_AGENT_LAB_ENABLED` | `lab_enabled` | **false** |
| Signal lab-api URL | `GEEGOO_AGENT_LAB_API_URL` | `lab_api_url` | `http://127.0.0.1:3250` |
| lab-api Bearer | `GEEGOO_SIGNAL_LAB_API_KEY` | `lab_api_key` | 回落 `signal_api_key` / MCP key |

路由（需 Lab 开关 + MCP Token，与 chat 一致）：

- `GET /v1/lab/experiments` — 实验列表
- `POST /v1/lab/experiments/price-analysis` — 运行 PA 实验

请求体示例：

```json
{
  "code": "00700.HK",
  "signal_frequency": "5m",
  "months": 3,
  "anchor_time": "2026-06-23 16:00:00",
  "spot_price": 0
}
```

响应含 `experiment: "price-analysis"` 与 `analysis`（多周期表 + `price_score` / `tf_scores` / `score_reason`）。

## 与 GeeGooSignal lab-api 的契约（Agent 侧定义，Signal 并行实现对齐）

Agent 通过 `internal/lab/signalclient` 访问 lab-api。**任务状态查询走 lab-api**，Agent 不直连 worker。

### `POST /v1/lab/kline/jobs`

创建 K 线取数任务（取数窗）。

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `code` | string | 是 | 标的代码 |
| `frequency` | string | 是 | `5m` / `60m` / `daily` / `weekly` 等 |
| `months_back` | int | 是 | 日历回溯月数（Agent 侧已按 TF 封顶） |
| `as_of` | string | 否 | 上海时区 `2006-01-02 15:04:05`，窗口结束锚点 |

响应：

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `job_id` | string | 任务 ID |

### `GET /v1/lab/jobs/{job_id}`

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `id` | string | 任务 ID |
| `status` | string | `pending` \| `running` \| `completed` \| `failed` |
| `error` | string | 失败原因 |
| `result` | object | `status=completed` 时存在 |
| `result.bars` | array | OHLCV bar 列表（与现网 probe `bars` 元素形状兼容：`time`/`open`/`high`/`low`/`close`/`volume`） |

鉴权：`Authorization: Bearer <GEEGOO_SIGNAL_LAB_API_KEY>`。

单测使用 `signalclient.MockClient`，**不依赖** Signal 已上线。

## 模块结构

```
internal/lab/
  service.go              # 实验编排入口
  bars_fetcher.go         # lab-api → priceanalysis.BarsFetcher
  priceanalysis/          # 自归档迁入的 PA 分析（纯 Agent 内计算）
  signalclient/           # lab-api HTTP Client + Mock
internal/runtimeapi/lab.go
internal/config/lab.go
```

## 如何新增实验

1. 在 `internal/lab` 增加子包或 `experiments/<name>`，**不要** import / 修改 `internal/workflow/strategydev` 诊断链。
2. 若需行情/K 线，扩展 `signalclient` 契约（或新增 lab-api 路径），并在 `docs/lab.md` 记录字段。
3. 在 `lab.Service` 增加方法，于 `runtimeapi/lab.go` 注册 `POST /v1/lab/experiments/<name>`。
4. 在 `GET /v1/lab/experiments` 列表中登记元数据。
5. 为 client 与 HTTP handler 编写 mock 单测。

## 与 Signal 侧待对齐项

- lab-api 路径前缀是否为 `/v1/lab/...`（含 `kline/jobs` 与 `jobs/{id}`）。
- `status` 枚举与 Agent 约定是否一致。
- `as_of` 语义是否与归档 PA 一致（截断为 `<= anchor` 的 bar，不含 anchor 之后）。
- 失败时 HTTP 状态码（4xx/5xx）与 JSON `error` 字段格式。
- 是否提供 batch 取数接口；当前 Agent 按 TF 多次 `CreateKlineJob`。
- lab-api 与 `cmd/lab-worker` 的部署端口、TLS、限流策略。
