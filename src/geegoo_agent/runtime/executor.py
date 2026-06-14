"""Tool executor (L4)."""

from __future__ import annotations

from geegoo_agent.infra.events import InProcessEventBus
from geegoo_agent.tools.registry import ToolRegistry
from geegoo_agent.tools.types import ToolCallRequest, ToolContext, ToolResult


class Executor:
    def __init__(self, registry: ToolRegistry, event_bus: InProcessEventBus | None = None) -> None:
        self._registry = registry
        self._event_bus = event_bus

    def execute(self, call: ToolCallRequest, ctx: ToolContext) -> ToolResult:
        if ctx.event_bus is None and self._event_bus is not None:
            ctx.event_bus = self._event_bus
        return self._registry.execute(call, ctx)
