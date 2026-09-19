# GeeGoo 支撑 / 阻力计算逻辑（Key Level Engine 整理版）

> 依据：`geegoo-key-level-engine.md`（v1.0）  
> 定位：**可实现的计算规范**，与「方法百科」[`support-resistance-methods.md`](./support-resistance-methods.md) 互补。  
> 原则：**输出 Price Zone + Evidence + Confluence**，不是单个「神奇价格」。

---

## 1. 输出是什么

### 1.1 不要

```text
support = 438.2
resistance = 452.7
```

### 1.2 要

```text
Support Zone: 438 ~ 442
Evidence: Daily Swing Low, Weekly VAL, HVN, Previous Week Low
Confluence: High
Role / State: support · untested
```

### 1.3 核心公式链（总览）

```text
OHLCV (+ 可选 Volume Profile、资金流)
    → Candidate Levels（多源候选价）
    → Zone Clustering（ATR 定距聚类）
    → Confluence Scoring（证据加权）
    → Zone State（未测/测试/突破…）
    → Top Support / Resistance Zones
    → API / 盘前日报 / Agent
```

**ATR 的职责**：定 **Zone 半宽** 与 **聚类阈值**，**不**单独当作 S/R 价位。  
**Pivot / BB / MA / SAR**：**Reference 或 Context**，**不是** Primary Key Level（见 §8、§12）。

---

## 2. 输入数据

| 层级 | 序列 | 用途 |
|------|------|------|
| **P0 最低** | Daily、Weekly OHLCV | Swing、Prev H/L、ATR、聚类 |
| **P1** | + 60m（可选 15m） | 多周期 Swing、Profile |
| **P1** | + 分价成交量 | Volume Profile（POC/VAH/VAL/HVN/LVN） |
| **P2** | + 资金流 / 资金分布 | Evidence 加权，**不**直接输出单一 support 价 |

请求形态（目标 API，§10）：

```json
{
  "code": "000001.SZ",
  "bars": { "daily": [], "weekly": [], "60m": [] },
  "volume_profile": true,
  "capital_data": true
}
```

---

## 3. 阶段 A：候选位（Candidate Generators）

每个候选输出至少：`price`（或 `low/high`）、`type`、`timeframe`、`weight`（供聚类中心用）。

### 3.1 Swing High / Swing Low（结构 · **第一核心**）

对 K 线索引 `i`、左右窗口 `k`（按周期配置，勿写死）：

**Swing High**

```text
high[i] > high[i-k .. i-1]
high[i] >= high[i+1 .. i+k]
```

**Swing Low**

```text
low[i] < low[i-k .. i-1]
low[i] <= low[i+1 .. i+k]
```

| 周期 | 推荐 k |
|------|--------|
| Daily | 2 ~ 3 |
| Weekly | 1 ~ 2 |
| 60m / 15m | 2 ~ 5 |

**Swing 强度（可选，用于排序/评分）**

```text
reversal_distance = 后续最大反向移动 / ATR(14)
```

另可记：`volume_percentile`、`bars_since`、`touch_count`。

---

### 3.2 Previous High / Low（结构 · 高确定性参考）

候选类型（各取一条 high / low）：

```text
PDH / PDL     — Previous Day High / Low
PWH / PWL     — Previous Week High / Low
PMH / PML     — Previous Month High / Low（可选）
```

**规则**：只作 **Candidate**，不自动等于最终 Top1 S/R；进入聚类后与 Swing、Profile 等合并。

---

### 3.3 Volume Profile（量能 · **第二核心**，P1）

对 **Daily / Weekly / Rolling N-Day**（及可选 Fixed Range）做分价成交量直方图。

**POC**

```text
POC = argmax_price bin(volume)
```

**Value Area（默认 70% 总成交量）**

1. 定 POC  
2. 从 POC 向上下比较相邻桶成交量，每次扩展成交量更大的一侧  
3. 累计达 `value_area_percent`（默认 70%）  
4. 上沿 = **VAH**，下沿 = **VAL**

**HVN / LVN**

```text
HVN = 局部成交量峰（高接受区）→ 作 Zone 证据
LVN = 局部成交量谷（低接受 / 易穿越）→ 作突破上下文，少作硬 S/R
```

---

### 3.4 Pivot（Reference · P1）

上一周期 `H, L, C`（通常 **前一交易日**）：

```text
P  = (H + L + C) / 3
R1 = 2P - L          S1 = 2P - H
R2 = P + (H - L)     S2 = P - (H - L)
R3 = H + 2(P - L)    S3 = L - 2(H - P)
```

**第一阶段只产出候选**：`P, R1, S1`（第二阶段再加 R2/S2）。  
**权重建议偏低**（见 §5.2），定位 **盘前当日参考边界**，非 Major Structure。

---

### 3.5 Gap Zone（P2）

```text
向上缺口: today.low > previous.high  → Zone [previous.high, today.low]
向下缺口: today.high < previous.low → Zone [today.high, previous.low]
```

有效性另判：是否回补、是否重新进入。

---

### 3.6 心理整数（Weak Candidate · P2+）

```text
10, 20, 50, 100, 500, 1000 …（按标的量级动态）
strength_weight ≤ weak，不得单独成为 Major Zone
```

---

### 3.7 资金流 / 资金分布（Evidence · P2，**不造裸价**）

**禁止**：`support = 438`（仅从资金字段直接赋值）。

**允许**：在已有 **438~442** 候选带附近增加证据，例如：

```text
Evidence: capital_accumulation @ 435~440
Evidence: large_order_net_inflow concentrated @ 438~442
```

盘前已有 `get_capital_flow(DAY)`、`get_capital_distribution` → 只进 **Confluence Capital +1**，不替代结构/Profile。

---

### 3.8 动态指标（Context only · P3）

| 指标 | 计算 | 引擎中的角色 |
|------|------|----------------|
| **ATR(14)** | 标准 TR 平滑 | **Zone 宽度、聚类阈值、突破距离** |
| **Bollinger** | SMA20 ± 2σ | `context.bb_upper/lower`，超买超卖/波动边界 |
| **MA20/60/120/250** | SMA/EMA | 趋势上下文 |
| **VWAP** | 日内加权 | Session / Anchored VWAP，日内上下文 |
| **SAR / 当前 QFL / Fib / 纯 LLM 报价** | — | **不作为 Primary Candidate**（见 §12） |

---

## 4. 阶段 B：聚类成 Zone（**最关键一步**）

### 4.1 问题

候选例如：

```text
437.8 Swing Low | 439.2 Daily VAL | 440.0 Weekly VAL | 440.5 PWL | 441.1 HVN | 442.0 Pivot S1
```

→ **不是 6 个 Support**，而是 **一个 Support Zone**。

### 4.2 聚类距离

```text
cluster_threshold = max( ATR(14) * 0.35, tick_size * N )
```

两候选价 `a, b`：

```text
若 abs(a.price - b.price) <= cluster_threshold → 同一 Zone
```

**禁止**全局固定价格差（如永远 ±0.5 元）。

### 4.3 Zone 中心（加权）

```text
center = Σ(price_i × weight_i) / Σ(weight_i)
```

**初始权重（需回测校准，非市场真理）**

| 候选类型 | weight |
|----------|--------|
| Weekly Swing | 1.5 |
| Daily Swing | 1.2 |
| Intraday Swing | 0.8 |
| Weekly POC | 1.5 |
| Daily POC | 1.2 |
| VAH / VAL | 1.0 |
| HVN | 1.0 |
| Previous Week H/L | 1.2 |
| Previous Day H/L | 1.0 |
| Pivot P / S1 / R1 | 0.7 |
| Psychological | 0.4 |

**多周期系数（同时出现时提高 Confluence，可叠在 weight 或评分层）**

```text
Weekly = 1.5   Daily = 1.2   60m = 0.8   15m = 0.5
```

### 4.4 Zone 半宽

```text
half_width = max(
  ATR(14) * 0.25,
  candidate_spread * 0.5,
  tick_size * minimum_ticks
)

zone_low  = center - half_width
zone_high = center + half_width
```

**上限**：`zone_high - zone_low <= ATR(14) * 1.0`；超出则 **拆成多个 Zone**。

**禁止**唯一规则用固定百分比 ±1% 定宽（不同价位股票波动结构不同）。

---

## 5. 阶段 C：Confluence 评分

保存 **`confluence_score`（0~11）** + **完整 `sources` 列表**，而非单一 `strength=0.87`。

| 维度 | 上限 | 规则摘要 |
|------|------|----------|
| **Structure** | 3 | Swing H/L +2；Prev H/L +1；Repeated rejection +1 |
| **Volume** | 3 | POC +2；VAH/VAL +1；HVN +1 |
| **Multi-TF** | 2 | Weekly+Daily +2；Daily+Intraday +1 |
| **Reference** | 1 | Pivot +1 |
| **Capital** | 1 | 堆积/分布与 Zone 一致 +1 |
| **Recency** | 1 | 新近且未被有效突破 +1；反复穿越 0 |

Support / Resistance **分开**聚类与排序。

---

## 6. 阶段 D：Zone 状态与角色

### 6.1 状态 `state`

```text
untested → tested → rejected / broken → retested → invalidated
```

### 6.2 突破（勿用 close > R 一刀切）

```text
break_distance = close - zone_high   （阻力为例）

possible_breakout:
  break_distance > ATR * 0.15
  AND volume > SMA(volume,20) * 1.2

confirmed_breakout: 后续 K 线持续站在 Zone 外
```

阈值需回测校准。

### 6.3 角色 `role`（可翻转）

```text
resistance 突破 → 回踩 → 原阻力可能变为 support（role: both / 动态更新）
support 跌破 → 同理
```

---

## 7. 阶段 E：排序与对外展示

**分开** Support / Resistance 列表，建议每侧输出档位：

```text
nearest — 距现价最近
major   — 共振最强
next    — 次级
```

排序因子（综合，非 `score/distance` 单公式）：

1. Confluence  
2. Multi-Timeframe  
3. Structure strength  
4. Volume evidence  
5. Distance from current price  
6. Recency  

**Agent / 盘前摘要**（示例）：

```text
当前价 452.3；上方 454~458 为多周期共振阻力；下方 438~442 为主要支撑。
```

---

## 8. API 响应形状（当前契约，已实现）

**规范文档**：`GeeGooSignal/docs/getSupportingPrice.md`（响应体**不含** `engine_version`）

| 块 | 读者 | 要点 |
|----|------|------|
| **`current_price`** | 业务 / LLM | 现价（通常日线末收） |
| **`judgment`** | 业务 / 自动化 | 主支撑/阻力带（center/low/high、score、state、confidence、sources）+ `regime` + `summary` |
| **`candidates`** | 图表 / 深度 | 次级带（每侧最多约 3）；无则整键省略 |
| **`refs`** | LLM 核验 | `bars`、`atr14`、`pivot`、`evidence`（聚类候选价）、可选 `capital_hint` |
| **`extra`** | 重度分析 | `include_extra=true`：全 zone、MA/BB/VWAP 等 |
| **`legacy`** | 网格 App | `include_legacy=true`：QFL/BB/SAR/high/low（QFL ≈ 主档 center） |

**LLM**：读 `judgment` + `refs.evidence`；若调整价位需在答复中说明依据，勿在工具层覆盖引擎 JSON。

---

## 9. 实施分期（与计算模块对应）

| 阶段 | 计算模块 | 说明 |
|------|----------|------|
| **P0** | Swing、PDH/PDL/PWH/PWL、ATR、聚类、Confluence、MTF | 可替代现 `getSupportingPrice` 的 high/low/QFL/SAR 主输出 |
| **P1** | Volume Profile（POC/VAH/VAL/HVN/LVN）、Pivot P/S1/R1 | 能力质变 |
| **P2** | VWAP、Gap、Capital Evidence、State、Breakout/Retest | 与盘前资金数据对齐 |
| **P3** | Fib、MA、Bollinger、TPO、ML 校准权重 | 仅 Context 或二级证据 |

---

## 10. 现网对照（迁移时必读）

| 现组件 | 行为 | 与 Engine 关系 |
|--------|------|----------------|
| **`POST /getSupportingPrice`** | BBAND、SAR、区间 high/low、自定义 QFL | **Legacy**；QFL/SAR/单 BB **不应**再作 Primary；P0 后改为调 Key Level Engine 或薄封装 |
| **盘前 `ExtractWeeklyKeyLevels`** | 从 weekly MCP **文本** regex 抽两个 float | **过渡方案**；应逐步改为 Engine zones + LLM 只写 **解释**，或 weekly 文本与 Engine **交叉校验** |
| **盘前资金步骤** | reason 文案 | 接入 §3.7 **Capital Evidence** |
| **盘后日报** | 无 S/R 重算 | 引用盘前或当日 Engine 快照即可 |

---

## 11. 数据模型（Go 参考）

```go
type PriceZone struct {
    Low, High, Center float64
    Role              ZoneRole   // support | resistance | both | neutral
    State             ZoneState  // untested | tested | ...
    Confluence        int
    Sources           []LevelEvidence
    Timeframes        []string
    FirstSeen, LastTest time.Time
}

type LevelEvidence struct {
    Type, Timeframe string
    Price           float64
    Weight, Strength float64
}
```

---

## 12. 明确不作为「核心 S/R 算法」的项

以下仅 **secondary / context**，不得单独成为 Primary Key Level：

```text
当前 GeeGoo 自定义 QFL（与社区 QFL 语义不一致）
SAR 单独价位
单独 Bollinger 上下轨当作唯一 S/R
单独 MA / Fib / 整数位
LLM 直接生成的 support/resistance 数字（无 Candidate 回溯）
```

---

## 13. 一句话定稿

> **支撑/阻力 = 由结构（Swing/Prev H-L）、成交量分布（Profile）、多周期与可选资金证据形成的 Price Zone；ATR 定带宽，Confluence 定强度，State 定是否仍有效。**

---

## 14. 文档关系

| 文件 | 内容 |
|------|------|
| `geegoo-key-level-engine.md` | 完整产品设计 + 原则 + 业界对照（桌面 master） |
| **本文** `key-level-calculation-logic.md` | **计算步骤与公式** 整理 |
| `support-resistance-methods.md` | 业界方法百科 + 旧 API/日报现状 |

---

## 15. 实现状态（GeeGooSignal）

| 模块 | 路径 |
|------|------|
| P0 Engine | `GeeGooSignal/internal/keylevel` |
| P1 | Volume Profile（POC/VAH/VAL/HVN）、Pivot P/S1/R1 |
| P2 | Gap、Session/Anchored VWAP、Zone 状态与突破/回踩（含放量）、capital hint/区间共振、Weekly Profile、60m Swing |
| P3（Context） | BB、SAR、MA20/60/120/250、心理整数 weak candidate；Fib/TPO/ML 不作 Primary |
| 盘前步骤 | `key_levels` → `get_key_levels`（可选步骤，结果写入 working + 报告「结构引擎」段） |
| HTTP | `POST /getSupportingPrice` → `internal/signal/support` |
| Agent 工具 | `get_key_levels`（Chat / 手动） |
| 盘前 workflow | `SignalKeyLevelFetcher` + `MergeKeyLevels`（Engine 优先，weekly 文本交叉校验） |
| 行为 | 无 `bars` 时自动拉 `daily`×120 + `weekly`×52（+ 可选 60m）；网格读 `data.legacy.QFLSupport`/`QFLResistance` |
| Doctor | `tool probe: get_key_levels (getSupportingPrice)` 冒烟 |

*修订：2026-09-19 · judgment + refs + candidates（契约见 Signal 文档）*
