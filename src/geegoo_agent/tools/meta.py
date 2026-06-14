"""Meta / logging tools."""

from __future__ import annotations

from datetime import date

from pydantic import BaseModel, Field

from geegoo_agent.infra.sandbox import WorkspaceGuard
from geegoo_agent.tools.base import BaseTool
from geegoo_agent.tools.types import ToolCategory, ToolContext, ToolResult


class WriteExecutionLogParams(BaseModel):
    step: str = Field(description="Workflow step name")
    message: str = Field(description="Log message")
    status: str = Field(default="ok", description="Step status: ok|error|skipped")


class WriteExecutionLogTool(BaseTool):
    name = "write_execution_log"
    description = "Append a line to the session execution log in the workspace."
    category = ToolCategory.META
    input_model = WriteExecutionLogParams

    def run(self, params: WriteExecutionLogParams, ctx: ToolContext) -> ToolResult:
        guard = WorkspaceGuard(ctx.workspace_root)
        log_path = guard.resolve(f"{date.today().isoformat()}/execution-log.md")
        log_path.parent.mkdir(parents=True, exist_ok=True)

        line = f"- [{params.status}] {params.step}: {params.message}\n"
        if log_path.exists():
            existing = log_path.read_text(encoding="utf-8")
        else:
            existing = f"# Execution Log — {ctx.session_id}\n\n"
        log_path.write_text(existing + line, encoding="utf-8")

        return ToolResult(
            status="ok",
            summary=f"Logged step {params.step}",
            data={"path": str(log_path)},
        )
