"""Hermes-style welcome banner for ``geegoo chat``."""

from __future__ import annotations

import os
import shutil
import subprocess
from pathlib import Path
from typing import TYPE_CHECKING, Any

from geegoo_agent import __version__
from geegoo_agent.paths import default_install_dir
from geegoo_agent.tools.types import ToolCategory

if TYPE_CHECKING:
    from rich.console import Console

    from geegoo_agent.tools.registry import ToolRegistry

# Palette aligned with chat_ui.py (gold + finance teal)
_C_GOLD = "#FFD700"
_C_AMBER = "#FFBF00"
_C_ACCENT = "#FFD700"
_C_DIM = "#9CA3AF"
_C_BORDER = "#CD7F32"
_C_TEXT = "#E5E7EB"

_CATEGORY_LABELS: dict[ToolCategory, str] = {
    ToolCategory.PERCEPTION: "perceive",
    ToolCategory.ANALYSIS: "analyze",
    ToolCategory.DECISION: "decide",
    ToolCategory.ACTION: "act",
    ToolCategory.META: "meta",
}

_CATEGORY_ORDER = (
    ToolCategory.PERCEPTION,
    ToolCategory.ANALYSIS,
    ToolCategory.DECISION,
    ToolCategory.ACTION,
    ToolCategory.META,
)

GEEGOO_HERO = f"""[bold {_C_ACCENT}]    ╭──╮      ╭─╮   ╭──╮[/]
[bold {_C_GOLD}]   ╭╯  ╰╮    ╭╯ ╰╮ ╭╯  ╰╮[/]
[bold {_C_ACCENT}]  ╭╯    ╰╮  │   │╭╯    ╰╮[/]
[bold {_C_GOLD}] ╭╯      ╰╮ │   ││      │[/]
[bold {_C_ACCENT}] │   ▲    │ │ ▲ ││   ▼   │[/]
[bold {_C_GOLD}] │  ╱│╲   │ │╱ │╲││  ╱│╲  │[/]
[bold {_C_ACCENT}] ╰╮      ╭╯ ╰╮   ╭╯╰╮      ╭╯[/]
[bold {_C_GOLD}]  ╰╮    ╭╯   ╰╮ ╭╯  ╰╮    ╭╯[/]
[bold {_C_ACCENT}]   ╰╮  ╭╯     ╰─╯    ╰╮  ╭╯[/]
[bold {_C_GOLD}]    ╰──╯              ╰──╯[/]"""

GEEGOO_WIDE_LOGO = """[bold {_accent}]██████╗ [/][bold {_gold}]██╗[/][bold {_accent}] ██████╗ [/][bold {_gold}] ██████╗[/]
[bold {_accent}]██╔════╝[/][bold {_gold}] ██║[/][bold {_accent}]██╔════╝ [/][bold {_gold}]██╔═══██╗[/]
[bold {_accent}]██║  ███╗[/][bold {_gold}]██║[/][bold {_accent}]██║  ███╗[/][bold {_gold}]██║   ██║[/]
[bold {_accent}]██║   ██║[/][bold {_gold}]██║[/][bold {_accent}]██║   ██║[/][bold {_gold}]██║   ██║[/]
[bold {_accent}]╚██████╔╝[/][bold {_gold}]██║[/][bold {_accent}]╚██████╔╝[/][bold {_gold}]╚██████╔╝[/]
[bold {_accent}] ╚═════╝ [/][bold {_gold}]╚═╝[/][bold {_accent}] ╚═════╝ [/][bold {_gold}] ╚═════╝[/]""".format(
    _accent=_C_ACCENT,
    _gold=_C_GOLD,
)


def resolve_upstream_rev(install_dir: Path | None = None) -> str:
    env = os.environ.get("GEEGOO_REVISION", "").strip()
    if env:
        return env[:12]
    root = install_dir or default_install_dir()
    git_dir = root / ".git"
    if not git_dir.is_dir():
        return "local"
    try:
        out = subprocess.run(
            ["git", "-C", str(root), "rev-parse", "--short", "HEAD"],
            capture_output=True,
            text=True,
            timeout=2,
            check=False,
        )
        rev = (out.stdout or "").strip()
        return rev or "local"
    except (OSError, subprocess.TimeoutExpired):
        return "local"


def format_version_label(*, install_dir: Path | None = None) -> str:
    rev = resolve_upstream_rev(install_dir)
    return f"GeeGoo Agent v{__version__} · upstream {rev}"


def group_tools_by_category(registry: ToolRegistry) -> dict[str, list[str]]:
    buckets: dict[str, list[str]] = {}
    for name in registry.list_names():
        tool = registry.get(name)
        label = _CATEGORY_LABELS.get(tool.category, str(tool.category))
        buckets.setdefault(label, []).append(name)
    for names in buckets.values():
        names.sort()
    return buckets


def scan_skills(project_root: Path | None) -> dict[str, list[str]]:
    if project_root is None:
        return {}
    skills_dir = project_root / "skills"
    if not skills_dir.is_dir():
        return {}
    found: dict[str, list[str]] = {}
    for skill_md in sorted(skills_dir.rglob("SKILL.md")):
        rel = skill_md.parent.relative_to(skills_dir)
        parts = rel.parts
        if not parts:
            continue
        if parts[0] == "bundled" and len(parts) > 1:
            category = "bundled"
            name = parts[1]
        elif len(parts) == 1:
            category = "workflows"
            name = parts[0]
        else:
            category = parts[0]
            name = parts[-1]
        found.setdefault(category, []).append(name)
    for names in found.values():
        names.sort()
    return found


def _truncate_tool_list(names: list[str], *, max_len: int = 44) -> str:
    if not names:
        return ""
    parts: list[str] = []
    length = 0
    for name in names:
        extra = len(name) + (2 if parts else 0)
        if length + extra > max_len:
            parts.append("...")
            break
        parts.append(name)
        length += extra
    return ", ".join(parts)


def _short_url_host(url: str) -> str:
    text = (url or "").strip()
    if not text:
        return "—"
    for prefix in ("https://", "http://"):
        if text.startswith(prefix):
            text = text[len(prefix) :]
    return text.rstrip("/")


def build_plain_banner(
    *,
    session_id: str,
    provider: str,
    model: str,
    registry: ToolRegistry,
    thinking: bool,
    dry_run: bool,
    workspace: Path | None,
    install_dir: Path | None,
    api_hosts: dict[str, str] | None = None,
    project_root: Path | None = None,
) -> str:
    lines: list[str] = []
    lines.append(format_version_label(install_dir=install_dir))
    lines.append(f"Model: {provider} / {model}")
    lines.append(f"Session: {session_id}")
    if workspace:
        lines.append(f"CWD: {workspace}")
    think = "on" if thinking else "off"
    dry = "on" if dry_run else "off"
    lines.append(f"Think: {think}  Dry-run: {dry}")
    lines.append("")
    lines.append("Available Tools:")
    groups = group_tools_by_category(registry)
    for cat in _CATEGORY_ORDER:
        label = _CATEGORY_LABELS[cat]
        names = groups.get(label, [])
        if names:
            lines.append(f"  {label}: {', '.join(names)}")
    if api_hosts:
        lines.append("")
        lines.append("APIs:")
        for key, host in api_hosts.items():
            lines.append(f"  {key}: {host}")
    skills = scan_skills(project_root)
    if skills:
        lines.append("")
        lines.append("Skills:")
        for category in sorted(skills):
            lines.append(f"  {category}: {', '.join(skills[category])}")
    total_tools = len(registry.list_names())
    total_skills = sum(len(v) for v in skills.values())
    lines.append("")
    lines.append(f"{total_tools} tools · {total_skills} skills · /help for commands")
    lines.append("")
    lines.append("Welcome to GeeGoo Agent! Type your message or /help for commands.")
    return "\n".join(lines) + "\n"


def build_welcome_banner(
    console: Console,
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
    """Print Hermes-style two-column welcome panel."""
    from rich.panel import Panel
    from rich.table import Table

    accent = _C_AMBER
    dim = _C_DIM
    text = _C_TEXT

    layout = Table.grid(padding=(0, 2))
    layout.add_column("left", justify="center", ratio=1)
    layout.add_column("right", justify="left", ratio=2)

    model_short = model.split("/")[-1] if "/" in model else model
    if len(model_short) > 28:
        model_short = model_short[:25] + "..."

    left_lines = ["", GEEGOO_HERO, ""]
    left_lines.append(f"[{accent}]{model_short}[/] [dim {dim}]·[/] [dim {dim}]{provider}[/]")
    think_label = "on" if thinking else "off"
    dry_label = "on" if dry_run else "off"
    left_lines.append(
        f"[dim {dim}]think {think_label}[/] [dim {dim}]·[/] "
        f"[dim {dim}]dry-run {dry_label}[/]"
    )
    if workspace:
        cwd = str(workspace)
        if len(cwd) > 36:
            cwd = "…" + cwd[-35:]
        left_lines.append(f"[dim {dim}]{cwd}[/]")
    left_lines.append(f"[dim {dim}]Session: {session_id}[/]")
    left_content = "\n".join(left_lines)

    right_lines: list[str] = [f"[bold {accent}]Available Tools[/]"]
    groups = group_tools_by_category(registry)
    display_cats = [_CATEGORY_LABELS[c] for c in _CATEGORY_ORDER if _CATEGORY_LABELS[c] in groups]
    remaining_cats = max(0, len(display_cats) - 7)
    for label in display_cats[:7]:
        names = groups[label]
        tools_str = _truncate_tool_list(names)
        right_lines.append(f"[dim {dim}]{label}:[/] [{text}]{tools_str}[/]")
    if remaining_cats:
        right_lines.append(f"[dim {dim}](and {remaining_cats} more categories...)[/]")

    if api_hosts:
        right_lines.append("")
        right_lines.append(f"[bold {accent}]APIs[/]")
        for key, host in api_hosts.items():
            right_lines.append(f"[dim {dim}]{key}[/] [{text}]({host})[/]")

    skills = scan_skills(project_root)
    total_skills = sum(len(v) for v in skills.values())
    right_lines.append("")
    right_lines.append(f"[bold {accent}]Available Skills[/]")
    if skills:
        for category in sorted(skills.keys())[:8]:
            names = skills[category]
            if len(names) > 6:
                skills_str = ", ".join(names[:6]) + f" +{len(names) - 6} more"
            else:
                skills_str = ", ".join(names)
            if len(skills_str) > 52:
                skills_str = skills_str[:49] + "..."
            right_lines.append(f"[dim {dim}]{category}:[/] [{text}]{skills_str}[/]")
        if len(skills) > 8:
            right_lines.append(f"[dim {dim}](and {len(skills) - 8} more categories...)[/]")
    else:
        right_lines.append(f"[dim {dim}]No skills in project[/]")

    total_tools = len(registry.list_names())
    summary = f"{total_tools} tools · {total_skills} skills · /help for commands"
    right_lines.append("")
    right_lines.append(f"[dim {dim}]{summary}[/]")
    right_content = "\n".join(right_lines)

    layout.add_row(left_content, right_content)
    title = format_version_label(install_dir=install_dir)
    panel = Panel(
        layout,
        title=f"[bold {_C_GOLD}]{title}[/]",
        border_style=_C_BORDER,
        padding=(0, 2),
    )

    console.print()
    term_width = shutil.get_terminal_size((120, 24)).columns
    if term_width >= 95:
        console.print(GEEGOO_WIDE_LOGO)
        console.print()
    console.print(panel)
    console.print()
    console.print(
        f"[{text}]Welcome to GeeGoo Agent![/] "
        f"[dim {dim}]Type your message or /help for commands.[/]"
    )
    console.print(
        f"[dim {dim}]✦ Tip: [/][dim {dim}]/think on[/][dim {dim}] shows DeepSeek reasoning; "
        f"[/][dim {dim}]/verbose off[/][dim {dim}] hides live steps.[/]"
    )
    console.print()


def api_hosts_from_config(config: Any) -> dict[str, str]:
    """Build short API host map from :class:`AppConfig`."""
    hosts: dict[str, str] = {}
    base = getattr(config, "base_url", "") or ""
    geegoo = getattr(config, "geegoo_url", "") or ""
    if base:
        hosts["market"] = _short_url_host(base)
    if geegoo:
        hosts["geegoo-bot"] = _short_url_host(geegoo)
    signal = getattr(config, "signal_base_url", "") or ""
    if signal:
        hosts["signal"] = _short_url_host(signal)
    return hosts
