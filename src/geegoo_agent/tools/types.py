"""Tool execution types."""

from __future__ import annotations

from dataclasses import dataclass, field
from enum import StrEnum
from pathlib import Path
from typing import TYPE_CHECKING, Any, Literal

from geegoo_agent.clients.geegoo_bot import GeeGooBotClient
from geegoo_agent.clients.market import MarketClient
from geegoo_agent.infra.events import InProcessEventBus

if TYPE_CHECKING:
    from geegoo_agent.infra.state_store import FileStateStore
    from geegoo_agent.llm.gateway import ModelGateway
    from geegoo_agent.memory.working import WorkingMemoryStore


class ToolCategory(StrEnum):
    PERCEPTION = "perception"
    ANALYSIS = "analysis"
    DECISION = "decision"
    ACTION = "action"
    META = "meta"


@dataclass
class ToolContext:
    session_id: str
    mcp_token: str
    dry_run: bool
    workspace_root: Path
    market_client: MarketClient
    geegoo_bot_client: GeeGooBotClient | None = None
    working_store: WorkingMemoryStore | None = None
    state_store: FileStateStore | None = None
    project_root: Path | None = None
    feishu_webhook_url: str | None = None
    event_bus: InProcessEventBus | None = None
    llm_gateway: ModelGateway | None = None
    step: int = 0


@dataclass
class ToolResult:
    status: Literal["ok", "error", "skipped", "dry_run"]
    summary: str
    data: dict[str, Any] | None = None
    exit_code: int = 0


@dataclass
class ToolCallRequest:
    name: str
    arguments: dict[str, Any] = field(default_factory=dict)
