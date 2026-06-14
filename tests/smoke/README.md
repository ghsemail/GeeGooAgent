# 真机冒烟清单（Step 15）

自动化测试（`pytest`）默认 **mock 全部 HTTP/LLM**。本目录记录 **手动** 冒烟步骤，仅在部署服务器上执行。

## 环境准备

```bash
# 1. 安装（示例路径 /opt/geegoo-agent）
cd /opt/geegoo-agent
python3.11 -m venv venv
source venv/bin/activate
pip install -e .

# 2. 配置
sudo mkdir -p /etc/geegoo-agent /var/lib/geegoo-agent
sudo cp config.example.json /etc/geegoo-agent/config.json
sudo cp deploy/env.example /etc/geegoo-agent/env
# 编辑 config.json 与 env，填入真实密钥（勿提交 git）
sudo chmod 600 /etc/geegoo-agent/config.json /etc/geegoo-agent/env

# 3. 从 TradingBot 同步 Bearer，并写入 mcp_token（部署时配置，非每次调用传入）
python scripts/sync_config_from_tradingbot.py --tradingbot /path/to/TradingBot --output /etc/geegoo-agent/config.json
# 编辑 config.json：填入 "mcp_token": "<用户MCP令牌>"
# output_dir 建议："/var/lib/geegoo-agent/data"
```

## 冒烟表

| # | 操作 | 预期 | 记录 |
|---|------|------|------|
| 1 | `geegoo run pre_market --dry-run --config /etc/geegoo-agent/config.json` | exit 0；`status=completed` | ☐ 日期 / 操作人 |
| 2 | 非交易日：`geegoo run pre_market --config /etc/geegoo-agent/config.json`（节假日或周末） | exit 0；execution-log 含 `check_trading_day`；**不**调用后续 API；`phase=done` | ☐ |
| 3 | 交易日 dry-run 后检查产出 | `{output_dir}/{date}/execution-log.md` 存在；`reports/{date}/*-premarket.md` 与 mock 股数一致；supervisor 行 `[ok] supervisor` | ☐ |
| 4 | 交易日实盘（小流量：确认仅监控 1 股时可临时缩减 bot 列表） | 每股 md + API `report_id`；execution-log 无 `[error]` | ☐ |
| 5 | 杀进程后 resume | 运行中 `kill` 后：`geegoo resume --session <ID> --config /etc/geegoo-agent/config.json` → exit 0，supervisor 通过 | ☐ |

## 非交易日短路验证（#2 详解）

非交易日无需完整跑盘，重点验证 **短路路径**：

```bash
geegoo run pre_market --config /etc/geegoo-agent/config.json
echo $?   # 期望 0

LOG=/var/lib/geegoo-agent/data/$(date +%Y-%m-%d)/execution-log.md
grep check_trading_day "$LOG"
# 期望：is_trading_day=false，且无 get_report_bot_codes 后续步骤
```

查看 working 终态（可选）：

```bash
ls /var/lib/geegoo-agent/data/working/
# 对应 session JSON 中 phase=done, is_trading_day=false
```

## systemd 冒烟

```bash
sudo cp deploy/systemd/geegoo-agent-pre-market.service /etc/systemd/system/
sudo cp deploy/systemd/geegoo-agent-pre-market.timer /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable --now geegoo-agent-pre-market.timer
systemctl list-timers geegoo-agent-pre-market.timer
# 手动触发一次（不必等到 08:00）
sudo systemctl start geegoo-agent-pre-market.service
journalctl -u geegoo-agent-pre-market.service -n 50 --no-pager
```

## 通过标准

- 冒烟表 #1–#3 全部 ☐ → ✅
- #4–#5 在首个真实交易日完成
- 切换 Hermes 前完成 [deploy/hermes-migration-checklist.md](../../deploy/hermes-migration-checklist.md)
