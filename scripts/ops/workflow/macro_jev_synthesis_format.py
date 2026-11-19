"""Normalize macro pool/overall JEV chat output for canonical storage and Monday UI."""

from __future__ import annotations

import re

# Stored pool/overall reasons should be readable in Chinese, not long English scratchpads.
DEFAULT_POOL_JEV_REASON_MAX = 8000
_CJK = re.compile(r"[\u4e00-\u9fff]")
_MD_HEADING = re.compile(r"^##\s+", re.MULTILINE)
_EN_PREAMBLE = re.compile(
    r"^(?:let me synthesize|given inputs|i'll synthesize|thinking)",
    re.I | re.MULTILINE,
)


def _cjk_ratio(text: str) -> float:
    if not text:
        return 0.0
    cjk = len(_CJK.findall(text))
    return cjk / max(len(text), 1)


def format_pool_jev_reason(text: str, *, max_len: int = DEFAULT_POOL_JEV_REASON_MAX) -> str:
    """Prefer Chinese Markdown body; drop leading English chain-of-thought when possible."""
    raw = (text or "").strip()
    if not raw:
        return raw
    if _cjk_ratio(raw) >= 0.12 and not _EN_PREAMBLE.search(raw[:200]):
        out = raw
    else:
        headings = list(_MD_HEADING.finditer(raw))
        best_start = 0
        best_score = _cjk_ratio(raw)
        for m in headings:
            chunk = raw[m.start() :]
            score = _cjk_ratio(chunk[: min(2000, len(chunk))])
            if score > best_score:
                best_score = score
                best_start = m.start()
        # First substantial Chinese paragraph if no markdown structure.
        if best_score < 0.08:
            for m in re.finditer(r"[\u4e00-\u9fff]", raw):
                best_start = m.start()
                break
        out = raw[best_start:].lstrip()
        if not out:
            out = raw
    if len(out) > max_len:
        out = out[: max_len - 1].rstrip() + "…"
    return out