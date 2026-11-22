---
name: diagnose_macro
description: 宏观日线四路 diagnose + JEV（HTTP skill，不覆盖日批表）
version: "1.0.0"
---

# diagnose_macro

与 `diagnose_strategy` 相同 JEV 流水线，但事实仅 **日线**（`POST /v1/diagnose/macro/*`）。`result` 无 `signal_frequency`，含 `monitor_session_close`。

**不会** 在单次调用时 PUT/覆盖宏观 canonical 日结果表；`internal/diagnoseskill.AssembleMacroSymbolLedgerRow` 仅组装落库字段（`jev_direction` 映射 long/short/neutral）供批处理使用。

## 触发

`POST /v1/skills/run`

```json
{
  "skill": "diagnose_macro",
  "macro": {
    "code": "00700.HK",
    "as_of": "可选"
  }
}
```

必填 `macro.code`；不要 frequency 字段。