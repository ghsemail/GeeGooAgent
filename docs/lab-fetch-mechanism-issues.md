# Lab / diagnose 取数机制问题清单（Agent 侧）

供策略开发转述用户与排期。

## 已在 main（PR #86）缓解

- 卡片 light enrich 空 MTF 壳 → 不再输出 `kline_pattern_mtf`
- 60m 阶梯去掉 4h/240m；align_note 区分未加载/不支持/真失败
- lab PA：`CachedLabBarsFetcher` + 长窗本地截断

## 本 PR 新增（#87 retry）与本 PR（lab PA 日/周）

| 问题 | 表现 | 处理 |
| --- | --- | --- |
| lab-api 无重试 | 一次 poll timeout/5xx → PA「拉数失败」 | `HTTPClient.FetchBars` 最多 3 次退避新 job |
| MTF probe 无重试 | 瞬时 market fetch failed | probe 最多 3 次退避（unsupported 不重试） |
| phase ctx 已取消 | 与真失败混为「拉数失败」 | PA 行「取数超时或已取消」 |
| **CachedLabBarsFetcher 负缓存** | 锚点 daily lab 失败后 PA 同 key 直接「拉数失败」；MTF 日/周已有 probe 数据 | 仅缓存 ok 且非空；PA 独立 `DefaultLabDetailPATimeout`；MTF probe 回退；PA 多 TF 并行 |

## 生产复验（flow-20261024T151342 @ fe7574e）

整机日志无「拉数失败」/ MTF align 失败 / phase 超时；卡片 light enrich 无 MTF 壳。残余「拉数失败」反馈需先核对 FE 是否用 **episode 列表序号** 拼 `detail_ref`（应为 `start_idx`，见 `docs/lab.md` §detail_ref）。

## 仍存在的机制风险（未靠抬超时糊弄）

1. **`DefaultLabDetailPhaseTimeout=8s` 仅包 MTF probe**（PA 已拆到 45s budget）。冷 lab-api 仍可能单 TF 慢；依赖 run 级正缓存 + MTF 回退。

2. **`BarsFetcher` 仅 `(bars, ok)`**  
   上层无法区分 unsupported / 4xx / 超时 / 空 bars，PA 只能粗分「拉数失败」与「取数超时或已取消」。  
   **建议**：lab 模块可选 `FetchBarsResult` 带 `Reason` enum（后续契约）。

3. **lab-worker 并发 4**  
   FE prefetch 多 ref 仍会排队；重试换新 job 可能加剧队列。  
   **建议**：坚持 run 级缓存 + 少打 job；必要时 lab-api 侧 dedupe（Signal 仓）。

4. **归档 `KlinePatternDisplayEnabled=true` 非 lab 路径**  
   仍会全量 MTF probe（含历史 4h 阶梯 fill_four）。生产默认应关；若误开仍可能拉数。  
   **建议**：配置门禁 + 文档；非 lab 不 probe 4h。

5. **prewarm 在 eval_done 卡片之后**  
   首屏不依赖 MTF；lab-detail 首点仍可能冷启动多 TF。依赖 cache 第二次起改善。

## 验证命令

```bash
go test ./internal/lab/signalclient/... ./internal/lab/klinepattern/... \
  ./internal/lab/priceanalysis/... ./internal/workflow/strategydev/... -count=1
```
