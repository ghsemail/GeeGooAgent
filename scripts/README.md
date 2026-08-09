# GeeGooAgent 运维脚本

本目录集中存放 Agent 相关的部署、探测、验证与工作流脚本。**不要**在根目录新增零散脚本，按用途放入 `ops/` 子目录。

## 目录结构

| 路径 | 用途 | 示例 |
|------|------|------|
| `install.sh` | 官方一键安装（GitHub raw 引用，**勿移动**） | `curl …/scripts/install.sh \| bash` |
| `install-go.sh` / `ensure-go.sh` | Go 工具链安装 | 服务器初始化 |
| `hooks/` | Git / CI 钩子示例 | `audit-tool.example.sh` |
| `ops/deploy/` | 部署、nginx 补丁、快速上线 | `deploy_agent_server.py` |
| `ops/probe/` | 联调探测、诊断、审计 | `probe_agent_bff.py` |
| `ops/verify/` | 验收与 E2E 验证 | `verify_agent_chat_e2e.py` |
| `ops/workflow/` | 盘前/盘中/盘后手工触发 | `run_hk_intraday_live.py` |
| `ops/fix/` | 一次性修复、同步、迁移 | `fix_llm_key.py` |
| `ops/misc/` | 其他运维工具 | `find_ghsemail_user.py` |
| `dev/` | 本地开发 / bench / Go 小工具 | `bench_dashboard.py` |

## 约定

- **临时脚本**：以 `_` 开头，仅本机使用，已在 `.gitignore` 忽略；用完删除。
- **探测输出**：`probe_*_out.txt`、`*_result.json` 等不得提交；跑完即删。
- **服务器**：`geegoo-agent` 与仓库 `scripts/` 对齐（`git pull`）；TradingBot/GeeGooBot 上散落的 `_*.py` 为历史临时文件，可安全删除。

## 常用命令

```bash
# 盘中决策 live 测试（港股）
python scripts/ops/workflow/run_hk_intraday_live.py

# 清空用户全部盘中报告
python scripts/ops/workflow/clear_all_intraday_reports.py

# Agent 部署
python scripts/ops/deploy/deploy_agent_server.py
```
