---
name: custom-signal-admin
description: 定制策略、custom_signal、MACDResonance、custom.index、signal_id、supported_frequencies、定制信号 CRUD、策略注册表。用户要「定制信号」「MACD 共振」「改策略参数」时触发。
---

# 定制策略（custom_signal_strategies）

## 适用 Toolset

`custom_signal` · `strategy` · `bot-manager`（配置 Bot 时衔接）

## 标准流程

### 浏览

1. `get_custom_signal_for_skill` — Skill 友好列表（中文 name、`custom`、`supported_frequencies`）
2. 或 `get_custom_signal` — 完整 i18n
3. `get_all_custom_signal_id` — 仅 ID

### 新增 / 编辑前

1. **`get_custom_strategy_definitions`** — 查注册表合法 **`custom.index`**（如 `MACDResonance`）、默认 param、`param_schema`
2. 按 schema 组装 `custom`：`index`、`type`（`signal`/`flag`）、`param`
3. **不要**在文档里存 `frequency`；周期由注册表 → **`supported_frequencies`** 只读返回

### 写入

- 新增：`add_custom_signal`（name/brief/info i18n + custom + video_url/img_url）
- 更新：`edit_custom_signal`（`signal_id`）
- 删除：`delete_custom_signal`
- 全部须 **用户确认**

### 接到 Bot

`buy_signal` / `sell_signal` 项使用 **`index`**（= `custom.index`），外加 `type`、`param`；Bot 顶层 **`frequency`** 取 `supported_frequencies` 之一（如 `15m`、`5m`）。

## 硬规则

- 字段统一 **`custom.index`**（废弃 `custom.strategy`）
- 新增前 index **必须**在注册表存在
- `supported_frequencies` 为空时：检查 Mongo 是否仍用旧字段，或让运维确认 catalog 版本

## 反模式

- 未查 definitions 就 add
- 把 timeframe 写进 `param` 而非 Bot frequency
