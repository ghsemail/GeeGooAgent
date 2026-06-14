"""Supervisor engine — load checks and verify working memory."""

from __future__ import annotations

from dataclasses import dataclass
from pathlib import Path
from typing import Any

import yaml
from pydantic import BaseModel, Field

from geegoo_agent.exceptions import ConfigError
from geegoo_agent.memory.models import PreMarketWorking
from geegoo_agent.supervisor.checks import CheckResult, CheckSpec, run_check


class SupervisorChecksFile(BaseModel):
    skill: str
    description: str = ""
    checks: list[CheckSpec] = Field(default_factory=list)


@dataclass
class SupervisorResult:
    ok: bool
    checks: list[CheckResult]

    @property
    def summary(self) -> str:
        if self.ok:
            return f"supervisor passed ({len(self.checks)} checks)"
        failed = [c for c in self.checks if not c.passed]
        parts = [f"{c.name}: {c.message}" for c in failed]
        return "supervisor failed — " + "; ".join(parts)

    @property
    def recoverable(self) -> bool:
        failed = [c for c in self.checks if not c.passed]
        return bool(failed) and all(c.recoverable for c in failed)


class SupervisorEngine:
    def __init__(self, project_root: Path | None = None) -> None:
        self._root = project_root or Path(__file__).resolve().parents[3]

    def checks_path(self, skill_name: str) -> Path:
        return self._root / "skills" / skill_name / "supervisor_checks.yaml"

    def load_checks(self, skill_name: str) -> SupervisorChecksFile:
        path = self.checks_path(skill_name)
        if not path.exists():
            raise ConfigError(f"supervisor checks not found: {path}")
        try:
            data: dict[str, Any] = yaml.safe_load(path.read_text(encoding="utf-8"))
        except yaml.YAMLError as exc:
            raise ConfigError(f"invalid supervisor YAML: {path}") from exc
        try:
            return SupervisorChecksFile.model_validate(data)
        except Exception as exc:
            raise ConfigError(f"invalid supervisor checks for {skill_name}: {exc}") from exc

    def verify(
        self,
        working: PreMarketWorking,
        checks: SupervisorChecksFile,
        *,
        workspace_root: Path,
        report_date: str | None = None,
    ) -> SupervisorResult:
        if working.is_trading_day is False:
            return SupervisorResult(ok=True, checks=[])

        results = [
            run_check(
                spec,
                working,
                workspace_root=workspace_root,
                report_date=report_date,
            )
            for spec in checks.checks
        ]
        ok = all(r.passed for r in results)
        return SupervisorResult(ok=ok, checks=results)
