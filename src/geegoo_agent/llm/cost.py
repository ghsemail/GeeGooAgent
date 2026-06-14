"""Token usage tracking (L1 CostManager MVP)."""

from __future__ import annotations

from dataclasses import dataclass

from geegoo_agent.llm.types import TokenUsage


@dataclass
class CostRecord:
    session_id: str
    step: int
    usage: TokenUsage


class CostManager:
    def __init__(self) -> None:
        self._records: list[CostRecord] = []

    def record(self, session_id: str, step: int, usage: TokenUsage) -> None:
        self._records.append(CostRecord(session_id=session_id, step=step, usage=usage))

    def session_total(self, session_id: str) -> TokenUsage:
        items = [r.usage for r in self._records if r.session_id == session_id]
        if not items:
            return TokenUsage(prompt_tokens=0, completion_tokens=0, model="")
        prompt = sum(u.prompt_tokens for u in items)
        completion = sum(u.completion_tokens for u in items)
        usd = sum(u.estimated_usd for u in items)
        model = items[-1].model
        return TokenUsage(
            prompt_tokens=prompt,
            completion_tokens=completion,
            model=model,
            estimated_usd=usd,
        )

    def all_records(self) -> list[CostRecord]:
        return list(self._records)
