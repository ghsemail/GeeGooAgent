"""Pre-market supervisor entry point."""

from __future__ import annotations

from datetime import date
from pathlib import Path

from geegoo_agent.memory.models import PreMarketWorking
from geegoo_agent.supervisor.engine import SupervisorEngine, SupervisorResult


def run_pre_market_supervisor(
    working: PreMarketWorking,
    *,
    workspace_root: Path,
    project_root: Path | None = None,
    report_date: str | None = None,
) -> SupervisorResult:
    """Run post-workflow acceptance checks for pre_market skill."""
    engine = SupervisorEngine(project_root)
    checks = engine.load_checks("pre_market")
    return engine.verify(
        working,
        checks,
        workspace_root=workspace_root,
        report_date=report_date or date.today().isoformat(),
    )
