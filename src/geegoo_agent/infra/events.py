"""In-process event bus (L0)."""

from __future__ import annotations

import logging
from collections import defaultdict
from collections.abc import Callable
from typing import Any, Protocol

logger = logging.getLogger(__name__)

EventHandler = Callable[[dict[str, Any]], None]


class EventBus(Protocol):
    def emit(self, event: str, payload: dict[str, Any]) -> None: ...
    def on(self, event: str, handler: EventHandler) -> None: ...


class InProcessEventBus:
    """Synchronous in-process event bus; handler errors are logged, not propagated."""

    def __init__(self) -> None:
        self._handlers: dict[str, list[EventHandler]] = defaultdict(list)
        self.history: list[tuple[str, dict[str, Any]]] = []

    def on(self, event: str, handler: EventHandler) -> None:
        self._handlers[event].append(handler)

    def emit(self, event: str, payload: dict[str, Any]) -> None:
        record = dict(payload)
        self.history.append((event, record))
        for handler in self._handlers.get(event, []):
            try:
                handler(record)
            except Exception:
                logger.exception("event_handler_failed", extra={"event": event})
