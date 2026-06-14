"""Sandbox-wrapped tool execution."""

from __future__ import annotations

import logging

from geegoo_agent.exceptions import SandboxError
from geegoo_agent.tools.base import BaseTool
from geegoo_agent.tools.types import ToolContext, ToolResult

logger = logging.getLogger(__name__)


class SandboxManager:
    """Execute tools inside workspace/network policy envelope."""

    def execute(self, tool: BaseTool, params, ctx: ToolContext) -> ToolResult:
        try:
            result = tool.run(params, ctx)
            return self._wrap(result)
        except SandboxError as exc:
            logger.warning("sandbox_blocked", extra={"tool": tool.name, "error": str(exc)})
            return ToolResult(status="error", summary=str(exc), exit_code=2)
        except Exception as exc:
            logger.warning("tool_failed tool=%s error=%s", tool.name, exc)
            return ToolResult(status="error", summary=str(exc), exit_code=1)

    def _wrap(self, result: ToolResult) -> ToolResult:
        if len(result.summary) > 2000:
            return ToolResult(
                status=result.status,
                summary=result.summary[:2000] + "…[truncated]",
                data=result.data,
                exit_code=result.exit_code,
            )
        return result
