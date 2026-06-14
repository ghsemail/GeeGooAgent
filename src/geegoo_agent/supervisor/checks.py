"""Individual supervisor check implementations."""

from __future__ import annotations

from dataclasses import dataclass
from datetime import date
from pathlib import Path
from typing import Literal

from pydantic import BaseModel, Field

from geegoo_agent.memory.models import PreMarketWorking
from geegoo_agent.runtime.pre_market_report import build_create_report_args
from geegoo_agent.tools.schemas import PreMarketReportCreate

CheckType = Literal["execution_log_contains", "stocks_status", "api_response", "file_exists"]


class CheckSpec(BaseModel):
    name: str
    type: CheckType
    substring: str | None = None
    expect_phase: str | None = None
    for_status: str | None = None
    require_fields: list[str] = Field(default_factory=list)
    pattern: str | None = None
    required: list[str] = Field(default_factory=list)


@dataclass
class CheckResult:
    name: str
    passed: bool
    message: str
    recoverable: bool = False


def _stocks_for_status(working: PreMarketWorking, status: str | None) -> list[str]:
    if status is None:
        return list(working.stocks.keys())
    return [code for code, ws in working.stocks.items() if ws.status == status]


def run_execution_log_contains(
    spec: CheckSpec,
    *,
    workspace_root: Path,
    report_date: str,
) -> CheckResult:
    substring = spec.substring or ""
    log_path = workspace_root / report_date / "execution-log.md"
    if not log_path.exists():
        return CheckResult(
            spec.name,
            False,
            f"execution log missing: {log_path}",
            recoverable=True,
        )
    content = log_path.read_text(encoding="utf-8")
    if substring not in content:
        return CheckResult(
            spec.name,
            False,
            f"execution log missing substring: {substring!r}",
            recoverable=True,
        )
    return CheckResult(spec.name, True, f"execution log contains {substring!r}")


def run_stocks_status(spec: CheckSpec, working: PreMarketWorking) -> CheckResult:
    if spec.expect_phase is not None and working.phase != spec.expect_phase:
        recoverable = working.phase in {"phase_a", "phase_b"}
        return CheckResult(
            spec.name,
            False,
            f"expected phase={spec.expect_phase}, got {working.phase}",
            recoverable=recoverable,
        )

    codes = _stocks_for_status(working, spec.for_status)
    if spec.for_status and not codes and working.stocks:
        skipped = all(ws.status == "skipped" for ws in working.stocks.values())
        if skipped:
            return CheckResult(spec.name, True, "all stocks skipped")

    missing: list[str] = []
    for code in codes:
        ws = working.stocks[code]
        for field in spec.require_fields:
            value = getattr(ws, field, None)
            if value is None or (isinstance(value, str) and not value.strip()):
                missing.append(f"{code}.{field}")

    if missing:
        return CheckResult(
            spec.name,
            False,
            f"missing fields: {', '.join(missing)}",
            recoverable=True,
        )
    if spec.expect_phase:
        return CheckResult(spec.name, True, f"phase={working.phase}")
    return CheckResult(spec.name, True, f"checked {len(codes)} stock(s)")


def run_file_exists(
    spec: CheckSpec,
    working: PreMarketWorking,
    *,
    workspace_root: Path,
    report_date: str,
) -> CheckResult:
    pattern = spec.pattern or ""
    codes = _stocks_for_status(working, spec.for_status)
    missing: list[str] = []
    for code in codes:
        rel = pattern.format(date=report_date, code=code)
        path = workspace_root / rel
        if not path.exists():
            missing.append(str(path))

    if missing:
        return CheckResult(
            spec.name,
            False,
            f"missing files: {', '.join(missing)}",
            recoverable=True,
        )
    return CheckResult(spec.name, True, f"all {len(codes)} report file(s) exist")


def run_api_response(spec: CheckSpec, working: PreMarketWorking) -> CheckResult:
    codes = _stocks_for_status(working, spec.for_status)
    errors: list[str] = []
    for code in codes:
        try:
            args = build_create_report_args(working, code)
        except Exception as exc:
            errors.append(f"{code}: build args failed: {exc}")
            continue
        missing = [field for field in spec.required if not args.get(field)]
        if missing:
            errors.append(f"{code}: missing {', '.join(missing)}")
            continue
        try:
            PreMarketReportCreate.model_validate(args)
        except Exception as exc:
            errors.append(f"{code}: schema invalid: {exc}")

    if errors:
        return CheckResult(
            spec.name,
            False,
            "; ".join(errors),
            recoverable=True,
        )
    return CheckResult(spec.name, True, f"validated {len(codes)} payload(s)")


def run_check(
    spec: CheckSpec,
    working: PreMarketWorking,
    *,
    workspace_root: Path,
    report_date: str | None = None,
) -> CheckResult:
    day = report_date or date.today().isoformat()
    if spec.type == "execution_log_contains":
        return run_execution_log_contains(spec, workspace_root=workspace_root, report_date=day)
    if spec.type == "stocks_status":
        return run_stocks_status(spec, working)
    if spec.type == "file_exists":
        return run_file_exists(spec, working, workspace_root=workspace_root, report_date=day)
    if spec.type == "api_response":
        return run_api_response(spec, working)
    return CheckResult(spec.name, False, f"unknown check type: {spec.type}")
