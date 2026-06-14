"""Unit tests for chat line input."""

from __future__ import annotations

import io

import pytest

from geegoo_agent.runtime.chat_input import read_chat_line


@pytest.mark.unit
def test_read_chat_line_plain_mode() -> None:
    stdin = io.StringIO("hello\n")
    stdout = io.StringIO()
    line = read_chat_line(plain=True, stdin=stdin, stdout=stdout)
    assert line == "hello"
    assert "❯" in stdout.getvalue()
