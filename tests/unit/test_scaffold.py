"""Step 0 scaffold smoke tests."""

from __future__ import annotations

import geegoo_agent
from geegoo_agent.cli import build_parser, main


def test_package_version() -> None:
    assert geegoo_agent.__version__ == "0.1.0"


def test_cli_help_exits_zero() -> None:
    assert main([]) == 0


def test_cli_parser_has_run_and_resume() -> None:
    parser = build_parser()
    args = parser.parse_args(["run", "pre_market", "--dry-run"])
    assert args.command == "run"
    assert args.skill == "pre_market"
    assert args.dry_run is True

    resume_args = parser.parse_args(["resume", "--session", "sess-abc"])
    assert resume_args.command == "resume"
    assert resume_args.session == "sess-abc"


def test_tmp_output_dir_fixture(tmp_output_dir) -> None:
    assert tmp_output_dir.exists()
    assert tmp_output_dir.name == "data"
