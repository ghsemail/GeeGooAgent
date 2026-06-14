"""Interactive line input for ``geegoo chat`` (backspace, arrows, Unicode)."""

from __future__ import annotations

import sys
from typing import TextIO

from geegoo_agent.runtime.chat_ui import use_plain_ui

_PROMPT = "❯ "


def read_chat_line(
    *,
    plain: bool | None = None,
    stdin: TextIO | None = None,
    stdout: TextIO | None = None,
) -> str:
    """Read one user line with proper terminal editing when available."""
    in_stream = stdin or sys.stdin
    out_stream = stdout or sys.stdout
    is_plain = use_plain_ui(out_stream) if plain is None else plain

    if is_plain or not in_stream.isatty():
        out_stream.write(_PROMPT)
        out_stream.flush()
        return in_stream.readline().rstrip("\n")

    try:
        from prompt_toolkit import PromptSession
        from prompt_toolkit.input import create_input
        from prompt_toolkit.output import create_output
        from prompt_toolkit.shortcuts import CompleteStyle
        from prompt_toolkit.styles import Style

        from geegoo_agent.runtime.chat_commands import slash_command_completer
    except ImportError:
        return _readline_fallback(in_stream, out_stream)

    style = Style.from_dict(
        {
            "": "#FFF8DC",
            "prompt": "bold #FFD700",
            "cursor": "bold #FFBF00",
        }
    )
    session = PromptSession(
        input=create_input(in_stream),
        output=create_output(out_stream),
        style=style,
        completer=slash_command_completer(),
        complete_while_typing=True,
        complete_style=CompleteStyle.MULTI_COLUMN,
    )
    try:
        return session.prompt([("class:prompt", _PROMPT)]).rstrip("\n")
    except EOFError:
        raise
    except KeyboardInterrupt:
        raise


def _readline_fallback(in_stream: TextIO, out_stream: TextIO) -> str:
    try:
        import readline  # noqa: F401  # enables line editing for input()
    except ImportError:
        pass
    out_stream.write(_PROMPT)
    out_stream.flush()
    try:
        return input("").rstrip("\n")
    except EOFError:
        raise
