"""Search past ``geegoo chat`` sessions (Hermes-style recall)."""

from __future__ import annotations

import json
import re
from dataclasses import dataclass
from typing import Any

from geegoo_agent.runtime.chat_session import ChatSession, ChatSessionStore

_PRICE_TOOLS = frozenset({"get_current_price", "get_ticker"})
_SEARCH_TOOLS = frozenset({"search_code"})


@dataclass(frozen=True)
class StockEvent:
    code: str | None = None
    query: str | None = None
    price: float | None = None
    tool: str = ""
    summary: str = ""


@dataclass(frozen=True)
class SessionRecallHit:
    session_id: str
    updated_at: str
    score: int
    user_queries: list[str]
    stock_events: list[StockEvent]
    snippet: str


def _parse_tool_content(content: str | None) -> dict[str, Any]:
    if not content:
        return {}
    try:
        payload = json.loads(content)
    except (json.JSONDecodeError, TypeError):
        return {}
    return payload if isinstance(payload, dict) else {}


def extract_stock_events(session: ChatSession) -> list[StockEvent]:
    """Pull price/search events from a persisted chat session."""
    events: list[StockEvent] = []
    pending: dict[str, tuple[str, dict[str, Any]]] = {}

    for raw in session.messages:
        role = raw.get("role")
        if role == "assistant":
            for call in raw.get("tool_calls") or []:
                name = str(call.get("name", ""))
                args = call.get("arguments") or {}
                if not isinstance(args, dict):
                    args = {}
                call_id = str(call.get("id", ""))
                if name in _PRICE_TOOLS or name in _SEARCH_TOOLS:
                    pending[call_id] = (name, args)
        elif role == "tool":
            call_id = str(raw.get("tool_call_id", ""))
            if call_id not in pending:
                continue
            name, args = pending.pop(call_id)
            payload = _parse_tool_content(raw.get("content"))
            data = payload.get("data") if isinstance(payload.get("data"), dict) else payload
            summary = str(payload.get("summary", ""))
            code = None
            price = None
            query = None
            if isinstance(data, dict):
                code = data.get("code") or args.get("code")
                if data.get("price") is not None:
                    price = float(data["price"])
            if name == "search_code":
                query = str(args.get("regex") or args.get("query") or "")
            if name in _PRICE_TOOLS and args.get("code"):
                code = str(args.get("code"))
            if price is None and "price=" in summary:
                match = re.search(r"price=([0-9.]+)", summary)
                if match:
                    price = float(match.group(1))
            events.append(
                StockEvent(
                    code=str(code) if code else None,
                    query=query or None,
                    price=price,
                    tool=name,
                    summary=summary[:200],
                )
            )
    return events


def _user_queries(session: ChatSession) -> list[str]:
    queries: list[str] = []
    for raw in session.messages:
        if raw.get("role") == "user" and raw.get("content"):
            text = str(raw["content"]).strip()
            if text and not text.startswith("/"):
                queries.append(text)
    return queries


def _session_corpus(session: ChatSession) -> str:
    parts: list[str] = []
    parts.extend(_user_queries(session))
    for event in extract_stock_events(session):
        if event.code:
            parts.append(event.code)
        if event.query:
            parts.append(event.query)
        if event.summary:
            parts.append(event.summary)
    return " ".join(parts).lower()


def _build_snippet(session: ChatSession, events: list[StockEvent]) -> str:
    queries = _user_queries(session)
    priced = [e for e in events if e.code and e.tool in _PRICE_TOOLS]
    if priced:
        last = priced[-1]
        price_part = f" price={last.price}" if last.price is not None else ""
        return f"查价 {last.code}{price_part}"
    if queries:
        return queries[-1][:120]
    if events:
        last = events[-1]
        if last.query:
            return f"搜索 {last.query}"
    return "(no stock activity)"


def _score_query(corpus: str, query: str) -> int:
    q = query.strip().lower()
    if not q:
        return 1 if corpus.strip() else 0
    score = 0
    if q in corpus:
        score += 3
    for token in re.split(r"\s+", q):
        token = token.strip()
        if len(token) >= 2 and token in corpus:
            score += 1
    for hint in ("股价", "价格", "查", "股票", "腾讯", "茅台", "price"):
        if hint in q and hint in corpus:
            score += 1
    return score


def search_past_sessions(
    store: ChatSessionStore,
    query: str,
    *,
    exclude_session_id: str | None = None,
    limit: int = 5,
    scan_limit: int = 30,
) -> list[SessionRecallHit]:
    """Search recent chat sessions for stock/price activity."""
    keys = [k for k in store._store.list_keys("chat") if k.startswith("chat/")]
    sessions: list[ChatSession] = []
    for key in keys:
        session_id = key.split("/", 1)[-1]
        if session_id == exclude_session_id:
            continue
        loaded = store.load(session_id)
        if loaded is not None and len(loaded.messages) > 1:
            sessions.append(loaded)

    sessions.sort(key=lambda s: s.updated_at, reverse=True)
    sessions = sessions[:scan_limit]

    hits: list[SessionRecallHit] = []
    for session in sessions:
        events = extract_stock_events(session)
        queries = _user_queries(session)
        corpus = _session_corpus(session)
        if not events and not queries:
            continue
        score = _score_query(corpus, query)
        if query.strip():
            if score <= 0:
                continue
        elif not events:
            continue
        else:
            score = max(score, 1)

        hits.append(
            SessionRecallHit(
                session_id=session.id,
                updated_at=session.updated_at,
                score=score,
                user_queries=queries[-5:],
                stock_events=events,
                snippet=_build_snippet(session, events),
            )
        )

    hits.sort(key=lambda h: (h.score, h.updated_at), reverse=True)
    return hits[:limit]


def hits_to_data(hits: list[SessionRecallHit]) -> dict[str, Any]:
    return {
        "count": len(hits),
        "matches": [
            {
                "session_id": hit.session_id,
                "updated_at": hit.updated_at,
                "score": hit.score,
                "snippet": hit.snippet,
                "user_queries": hit.user_queries,
                "stock_events": [
                    {
                        "code": e.code,
                        "query": e.query,
                        "price": e.price,
                        "tool": e.tool,
                        "summary": e.summary,
                    }
                    for e in hit.stock_events
                ],
            }
            for hit in hits
        ],
    }
