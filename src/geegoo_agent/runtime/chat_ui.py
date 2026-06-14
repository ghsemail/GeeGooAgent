"""Hermes-inspired Rich terminal UI for ``geegoo chat``."""

from __future__ import annotations

import json
import os
import sys
import time
from pathlib import Path
from typing import Any, Callable, TextIO

from geegoo_agent.runtime.chat_banner import build_plain_banner, build_welcome_banner
from geegoo_agent.tools.registry import ToolRegistry

ProgressFn = Callable[[str, dict[str, Any]], None]

# GeeGoo palette (Hermes bright gold — no magenta/purple)
_C_GOLD = "#FFD700"
_C_AMBER = "#FFBF00"
_C_OK = "#8FBC8F"
_C_ERR = "#F08080"
_C_DIM = "#9CA3AF"
_C_BORDER = "#B8860B"

_GEEGOO_CHAT_THEME: dict[str, str] = {
    "markdown.h2": "bold #FFBF00",
    "markdown.h3": "bold #FFD700",
    "markdown.h4": "#FFD700",
    "markdown.block_quote": "#FFBF00",
    "markdown.list": "#FFD700",
    "markdown.item.number": "#FFD700",
    "markdown.code": "bold #FFBF00",
    "markdown.code_block": "#FFBF00",
    "markdown.link": "underline #FFD700",
    "markdown.link_url": "underline #FFBF00",
    "markdown.table.border": "#FFBF00",
    "markdown.table.header": "bold #FFD700",
}

_TOOL_EMOJI: dict[str, str] = {
    "search_code": "🔍",
    "get_current_price": "💹",
    "get_ticker": "💹",
    "get_position": "📊",
    "check_trading_day": "📅",
    "fetch_market_news": "📰",
    "fetch_stock_news": "📰",
    "get_mcp_analysis": "📊",
    "get_capital_flow": "💰",
    "get_capital_distribution": "💰",
    "get_stock_daily_reports": "📝",
    "list_today_reports": "📝",
    "write_execution_log": "📋",
    "recall": "🔍",
}


def _tool_emoji(name: str) -> str:
    return _TOOL_EMOJI.get(name, "⚡")


def _fmt_args_compact(arguments: dict[str, Any], *, limit: int = 48) -> str:
    if not arguments:
        return ""
    parts = [str(v) for v in arguments.values() if v not in (None, "", {}, [])]
    text = "  ".join(parts)
    if len(text) > limit:
        return text[: limit - 3] + "..."
    return text


def _fmt_args(arguments: dict[str, Any], *, limit: int = 72) -> str:
    try:
        text = json.dumps(arguments, ensure_ascii=False)
    except TypeError:
        text = str(arguments)
    if len(text) > limit:
        return text[: limit - 3] + "..."
    return text


def use_plain_ui(stdout: TextIO | None = None) -> bool:
    if os.environ.get("GEEGOO_CHAT_PLAIN", "").strip().lower() in {"1", "true", "yes"}:
        return True
    stream = stdout or sys.stdout
    return not stream.isatty()


class ChatUI:
    """Rich-rendered chat surface with plain-text fallback for pipes/tests."""

    def __init__(self, stdout: TextIO | None = None, *, plain: bool | None = None) -> None:
        self._stdout = stdout or sys.stdout
        self.plain = use_plain_ui(self._stdout) if plain is None else plain
        self._console = None
        self._tool_starts: dict[str, float] = {}
        self._tool_args: dict[str, str] = {}
        if not self.plain:
            from rich.console import Console
            from rich.theme import Theme

            self._console = Console(
                file=self._stdout,
                highlight=False,
                soft_wrap=True,
                theme=Theme(_GEEGOO_CHAT_THEME),
            )

    def _write(self, text: str) -> None:
        self._stdout.write(text)
        self._stdout.flush()

    def _rule_width(self) -> int:
        if self._console is not None:
            return max(40, min(80, self._console.width))
        return 40

    def _print_rule(self) -> None:
        line = "─" * self._rule_width()
        if self.plain:
            self._write(line + "\n")
            return
        assert self._console is not None
        self._console.print(f"[{_C_DIM}]{line}[/]")

    def print_banner(
        self,
        *,
        session_id: str,
        provider: str,
        model: str,
        registry: ToolRegistry,
        thinking: bool = False,
        dry_run: bool = False,
        workspace: Path | None = None,
        install_dir: Path | None = None,
        project_root: Path | None = None,
        api_hosts: dict[str, str] | None = None,
    ) -> None:
        if self.plain:
            self._write(
                "\n"
                + build_plain_banner(
                    session_id=session_id,
                    provider=provider,
                    model=model,
                    registry=registry,
                    thinking=thinking,
                    dry_run=dry_run,
                    workspace=workspace,
                    install_dir=install_dir,
                    project_root=project_root,
                    api_hosts=api_hosts,
                )
            )
            return

        assert self._console is not None
        build_welcome_banner(
            self._console,
            session_id=session_id,
            provider=provider,
            model=model,
            registry=registry,
            thinking=thinking,
            dry_run=dry_run,
            workspace=workspace,
            install_dir=install_dir,
            project_root=project_root,
            api_hosts=api_hosts,
        )

    def print_status_bar(
        self,
        *,
        model: str,
        thinking: bool,
        dry_run: bool,
        steps: int = 0,
    ) -> None:
        if self.plain:
            return
        model_short = model.split("/")[-1] if "/" in model else model
        if len(model_short) > 20:
            model_short = model_short[:17] + "..."
        think = "on" if thinking else "off"
        dry = "on" if dry_run else "off"
        assert self._console is not None
        self._console.print(
            f" [bold {_C_GOLD}]⚕[/] [bold {_C_GOLD}]{model_short}[/] "
            f"[{_C_DIM}]│ think {think} │ dry-run {dry} │ {steps} steps[/]"
        )

    def print_turn_footer(
        self,
        *,
        model: str,
        thinking: bool,
        dry_run: bool,
        steps: int = 0,
    ) -> None:
        if self.plain:
            self._write("\n")
            return
        self.print_status_bar(model=model, thinking=thinking, dry_run=dry_run, steps=steps)
        self._print_rule()

    def print_prompt(self) -> None:
        if self.plain:
            self._write("❯ ")
            return
        assert self._console is not None
        self._console.print(f"[bold {_C_GOLD}]❯[/] ", end="")

    def print_user(self, text: str) -> None:
        if self.plain:
            self._write(f"\n● {text}\n")
            return
        assert self._console is not None
        self._console.print(f"\n[bold {_C_GOLD}]●[/] {text}")

    def print_assistant(self, text: str) -> None:
        if self.plain:
            self._write(f"\n{text}\n\n")
            return

        from rich.markdown import Markdown
        from rich.panel import Panel

        assert self._console is not None
        self._console.print(
            Panel(
                Markdown(text),
                title=f"[bold {_C_GOLD}]⚕ GeeGoo[/]",
                border_style=_C_AMBER,
                padding=(0, 1),
            )
        )

    def print_help(self, text: str) -> None:
        if self.plain:
            self._write(text)
            return
        assert self._console is not None
        self._console.print(f"[{_C_DIM}]{text.strip()}[/]")

    def print_info(self, text: str) -> None:
        if self.plain:
            self._write(text + ("\n" if not text.endswith("\n") else ""))
            return
        assert self._console is not None
        self._console.print(f"[{_C_DIM}]{text}[/]")

    def print_error(self, text: str) -> None:
        if self.plain:
            self._write(f"✗ {text}\n")
            return
        assert self._console is not None
        self._console.print(f"[bold {_C_ERR}]✗ {text}[/]")

    def _tool_key(self, data: dict[str, Any]) -> str:
        return f"{data.get('step', 0)}:{data.get('name', '?')}"

    def emit_progress(self, event: str, data: dict[str, Any]) -> None:
        if event == "turn_start":
            if self.plain:
                self._write("────────────────\nInitializing agent...\n")
                return
            self._print_rule()
            assert self._console is not None
            self._console.print(f"[{_C_DIM}]Initializing agent...[/]")
            return

        if event == "round_start":
            if self.plain:
                self._write(f"⋯ step {data.get('step', '?')}\n")
            return

        if event == "llm_plan":
            if self.plain:
                reasoning = str(data.get("reasoning") or "").strip()
                content = str(data.get("content") or "").strip()
                tool_names = data.get("tool_names") or []
                if reasoning:
                    self._write(f"  [思考] {reasoning[:500]}\n")
                if content:
                    self._write(f"  [计划] {content[:300]}\n")
                if tool_names:
                    self._write(f"  [决策] 调用: {', '.join(tool_names)}\n")
            return

        if event == "llm_tools":
            if self.plain:
                names = data.get("tool_names") or []
                self._write(f"  ⋯ 计划调用: {', '.join(names)}\n")
            return

        if event == "tool_start":
            name = data.get("name", "?")
            display_name = "session_search" if name == "recall" else name
            key = self._tool_key(data)
            self._tool_starts[key] = time.monotonic()
            self._tool_args[key] = _fmt_args_compact(data.get("arguments") or {})
            if self.plain:
                args = _fmt_args(data.get("arguments") or {})
                self._write(f"  → {name}({args})\n")
                return
            emoji = _tool_emoji(name)
            assert self._console is not None
            self._console.print(f"  ┊ {emoji} [{_C_DIM}]preparing {display_name}…[/]")
            return

        if event == "tool_done":
            name = data.get("name", "?")
            status = data.get("status", "?")
            key = self._tool_key(data)
            started = self._tool_starts.pop(key, time.monotonic())
            args = self._tool_args.pop(key, "") or _fmt_args_compact(data.get("arguments") or {})
            duration = max(0.0, time.monotonic() - started)
            ok = status == "ok"
            if self.plain:
                mark = "✓" if ok else "✗"
                summary = str(data.get("summary", ""))[:160]
                self._write(f"  {mark} {name} [{status}] {summary}\n")
                return

            emoji = _tool_emoji(name)
            color = _C_GOLD if ok else _C_ERR
            args_part = f"  {args}" if args else ""
            label = "recall" if name == "recall" else name
            assert self._console is not None
            self._console.print(
                f"  ┊ {emoji} [{color}]{label:<22}[/][{_C_DIM}]{args_part}  {duration:.1f}s[/]"
            )
            return

        if event == "reply_start":
            return

        if event == "error":
            self.print_error(str(data.get("message", "error")))

    def make_progress_callback(self) -> ProgressFn:
        return self.emit_progress
