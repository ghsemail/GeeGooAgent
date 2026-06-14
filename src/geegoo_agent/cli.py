"""CLI entry point."""

from __future__ import annotations

import argparse
import os
import sys

from geegoo_agent.cli_meta import CLI_NAME
from geegoo_agent.doctor_cmd import run_doctor
from geegoo_agent.exceptions import ConfigError
from geegoo_agent.paths import default_config_path
from geegoo_agent.runtime.app import GeeGooApp
from geegoo_agent.runtime.chat_repl import ChatRepl
from geegoo_agent.setup_cmd import run_setup
from geegoo_agent.update_cmd import run_update

DEFAULT_CONFIG = str(default_config_path())


def _apply_mcp_token(mcp_token: str | None) -> None:
    if mcp_token:
        os.environ["MCP_TOKEN"] = mcp_token


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(prog=CLI_NAME, description="GeeGoo Agent CLI")
    sub = parser.add_subparsers(dest="command")

    run = sub.add_parser("run", help="Run a skill workflow")
    run.add_argument("skill", nargs="?", default="pre_market")
    run.add_argument("--dry-run", action="store_true")
    run.add_argument("--config", default=DEFAULT_CONFIG)
    run.add_argument(
        "--mcp-token",
        default=None,
        help="User MCP token (overrides config.json and MCP_TOKEN env)",
    )

    resume = sub.add_parser("resume", help="Resume a session from checkpoint")
    resume.add_argument("--session", required=True)
    resume.add_argument("--config", default=DEFAULT_CONFIG)
    resume.add_argument(
        "--mcp-token",
        default=None,
        help="User MCP token (overrides config.json and MCP_TOKEN env)",
    )

    setup = sub.add_parser(
        "setup",
        help="Install dependencies (editable) and configure GeeGoo + LLM",
    )
    setup.add_argument("--config", default=DEFAULT_CONFIG)
    setup.add_argument(
        "--provider",
        choices=["openai", "deepseek", "minimax"],
        default=None,
        help="LLM provider (skip prompt when set)",
    )
    setup.add_argument("--token-key", default=None, help="LLM API token_key")
    setup.add_argument("--mcp-token", default=None, help="GeeGoo user mcp_token")
    setup.add_argument(
        "--github-token",
        default=None,
        help="GitHub PAT for private-repo geegoo update (writes ~/.geegoo/github_token)",
    )
    setup.add_argument(
        "--tradingbot",
        default=None,
        help="TradingBot repo path to sync mk-/sk- Bearer keys",
    )
    setup.add_argument("--host", default="118.195.135.97", help="GeeGoo API host for sync")
    setup.add_argument(
        "--non-interactive",
        action="store_true",
        help="Only apply flags; do not prompt",
    )
    setup.add_argument(
        "--skip-install",
        action="store_true",
        help="Skip pip install -e (configuration only)",
    )

    chat = sub.add_parser("chat", help="Interactive chat with GeeGoo Agent (ReAct + tools)")
    chat.add_argument("--config", default=DEFAULT_CONFIG)
    chat.add_argument("--session", default=None, help="Resume a chat session id")
    chat.add_argument("--dry-run", action="store_true", help="Skip mutating API calls")
    chat.add_argument(
        "--mcp-token",
        default=None,
        help="User MCP token (overrides config.json and MCP_TOKEN env)",
    )

    doctor = sub.add_parser("doctor", help="Diagnose config, APIs, and LLM connectivity")
    doctor.add_argument("--config", default=DEFAULT_CONFIG)
    doctor.add_argument("--skip-llm", action="store_true", help="Skip LLM ping")
    doctor.add_argument("--skip-api", action="store_true", help="Skip HTTP API probes")

    update = sub.add_parser("update", help="Pull latest code and reinstall (Hermes-style)")
    update.add_argument("--branch", default="main", help="Git branch (default: main)")
    update.add_argument("--repo", default=None, help="Git remote URL (default: GEEGOO_REPO env)")
    update.add_argument(
        "--method",
        choices=["auto", "git", "tarball"],
        default="auto",
        help="auto: git pull if .git exists, else download tarball",
    )
    update.add_argument("--skip-pip", action="store_true", help="Only sync source, skip pip install")

    return parser


def main(argv: list[str] | None = None) -> int:
    args = build_parser().parse_args(argv)
    if args.command is None:
        build_parser().print_help()
        return 0

    dry_run_override = True if getattr(args, "dry_run", False) else None
    _apply_mcp_token(getattr(args, "mcp_token", None))

    try:
        if args.command == "run":
            app = GeeGooApp.from_config_path(args.config, dry_run=dry_run_override)
            result = app.run_skill(args.skill)
            print(f"session={result.session_id} status={result.status}")
            if result.last_error:
                print(f"error={result.last_error}", file=sys.stderr)
            app.close()
            return 0 if result.ok else 1

        if args.command == "resume":
            app = GeeGooApp.from_config_path(args.config, dry_run=dry_run_override)
            result = app.resume_session(args.session)
            print(f"session={result.session_id} status={result.status}")
            if result.last_error:
                print(f"error={result.last_error}", file=sys.stderr)
            app.close()
            return 0 if result.ok else 1

        if args.command == "chat":
            app = GeeGooApp.from_config_path(args.config, dry_run=dry_run_override)
            try:
                repl = ChatRepl.from_app(
                    app,
                    session_id=args.session,
                    dry_run=args.dry_run,
                    config_path=args.config,
                )
                code = repl.run()
            finally:
                app.close()
            return code

        if args.command == "doctor":
            return run_doctor(
                args.config,
                skip_llm=args.skip_llm,
                skip_api=args.skip_api,
            )

        if args.command == "update":
            return run_update(
                method=args.method,
                branch=args.branch,
                repo=args.repo,
                skip_pip=args.skip_pip,
            )

        if args.command == "setup":
            from pathlib import Path

            tradingbot = Path(args.tradingbot) if args.tradingbot else None
            run_setup(
                args.config,
                provider=args.provider,
                token_key=args.token_key,
                mcp_token=args.mcp_token,
                github_token_value=args.github_token,
                tradingbot=tradingbot,
                host=args.host,
                interactive=not args.non_interactive,
                skip_install=args.skip_install,
            )
            return 0
    except ConfigError as exc:
        print(str(exc), file=sys.stderr)
        return 2

    return 0


if __name__ == "__main__":
    raise SystemExit(main())
