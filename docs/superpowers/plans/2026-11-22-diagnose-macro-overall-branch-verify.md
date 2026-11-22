# 分支验证：真实单池 `diagnose_macro_overall`（不合并、不部署生产）

在**已授权隔离环境**（非生产 `3499`、非线上 macro-api 写库）验证 skill 真实调用 GeeGooData Choice/Chat。

## 前置

| 配置项（仅名称，勿入库明文） | 用途 |
|------------------------------|------|
| `GEEGOO_AGENT_OP_HTTP_URL` | Agent runtime 基址（分支部署或本地 `agent-runtime`） |
| `GEEGOO_AGENT_RUNTIME_API_KEY` | `POST /v1/skills/run` Bearer |
| `GEEGOO_DATA_CN_TOKEN`（或环境约定的 Data decision token） | JEV `Choice`/`Chat` Bearer |
| `GEEGOO_DATA_HTTP_URL`（可选） | Data API 基址，默认可用环境内约定端口 |

## 步骤

1. 检出 **GeeGooAgent** 分支 `cursor/macro-overall-jev-skill-1cb6`，构建并启动 runtime（环境 `install` 已覆盖则 `go build` + 既有启动脚本）。
2. 单池请求体（编排日 `2026-11-22`，crypto short + 已落地池理由摘录）：

```json
{
  "skill": "diagnose_macro_overall",
  "macro_overall": {
    "date": "2026-11-22",
    "pools": [{
      "pool_workflow": "crypto",
      "jev_direction": "short",
      "jev_reason": "<canonical 池理由全文>"
    }]
  }
}
```

3. 运行仓库脚本（记录壁钟与脱敏回包）：

```bash
cd GeeGooAgent/scripts/ops/workflow
python3 e2e_single_crypto_overall_skill.py
```

4. 验收：`status==ok`、`result.date==2026-11-22`、`detail_ref==diagnose_macro_overall`、`choices` 三项概率 ∈ [0,1]、`confidence` 与推荐项一致；失败时 `status==failed` 且无伪造 `neutral`。
5. **勿**对 macro-api 写 overall success（单池非七池）；生产 ledger/`PUT overall` 仍待正式 workflow。

## Signal 分支（可选同环境）

检出 `cursor/macro-overall-jev-choices-1cb6`，本地 Mongo 跑：

`go test ./internal/macro -count=1`

验证 overall `choices` GET 与 `ExpectedPoolMacroDate` 回归。