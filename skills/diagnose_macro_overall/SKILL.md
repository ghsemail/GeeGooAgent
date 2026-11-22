# diagnose_macro_overall

宏观整体风险偏好 JEV：接受 1–7 个监控池的 `jev_direction`/`jev_reason`，经 GeeGooData Choice + Chat 输出整体 long/short/neutral、置信度与三选项概率。

- **不写** macro-api canonical 表；编排/workflow 负责 ledger 与 `PUT /macro/jev/results/overall`。
- 单池调用仅用于 skill 验证；正式 overall success 仍须七池齐套。
- 契约见 `docs/superpowers/specs/2026-11-22-diagnose-macro-overall-design.md`。