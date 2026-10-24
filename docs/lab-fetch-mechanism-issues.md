# Lab / diagnose 取数机制问题清单（Agent 侧）

供策略开发转述用户与排期。

## 已在 main（PR #86）缓解

- 卡片 light enrich 空 MTF 壳 → 不再输出 `kline_pattern_mtf`
- 60m 阶梯去掉 4h/240m；align_note 区分未加载/不支持/真失败
- lab PA：`CachedLabBarsFetcher` + 长窗本地截断

## 本 PR 新增

| 问题 | 表现 | 处理 |
| --- | --- | --- |
| lab-api 无重试 | 一次 poll timeout/5xx → PA「拉数失败」 | `HTTPClient.FetchBars` 最多 3 次退避新 job |
| MTF probe 无重试 | 瞬时 market fetch failed | probe 最多 3 次退避（unsupported 不重试） |
| phase ctx 已取消 | 与真失败混为「拉数失败」 | PA 行「取数超时或已取消」 |

## 生产复验（flow-20261024T151342 @ fe7574e）

整机日志无「拉数失败」/ MTF align 失败 / phase 超时；卡片 light enrich 无 MTF 壳。残余「拉数失败」反馈需先核对 FE 是否用 **episode 列表序号** 拼 `detail_ref`（应为 `start_idx`，见 `docs/lab.md` §detail_ref）。

## 仍存在的机制风险（未靠抬超时糊弄）

1. **`DefaultLabDetailPhaseTimeout=8s` 包住整段 MTF probe + 多 TF PA**  
   单次 5m lab-api 冷取数生产实测 ~17s；8s 父 ctx 取消时 PA 行显示「取数超时或已取消」而非重试失败。  
   **建议**：phase 只包 MTF 或拆分 PA 独立 budget / 首取数后走缓存；另 PR 小步调整，不单靠 60s+。

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
