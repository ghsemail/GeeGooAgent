"""Unit tests for pre-market supervisor."""

from __future__ import annotations

from datetime import date
from pathlib import Path

import pytest

from geegoo_agent.memory.models import BotStock, PreMarketWorking, StockWorkspace
from geegoo_agent.supervisor.engine import SupervisorEngine
from geegoo_agent.supervisor.pre_market import run_pre_market_supervisor

PROJECT_ROOT = Path(__file__).resolve().parents[2]
CODE = "00700.HK"


def _reported_working(session_id: str = "sess-sup") -> PreMarketWorking:
    return PreMarketWorking(
        session_id=session_id,
        phase="done",
        is_trading_day=True,
        bot_codes=[
            BotStock(
                code=CODE,
                stock_name="腾讯控股",
                bot_id="bot-1",
                bot_name="test-bot",
                bot_type="DCA",
            )
        ],
        stocks={
            CODE: StockWorkspace(
                code=CODE,
                stock_name="腾讯控股",
                bot_id="bot-1",
                bot_name="test-bot",
                bot_type="DCA",
                status="reported",
                attitude="bullish",
                report_id="rid-1",
                report_ref="reports/2026-06-05/00700.HK-premarket.md",
            )
        },
    )


def _write_report_md(workspace: Path, code: str = CODE) -> Path:
    report_date = date.today().isoformat()
    path = workspace / "reports" / report_date / f"{code}-premarket.md"
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text("# premarket report\n", encoding="utf-8")
    return path


@pytest.mark.unit
def test_supervisor_passes_when_reports_complete(tmp_path: Path) -> None:
    _write_report_md(tmp_path)
    result = run_pre_market_supervisor(
        _reported_working(),
        workspace_root=tmp_path,
        project_root=PROJECT_ROOT,
    )
    assert result.ok
    assert len(result.checks) == 5
    assert "passed" in result.summary


@pytest.mark.unit
def test_supervisor_fails_when_local_md_missing(tmp_path: Path) -> None:
    result = run_pre_market_supervisor(
        _reported_working(),
        workspace_root=tmp_path,
        project_root=PROJECT_ROOT,
    )
    assert not result.ok
    assert result.recoverable
    assert any(c.name == "reported_stocks_local_md" and not c.passed for c in result.checks)


@pytest.mark.unit
def test_supervisor_fails_when_report_id_missing(tmp_path: Path) -> None:
    working = _reported_working()
    working.stocks[CODE].report_id = None
    _write_report_md(tmp_path)
    result = run_pre_market_supervisor(
        working,
        workspace_root=tmp_path,
        project_root=PROJECT_ROOT,
    )
    assert not result.ok
    assert any(c.name == "reported_stocks_api_created" and not c.passed for c in result.checks)


@pytest.mark.unit
def test_supervisor_skips_checks_on_non_trading_day(tmp_path: Path) -> None:
    working = PreMarketWorking(
        session_id="sess-holiday",
        phase="done",
        is_trading_day=False,
    )
    result = run_pre_market_supervisor(
        working,
        workspace_root=tmp_path,
        project_root=PROJECT_ROOT,
    )
    assert result.ok
    assert result.checks == []


@pytest.mark.unit
def test_supervisor_loads_checks_yaml() -> None:
    engine = SupervisorEngine(PROJECT_ROOT)
    checks = engine.load_checks("pre_market")
    assert checks.skill == "pre_market"
    assert len(checks.checks) >= 4
    names = {c.name for c in checks.checks}
    assert "reported_stocks_local_md" in names
