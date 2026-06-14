# Supervisor（横切）

## 职责

Agent 声明完成后，**确定性验收**——弥补 Hermes「跑完没人检查」。

## 流程

```mermaid
flowchart TD
  Done["Agent 声明完成"]
  Load["加载 supervisor_checks.yaml"]
  Run["逐项检查"]
  Pass{"通过?"}
  Retry["resume 补跑"]
  Alert["飞书 + exit 1"]
  OK["Completed"]

  Done --> Load --> Run --> Pass
  Pass -->|是| OK
  Pass -->|可补| Retry
  Pass -->|硬失败| Alert
  Retry --> Done
```



## checks.yaml 类型


| type                     | 说明                      |
| ------------------------ | ----------------------- |
| `execution_log_contains` | 日志含关键字                  |
| `stocks_status`          | WorkingMemory 每股 status |
| `api_response`           | create_report 字段        |
| `file_exists`            | 本地 md 存在                |


## 盘前示例

见 `skills/pre_market/supervisor_checks.yaml`（实现时创建）。

## 代码

`src/geegoo/supervisor/engine.py`, `checks.py`

## MVP

全部 check 类型 + 单次补跑 resume。