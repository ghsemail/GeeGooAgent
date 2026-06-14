# L5 — Skill 系统

## 职责

- 定义任务边界（盘前/盘后/盘中/策略/Bot）
- 提供 workflow、template、supervisor_checks
- 声明本 Skill 可用的 Tool 子集

## Skill 目录结构

```
skills/<name>/
├── SKILL.md                 # frontmatter + 描述 + 触发条件
├── workflow.md              # 步骤指南（无 API URL）
├── template.md              # 报告模板（如有）
└── supervisor_checks.yaml   # 机器可读验收项
```

## SkillLoader

**路径**：`src/geegoo/runtime/skill_loader.py`

```python
class LoadedSkill:
    name: str
    system_prompt: str          # 合并后
    tool_filter: set[str]       # 允许的工具名
    supervisor_checks: list     # yaml 解析
    mode: Literal["scheduled", "interactive", "signal"]

class SkillLoader:
    def load(self, name: str, mode: RunMode) -> LoadedSkill: ...
```

### System Prompt 组装顺序

```text
prompts/identity.md
+ rules/*.md（按字母序）
+ skills/<name>/SKILL.md
+ skills/<name>/workflow.md（精简）
```

## Skill 全景


| Skill                | 来源         | Phase     | 触发          |
| -------------------- | ---------- | --------- | ----------- |
| `pre_market`         | geegoo | **1 MVP** | timer 08:00 |
| `post_market`        | geegoo | 2         | timer 17:00 |
| `intraday`           | geegoo | 3         | webhook     |
| `on_demand_analysis` | geegoo       | 4         | chat        |
| `strategy`           | geegoo       | 5         | chat        |
| `bot_manager`        | geegoo       | 6         | chat        |
| `bundled/`*          | 两者         | 0+        | tools 内部调用  |


## Tool 过滤规则


| 模式          | 排除                                              |
| ----------- | ----------------------------------------------- |
| scheduled   | 所有 `create_*_bot`, `delete`_*, `wait_for_human` |
| signal      | Bot CRUD                                        |
| interactive | 无（Bot 创建走 `wait_for_human`）                     |


## MVP 实现清单

- `SkillLoader.load("pre_market")`
- 解析 `supervisor_checks.yaml`
- `bundled/geegoo/` 文档 stub，`enabled: false`

