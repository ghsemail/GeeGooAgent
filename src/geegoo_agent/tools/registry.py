"""Tool registry and execution dispatch."""

from __future__ import annotations

from pydantic import ValidationError

from geegoo_agent.exceptions import GeeGooAgentError
from geegoo_agent.infra.sandbox_manager import SandboxManager
from geegoo_agent.llm.types import ToolSchema
from geegoo_agent.tools.base import BaseTool
from geegoo_agent.tools.types import ToolCallRequest, ToolContext, ToolResult


class ToolNotFoundError(GeeGooAgentError):
    """Unknown tool name."""


BOT_MUTATION_TOOLS: frozenset[str] = frozenset(
    {
        # Trading bots
        "create_dca_bot",
        "update_dca_bot",
        "delete_dca_bot",
        "create_grid_bot",
        "update_grid_bot",
        "delete_grid_bot",
        "create_smart_trade",
        "update_smart_trade",
        "delete_smart_trade",
        "create_hdg_bot",
        "update_hdg_bot",
        "delete_hdg_bot",
        # Reminders
        "create_dca_reminder",
        "update_dca_reminder",
        "delete_dca_reminder",
        "create_grid_reminder",
        "update_grid_reminder",
        "delete_grid_reminder",
        "create_smart_reminder",
        "update_smart_reminder",
        "delete_smart_reminder",
        # Prompt templates
        "create_competitor_prompt_template",
        "edit_competitor_prompt_template",
        "delete_competitor_prompt_template",
        "create_etf_prompt_template",
        "edit_etf_prompt_template",
        "delete_etf_prompt_template",
        # Bot switch
        "switch_bot",
    }
)


class ToolRegistry:
    def __init__(self, sandbox: SandboxManager | None = None) -> None:
        self._tools: dict[str, BaseTool] = {}
        self._sandbox = sandbox or SandboxManager()

    def register(self, tool: BaseTool) -> None:
        self._tools[tool.name] = tool

    def get(self, name: str) -> BaseTool:
        if name not in self._tools:
            raise ToolNotFoundError(f"unknown tool: {name}")
        return self._tools[name]

    def list_names(self) -> list[str]:
        return sorted(self._tools.keys())

    def schemas(
        self,
        *,
        tool_filter: list[str] | None = None,
        mode: str = "scheduled",
    ) -> list[ToolSchema]:
        tools = list(self._tools.values())
        if mode in {"scheduled", "chat"}:
            tools = [t for t in tools if t.name not in BOT_MUTATION_TOOLS]
        if tool_filter is not None:
            allowed = set(tool_filter)
            tools = [t for t in tools if t.name in allowed]
        return [t.to_schema() for t in tools]

    def execute(self, call: ToolCallRequest, ctx: ToolContext) -> ToolResult:
        tool = self.get(call.name)
        try:
            params = tool.validate_params(call.arguments)
        except ValidationError as exc:
            return ToolResult(status="error", summary=str(exc), exit_code=1)
        if ctx.event_bus is not None:
            ctx.event_bus.emit("ToolCalled", {"tool": call.name, "step": ctx.step})
        result = self._sandbox.execute(tool, params, ctx)
        if ctx.event_bus is not None:
            ctx.event_bus.emit(
                "ToolCompleted",
                {"tool": call.name, "step": ctx.step, "status": result.status},
            )
        return result
