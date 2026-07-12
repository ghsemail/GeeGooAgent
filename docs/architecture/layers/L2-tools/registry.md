# L2 — ToolRegistry

## 职责

- 注册全部 Tool
- 按 Skill/模式过滤
- 导出 JSON Schema 供 LLM function calling
- 调度执行

## 接口

```python
class ToolRegistry:
    def register(self, tool: BaseTool) -> None: ...
    def schemas(self, skill: LoadedSkill) -> list[ToolSchema]: ...
    def get(self, name: str) -> BaseTool: ...
    def execute(self, call: ToolCall, ctx: ToolContext) -> ToolResult: ...

class BaseTool(ABC):
    name: str
    category: ToolCategory
    description: str
    Input: type[BaseModel]
    Output: type[BaseModel]

    def run(self, input: BaseModel, ctx: ToolContext) -> BaseModel: ...
```

## 过滤逻辑

```python
def schemas(self, skill: LoadedSkill) -> list[ToolSchema]:
    tools = self._all_tools
    if skill.mode == "scheduled":
        tools = [t for t in tools if t.name not in BOT_MUTATION_TOOLS]
    return [t.to_schema() for t in tools if t.name in skill.tool_filter]
```

## MVP 注册清单

见 [tool-catalog.md](./tool-catalog.md) — MVP 约 **19** 个 Tool；目标态 **~87** 个（含 geegoo 全量 Bot/Reminder/策略 + geegoo 实时/报告）。

## 模块文件（与 catalog §九 对齐）

| 文件                    | Tool 组                                                     |
| --------------------- | ---------------------------------------------------------- |
| `perceive.py`         | check_trading_day, search_code, get_position, fetch_*_news |
| `analyze.py`          | get_mcp_analysis, capital, attitude, reports, bot_log      |
| `analyze_strategy.py` | signals, generate_*, loopback                              |
| `analyze_logs.py`     | get_**bot_log, get**_reminder_log                          |
| `decide.py`           | recall_*, read_working_state                               |
| `act_reports.py`      | create/update_*_report, save_local, feishu                 |
| `act_reminders.py`    | dca/grid/smart reminder CRUD                               |
| `act_bots.py`         | dca/grid/smart/hdg bot CRUD                                |
| `meta.py`             | write_execution_log, spawn_subagent, wait_for_human        |

