---
name: bot-manager
description: 创建修改删除交易机器人、DCA、GRID、SmartTrade、HDG、提醒 Reminder、bot_id、analysis_prompt_list、持仓、用户自己的 Bot 列表。用户说「建个网格」「我的机器人」「改 DCA」「提醒」时触发。
---

# 交易 Bot / 提醒 Bot 管理

## 适用 Toolset

`trading_bot` · `hedge_bot` · `reminder_manager` · `market`（搜码/持仓）

## 标准流程

### A. 查用户已有 Bot

1. 按类型 `list_dca_bots` / `list_grid_bots` / `list_smart_trades` / `list_hdg_bots` / `list_*_reminders`
2. 在返回列表中按 **stock_name、code、botname** 过滤（不要只靠 `search_code` 猜用户指的是哪只）
3. 查日志：`get_*_log` 或 `get_bot_log_by_type`（需 `bot_id`）

### B. 创建 / 修改（写操作）

1. `search_code` 确认标的 **code**（创建前必做）
2. 若配置 `attitude.analysis_prompt_list`：
   - `get_single_prompt_template`（`type=tech`，`period` 与 `analysis_period` 一致，常用 `daily`/`monthly`）
   - 填入返回的 **`prompt_id`** 数组
3. SmartTrade **sell_only**：创建前 `get_position` 确认持仓
4. 组装参数 → **等用户确认** → `create_*` / `update_*`
5. 创建前在同类 `list_*` 中检查重名

### C. 删除

`list_*` 定位 `bot_id` → 用户确认 → `delete_*`

## 硬规则

- 所有 **create/update/delete** 须用户明确批准（ApprovalGate）
- GRID **Reminder**：`frequency` 显式传 `60m`（若产品默认未覆盖）
- DCA 信号：创建前展示 `get_index_signals` 或 `get_signal_combinations`，让用户选定 **`signal_id`**
- 定制策略 Bot：`get_custom_signal_for_skill` 取 `custom.index`；Bot 顶层 **`frequency`** 须在 `supported_frequencies` 内，**不要**把周期写进 `custom.param`
- 查「我的 Bot」优先 **list**，不是 search_code

## 反模式

- 未 list 就编造 bot_id
- 未 search_code 就 create
- 把 scheduled 盘前 workflow 的 `get_report_bot_codes` 当作用户 Bot 列表

## 输出

操作前列出将改字段摘要；成功后给 bot_id / 名称便于后续引用。
