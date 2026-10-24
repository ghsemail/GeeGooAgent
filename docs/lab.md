# GeeGooAgent Lab 模块

## 定位

`internal/lab` 承载 **实验能力**，与现网 `diagnose` / `strategydev` / worker 路径 **完全隔离**：

- 不修改现网信号诊断行为；现网 diagnose **不暴露** 实验 PA。
- 现网 worker **不执行** lab 任务（Signal 侧为独立 **lab-api + lab-worker**）。
- Lab 取数 **仅** 调用 GeeGooSignal **lab-api**（`:3250`），不经过 `probe_bot_signal_series`、不调用 `/getSupportingPrice` 或 probe `key_levels`。

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

## GeeGooSignal lab-api 契约（定稿）

- **Base URL**：`:3250`（如 `http://<signal-host>:3250`）
- **鉴权**：`Authorization: Bearer <GEEGOO_SIGNAL_LAB_API_KEY>`
- **路径前缀**：`/v1/lab/...`
- **v1 无 batch**：Agent 按周期 **多次** `POST /v1/lab/kline/jobs`（每个 TF 一次 CreateKlineJob），不调用 batch API。
- **任务状态**：仅通过 lab-api 轮询 `GET /v1/lab/jobs/{job_id}`；Agent **不直连** lab-worker。
- **Signal 并发**：lab-worker 同时 **running ≤ 4**，多余任务 **pending 排队**；Agent 客户端保持轮询即可，无需额外限流逻辑。

Agent 通过 `internal/lab/signalclient` 实现上述契约。单测使用 `MockClient`，**不依赖** Signal 已上线。

### `GET /health`（可选）

- 响应 **HTTP 200** 表示 lab-api 可达。
- Agent **可选**调用（如运维探测）；实验取数 **不强依赖** 健康检查。

### `POST /v1/lab/kline/jobs`

创建 K 线取数任务（取数窗）。

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `code` | string | 是 | 标的代码 |
| `frequency` | string | 是 | `5m` / `60m` / `daily` / `weekly` 等 |
| `months_back` | int | 是 | 日历回溯月数（Agent 按 **周期** 计算：`信号锚点跨度 + 分析窗月数 + 预热`，并 respect 单任务 5000 根上限） |
| `as_of` | string | 否 | **Asia/Shanghai**，格式 **`2006-01-02 15:04:05`**；lab-api 只返回 **`time <= as_of`** 的 bar（与现网 probe 一致）；**不传**则取到最新 |

成功响应：

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `job_id` | string | 任务 ID |

### `GET /v1/lab/jobs/{job_id}`

对已创建任务的轮询：**HTTP 200** + JSON 任务体（含 `pending` / `running` / `failed` / `completed`）。

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `id` | string | 任务 ID |
| `status` | string | **`pending`** \| **`running`** \| **`completed`** \| **`failed`**（仅此四值） |
| `error` | string | `status=failed` 时的失败原因 |
| `result` | object | `status=completed` 时 **必须** 存在 |
| `result.bars` | array | `status=completed` 时 **必须** 存在（可为空数组）；元素与现网 probe `bars` 兼容：`time`/`open`/`high`/`low`/`close`/`volume` |

### 失败语义

| 场景 | HTTP |  body |
| --- | --- | --- |
| 非法请求、鉴权失败、job 不存在等 | **4xx** | `{"error":"<message>"}` |
| lab-api 内部错误（创建 job 前） | **5xx** | 通常含 `{"error":...}` |
| job 已创建后 worker 失败 | 轮询 GET 仍 **200** | `status=failed`，`error` 字段说明原因 |
| job 成功 | 轮询 GET **200** | `status=completed`，**必有** `result.bars` |

Agent `signalclient` 对 4xx/5xx 解析 JSON `error` 字段并包装为 `APIError`；对 `completed` 但缺少 `result` / `result.bars` 视为协议错误。

### Agent 侧 `months_back` 算法（`MonthsBackForLabFetch`）

`months_back(tf) = CapMonthsBack(信号 months) + analysisWindowMonths(tf) + warmupMonths(tf)`，再按 `EstimateFetchBars` 校验 Signal **5000 根/任务** 上限。

| 周期 | 分析窗（tail 根数） | 分析窗折算月数 | 预热 | **示例：信号 months=3** |
| --- | ---: | ---: | ---: | ---: |
| 5m | 330 | 1 | 1 | **4**（5 会超 5000 根，压到 4） |
| 60m | 130 | 1 | 1 | **5** |
| 日线 | 60 | 3 | 1 | **7** |
| 周线 | 26（min 20） | 6 | 1 | **10**（约 43 根，不再缺周期） |

结构分析仍只对锚点截断后的 **tail 分析窗** 做 `AnalyzePriceFilter`，长取数窗仅用于保证最早锚点处根数充足。

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
2. 若需行情/K 线，在 **定稿契约** 下扩展 lab-api（需与 Signal 联调），并更新 `docs/lab.md`。
3. 在 `lab.Service` 增加方法，于 `runtimeapi/lab.go` 注册 `POST /v1/lab/experiments/<name>`。
4. 在 `GET /v1/lab/experiments` 列表中登记元数据。
5. 为 client 与 HTTP handler 编写 mock 单测。

## Signal 诊断 Lab 模式：lazy lab-detail 持久化

Lab 诊断主流程（`workflow_completed`）仍只下发卡片摘要 `lab_episode_summaries`（含 `detail_ref`）。完整 **价格分析 + System One JEV** 仅在 `POST /v1/signal-diagnose/lab-detail` 按需（或 FE prefetch）计算。

### `detail_ref` 契约（FE / 测试落盘必读）

| 字段 | 含义 |
| --- | --- |
| 格式 | `{side}:{bar_index}`，例如 `sell:190`、`buy:7` |
| `bar_index` | **图表 K 线索引**，与 `signal_eval.*_details[].start_idx` 相同 |
| **不是** | `buy_details` / `sell_details` 数组下标（episode 列表序号） |

示例：`sell_details[7]`（第 8 条卖 episode）若 `start_idx=190`，则 Agent 下发的 `detail_ref` 为 **`sell:190`**，**不是** `sell:7`。`lab_episode_summaries` 每条现含 `start_idx` 与 `episode_index`（列表序号），FE 打开详情时必须使用卡片/摘要里的 `detail_ref` 或 `start_idx` 拼 ref，禁止用 UI 行号直接拼 `sell:{row}`。

`POST /v1/signal-diagnose/lab-detail` 响应 `detail.bar_index` 与 ref 后缀一致。错误 ref（越界）返回 400，文案会提示 start_idx 语义。

**常见误用 `sell:7`（episode 列表序号 7）**：只要图表 `len(bars) > 7`，Agent 仍返回 **HTTP 200** 和完整 `detail`，但锚在 **bar 7**（与 `sell:190` 不同时间/形态/PA），**不是空响应**。弹窗若仍显示「拉数失败」，优先查 FE 是否用错 ref、是否把 `ref_hint` / `price_analysis.rows[].structure` / `align_note` 统一映射成同一文案。此时响应可能带可选字段 **`ref_hint`**（当 ref 不在 `lab_episode_summaries` 且疑似 episode 序号误用时）。

### 实验室弹窗：Agent 可能带「失败」文案的 JSON 路径

| 路径 | 典型值 | 含义 |
| --- | --- | --- |
| `detail.kline_pattern.mtf.timeframes[].align_note` / `detail.kline_pattern_mtf…` | `日线 拉数失败（仅展示信号周期）` | MTF probe 真失败 |
| 同上 | `日线 大周期未加载（请打开 lab 详情）` | bundle 缺序列（非「拉数失败」） |
| 同上 | `4h 周期暂不支持（已跳过）` | unsupported |
| `detail.price_analysis.rows[].structure` | `拉数失败` | lab-api 取 bars 失败 |
| 同上 | `取数超时或已取消` | `DefaultLabDetailPhaseTimeout`（8s）内 MTF+PA 共享 phase ctx 被取消 |
| `detail.degraded[]` | `kline_mtf_probe_unavailable` / `price_anchor_missing` | 降级标记，非 structure 文案 |

lab-detail **不**走卡片 light enrich；MTF 来自 run 级 probe + `BuildKlinePatternMTFContext`。生产 flow 在正确 ref + 成功取数时不应出现上表「拉数失败」行（与 fe7574e 整机日志一致）。

### 行为

| 场景 | Agent 行为 |
| --- | --- |
| 首次 `lab-detail`（某 `run_id` + `detail_ref`） | 执行 `BuildLabEpisodeDetail`（lab-api PA + JEV），成功后写入会话 metadata 缓存 |
| 同一会话再次请求相同 `run_id` + `detail_ref` | **直接返回缓存**，不重新调用 lab-api / System One |
| 无效 `detail_ref` | 400，与原先一致；**不写入缓存** |
| TTL | 与 `lab_diagnose_snapshots` 相同：**48h**，最多保留 **12** 个 `run_id` 条目 |

响应体在原有字段上增加 `cached`（bool）：命中缓存为 `true`，新算为 `false`。

### Chat metadata（Agent 侧）

键名：`lab_diagnose_episode_details`（`MetaKeyLabDiagnoseEpisodeDetails`）

```json
{
  "flow-20261024T120000": {
    "expires_at": "2026-10-26T12:00:00Z",
    "details": {
      "buy:3": { "...": "LabEpisodeDetail，与 lab-detail 响应 detail 同形" },
      "sell:1": { "...": "..." },
      "flow:0": { "...": "..." }
    }
  }
}
```

`details` 内每个 value 的 JSON 字段与 **`POST /v1/signal-diagnose/lab-detail` 响应里的 `detail`** 完全一致（含 `price_analysis`、`price_analysis_jev`、`jev_unified`、顶层 `kline_pattern_*`、`kline_pattern` 嵌套对象、`degraded` 等）。FE 可将某次 diagnose 的整段 `details`（或按 ref 合并）复制进 Signal 测试记录 Mongo 的 `run` 对象，历史打开时本地渲染，**无需再调 lab-detail**。

### `jev_unified`（综合 JEV，仅展示）

在 lab-detail 内 **PA-JEV（`price_analysis_jev`）与本地 K 线形态汇总** 都算完后，由 Agent 做后处理写入 `jev_unified`。**不**改 Typesafe / JevClient 上游；**不**参与买卖门控或信号拦截。

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `direction` | `"bull"` \| `"bear"` \| `"unclear"` | 综合方向 |
| `confidence` | number | 0～1 |
| `conflict` | bool | 结构与形态是否冲突 |
| `breakdown.structure` | string | PA-JEV 摘要（通常来自 `recommended_reason`） |
| `breakdown.pattern` | string | 本地形态摘要（`kline_pattern_summary` 或 bias 行） |

合成规则（硬性，signed 加权）：

```
signed(dir, c): bull → +c, bear → −c, unclear → 0
score = 0.7 × signed(结构) + 0.3 × signed(形态)
score > 0 → bull；score < 0 → bear；score = 0（浮点比较用 ε≈1e-12 防噪，**无** 0.15 等产品阈）
confidence = |score|（始终）
```

| `conflict` | 条件 |
| --- | --- |
| `false` | 缺形态（仅结构 leg 参与 score）；或结构+形态 **方向相同**（含均为 `unclear`） |
| `true` | 结构+形态均 present 且 **方向不一致**（含一侧 `unclear`、另一侧 bull/bear）；或 **缺结构**（JEV 不可用 → `direction=unclear`, `confidence=0`） |

- **结构置信**：PA-JEV 推荐项在 `choices` 中的 `probability`（0～1）。
- **形态置信**：信号 bar 上形态 hits 的看多/看空条数占比（多数侧 / 总方向条数）；无 hits 时用 `|bias.score|` 粗估并截断到 1。
- **形态缺失**：无 summary、bias、hits → `score = 0.7 × signed(结构)`，`conflict=false`。
- **结构缺失**：JEV 未 OK → 不参与 score，`unclear` + `conflict=true`。
- **与 PR#83 差异**：已移除 \|score\| < 0.15 → `unclear` 的硬阈；\|score\|=0.14 等弱信号仍输出 bull/bear。仅 **score 精确对消为 0**（含结构/形态反向同强度）时方向为 `unclear`。
- **历史（PR#82）**：曾在一版中对「反向」一律输出 `unclear`（已由 PR#83 signed 加权取代）。

示例（同向）：

```json
"jev_unified": {
  "direction": "bull",
  "confidence": 0.71,
  "conflict": false,
  "breakdown": {
    "structure": "多周期结构倾向看多（置信 70%）",
    "pattern": "锤子(看多)"
  }
}
```

示例（反向但未对消：结构看空 0.9、形态看多 0.3 → score≈−0.54）：

```json
"jev_unified": {
  "direction": "bear",
  "confidence": 0.54,
  "conflict": true,
  "breakdown": {
    "structure": "多周期结构倾向看空（置信 90%）",
    "pattern": "锤子(看多)"
  }
}
```

### 建议 FE / Mongo `run` 字段（Monday 落库）

在 diagnose log 的 `run` map 中增加（字段名与 Agent 对齐，便于 merge）：

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `lab_episode_details` | `map<string, LabEpisodeDetail>` | key = `detail_ref`（如 `buy:3`）；value = 与 lab-detail `detail` 相同 |
| `lab_episode_summaries` | 已有 | 卡片一行摘要；可选在 JEV 就绪后把 `price_analysis` 摘要行补全（非必须，详情以 `lab_episode_details` 为准） |

历史记录打开流程建议：

1. 若 `run.lab_episode_details[detail_ref]` 存在 → 直接展示，不调 Agent。
2. 否则若仍在 48h 内且会话在 Agent → 可调 `lab-detail`（会写缓存并返回 `cached: false/true`）。
3. 用户触发 prefetch / 点击详情且 lab-detail 成功返回后 → FE 将响应 `detail` 合并进 `run.lab_episode_details[detail_ref]` 再 `saveSignalDiagnoseLog`。

### HTTP：`POST /v1/signal-diagnose/lab-detail`

请求体不变：`session_id`、`run_id`、`detail_ref`、`mcp_token`（或 `X-MCP-Token`）。

成功响应示例：

```json
{
  "run_id": "flow-20261024T120000",
  "cached": false,
  "detail": {
    "ref": "buy:3",
    "side": "buy",
    "bar_index": 3,
    "price_analysis": { },
    "price_analysis_jev": { },
    "jev_unified": {
      "direction": "bull",
      "confidence": 0.71,
      "conflict": false,
      "breakdown": {
        "structure": "多周期结构倾向看多（置信 70%）",
        "pattern": "锤子(看多)"
      }
    },
    "kline_pattern_summary": "..."
  }
}
```

非 lab 诊断路径、workflow 硬拦截、JEV 进 `workflow_completed` 等行为 **未改**；JEV 仍为 lazy。
