---
name: prompt-template-admin
description: Prompt 模板管理、单项分析模板、EMA 模板、add_single_prompt、edit_prompt、switch 启用、竞品分析模板、ETF 模板、模板运营、Monday。用户要「加模板」「改 Prompt」「启用禁用模板」时触发。
---

# Prompt 模板运营

## 适用 Toolset

`prompt_admin` · `analyst_runtime`（验证列表）

## 两套存储（勿混用）

| 场景 | 读写 Tool | 运行时列表 |
|------|-----------|------------|
| **官方/用户单项模板**（EMA、指标、技术/基本面） | `add_single_prompt_template` · `edit_prompt_template` · `delete_prompt_template` · `switch_prompt_status` | `get_single_prompt_template`（仅 `switch=true`） |
| **竞品 / ETF 用户模板** | `create_competitor_prompt_template` · `create_etf_prompt_template` 及 edit/delete | **不在** `get_single_prompt_template`；供专项 analyze 流程 |

## 标准流程（single_prompt_template）

1. 查现有：`get_single_prompt_template`（`type`: `index`/`tech`/`fundamental`，可选 `period`）
2. 或按指标：`get_single_prompt_template_by_index`（`index`=variable，`period` 必填）
3. 新增：`add_single_prompt_template` → **默认未启用**
4. 启用：`switch_prompt_status`（`id`, `switch: true`）
5. 验证：再次 `get_single_prompt_template` 确认出现

## 标准流程（竞品 / ETF）

1. 用户确认 variable 格式（竞品：`00700.HK,09988.HK` 逗号分隔代码）
2. `create_competitor_prompt_template` 或 `create_etf_prompt_template`
3. 修改/删除用对应 `edit_*` / `delete_*`

## 硬规则

- 所有写操作须 **用户确认**
- `period` 为字符串数组；`type`（TemplateType）与筛选大类 `index/tech/fundamental` 对齐（见 agent-analyst 文档）
- `type=template` 分类模板仅 **admin** 可建
- 改完后提醒用户：未 switch 的模板 **不会** 进入 `get_mcp_analysis` 可选列表

## period 枚举（常用）

`daily` · `weekly` · `monthly` · `hourly` · `minutes` · `longterm` · `no_period`

## 反模式

- 用 edit 改竞品模板却去查 single_prompt 列表
- 新建后忘记 switch 就抱怨分析里找不到
