"""Live progress hooks for ``geegoo chat`` — delegates to :class:`ChatUI`."""

from __future__ import annotations

from typing import Any, Callable, TextIO

from geegoo_agent.runtime.chat_ui import ChatUI, ProgressFn


def make_progress_writer(
    stdout: TextIO,
    *,
    enabled: bool = True,
    plain: bool | None = None,
    ui: ChatUI | None = None,
) -> ProgressFn:
    """Build a callback that prints ReAct steps as they happen."""
    surface = ui or ChatUI(stdout, plain=plain)

    def emit(event: str, data: dict[str, Any]) -> None:
        if enabled:
            surface.emit_progress(event, data)

    return emit
