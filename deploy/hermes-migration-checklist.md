# Hermes → GeeGoo Agent 切换检查清单

> **本清单仅记录切换步骤，不在部署时自动禁用 Hermes。**

## 前置条件

- [ ] `pytest -q` 全绿（含 `tests/e2e/test_pre_market_dry_run.py`）
- [ ] 服务器已完成 [tests/smoke/README.md](../tests/smoke/README.md) 冒烟表 #1–#4
- [ ] `/etc/geegoo-agent/config.json` 权限 `600`，密钥未提交 git

## 并行验证（建议 1–3 个交易日）

- [ ] systemd timer 已 enable：`systemctl is-enabled geegoo-agent-pre-market.timer`
- [ ] 与 Hermes 盘前 cron **同时运行**，对比产出：
  - [ ] 每股本地 `{code}-premarket.md` 存在
  - [ ] GeeGoo API 有 `createPreMarketReport` 记录
  - [ ] `bot_id` / `bot_name` / `bot_type` 非空
  - [ ] execution-log 含 supervisor 通过记录
- [ ] 抽检 ≥3 股报告与 Hermes 可比（结构、态度映射、关键数值）

## 切换日

- [ ] 确认当日 `check_trading_day` 为交易日（或已验证非交易日短路路径）
- [ ] 禁用 Hermes 盘前 cron（**手动操作**，记录原 cron 行以便回滚）
- [ ] 仅保留 `geegoo-agent-pre-market.timer` 触发
- [ ] 观察首个独立运行日：journalctl + execution-log + supervisor

## 回滚

- [ ] 重新启用 Hermes 盘前 cron
- [ ] `systemctl disable --now geegoo-agent-pre-market.timer`
- [ ] 保留 `/var/lib/geegoo-agent` 数据以便排查
