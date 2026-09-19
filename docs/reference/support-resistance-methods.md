# 支撑位与阻力位：方法索引与 GeeGoo 现状

> **Key Level Engine 的计算流水线**（Swing → Profile → 聚类 → Zone）请以 **[key-level-calculation-logic.md](./key-level-calculation-logic.md)** 为准。  
> 完整产品规范见桌面 **`geegoo-key-level-engine.md`**。

本文保留：**业界方法清单**、**API/盘前对照**、**快速查阅表**。

---

## 1. GeeGoo 目标形态（摘要）

- 输出 **Zone + Evidence + Confluence**，不是单点 `support`/`resistance`。
- **Primary**：Swing、Previous H/L、Volume Profile（POC/VAH/VAL/HVN/LVN）。
- **Reference**：Pivot（P/S1/R1/S2/R2）。
- **Context**：ATR（定宽）、BB、**MA20/60/120/250**、Session/Anchored VWAP。
- **Evidence**：资金流/分布（不直接造价；hint 或价格区间 → Confluence）。
- **Weak**：心理整数位（低权重 Candidate）；**Deprecated 作 Primary**：QFL、SAR、单 BB → `legacy` / `context`。

---

## 2. 方法索引（按类别）· 实现状态

| 类别 | 方法 | Engine 中的角色 | 实现 |
|------|------|-----------------|------|
| 结构 | Swing H/L | P0 Candidate 核心 | ✅ Daily / Weekly / **60m** |
| 结构 | PDH/PDL/PWH/PWL | P0 Candidate | ✅ |
| 结构 | Gap | P2 Candidate | ✅ |
| 量能 | POC / VAH / VAL / HVN / **LVN** | P1 Candidate 核心 | ✅ **Daily + Weekly** Profile |
| 参考 | Pivot P,S1,R1 (+S2/R2) | P1 Reference | ✅ |
| 波动 | ATR(14) | 聚类阈值 + Zone 半宽 | ✅ |
| 上下文 | Bollinger / **MA** / VWAP | Context only | ✅ BB + **MA** + **session_vwap** + **anchored_vwap** |
| 证据 | Capital flow / distribution | Confluence +1 | ✅ `capital_evidence` + 盘前 hint |
| 弱 | 整数关口、Fib、SAR、LLM 裸价 | 非 Primary | ✅ 整数 **weak candidate**；SAR/BB 仅 context/legacy；Fib/TPO/ML **未作 Primary**（符合 §28） |

详细公式见 **[key-level-calculation-logic.md](./key-level-calculation-logic.md)**。  
代码：`GeeGooSignal/internal/keylevel`，HTTP v2：`GeeGooSignal/docs/getSupportingPrice.md`。

---

## 3. `getSupportingPrice`（当前）

| 块 | 内容 |
|----|------|
| `data.judgment` | 主支撑/阻力带 + `regime` + `summary` |
| `data.candidates` | 次级带（可选） |
| `data.refs` | `bars`、`atr14`、`pivot`、`evidence`、`capital_hint` |
| `data.extra` | `include_extra=true` 时扩展指标与全 zone |
| `data.legacy` | `include_legacy=true`：网格 QFL/BB/SAR/high/low |

无 `bars` 时服务端拉 daily×120、weekly×52、60m×120（`include_60m` 默认 true）。

---

## 4. 日报 / 盘前

| 报告 | S/R |
|------|-----|
| 个股盘前 | **`key_levels`** → `get_key_levels`（Engine）+ **`MergeKeyLevels`**（weekly MCP 文本交叉校验） |
| 市场盘前 / 个股盘后 | 仍无全市场 Engine 重算 |

---

## 5. 推荐产品层 5 类（与 Engine 映射）

| 用户标签 | Engine 来源 | 实现 |
|----------|-------------|------|
| 周线关键位 | Weekly Swing + **Weekly Profile** + weekly 文本校验 | ✅ |
| 结构位 | Daily/**60m** Swing + Prev H/L | ✅ |
| 当日参考 | Pivot P/S1/R1 | ✅ |
| 波动带 | Context BB（或 Zone 辅证） | ✅ |
| 资金量能区 | Capital Evidence on nearest Zone | ✅ |

---

*修订：2026-09-19 · 与 Key Level Engine 实现对齐*
