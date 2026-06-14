"""Unit tests for geegoo chat REPL."""

from __future__ import annotations

import io
from unittest.mock import MagicMock, patch

import pytest

import json

from geegoo_agent.runtime.chat_repl import ChatRepl
from geegoo_agent.runtime.chat_session import ChatSession, ChatSessionStore
from geegoo_agent.runtime.react_loop import ChatTurnResult
from geegoo_agent.infra.state_store import FileStateStore


@pytest.mark.unit
def test_chat_repl_slash_help_and_exit(tmp_path, sample_config) -> None:
    store = FileStateStore(tmp_path)
    session = ChatSession(id="chat-1")
    session_store = ChatSessionStore(store)
    session_store.save(session)

    app = MagicMock()
    app.state_store = store
    app.config = sample_config
    app.llm_gateway = MagicMock()
    app.executor = MagicMock()
    app.event_bus.history = []

    repl = ChatRepl(
        app=app,
        session=session,
        session_store=session_store,
        registry=MagicMock(),
        loop=MagicMock(),
        stdin=io.StringIO("/help\n/exit\n"),
        stdout=io.StringIO(),
    )

    code = repl.run()
    assert code == 0
    out = repl.stdout.getvalue()
    assert "/trace" in out
    assert "再见" in repl.stdout.getvalue() or "会话已保存" in out


@pytest.mark.unit
@patch("geegoo_agent.runtime.chat_repl.ReActLoop.run_turn")
def test_chat_repl_user_message_calls_loop(mock_turn, tmp_path, sample_config) -> None:
    store = FileStateStore(tmp_path)
    session = ChatSession(id="chat-2")
    session_store = ChatSessionStore(store)

    mock_turn.return_value = ChatTurnResult(
        assistant_text="你好",
        step_records=[],
    )

    app = MagicMock()
    app.state_store = store
    app.config = sample_config
    app.llm_gateway = MagicMock()
    app.executor = MagicMock()
    app.event_bus.history = []
    app._tool_context.return_value = MagicMock(dry_run=False)

    repl = ChatRepl(
        app=app,
        session=session,
        session_store=session_store,
        registry=MagicMock(list_names=lambda: [], schemas=lambda **_: []),
        loop=MagicMock(run_turn=mock_turn),
        stdin=io.StringIO("分析一下腾讯\n/exit\n"),
        stdout=io.StringIO(),
    )

    repl.run()
    mock_turn.assert_called_once()
    assert "你好" in repl.stdout.getvalue()


@pytest.mark.unit
def test_chat_repl_model_list_and_switch(tmp_path, sample_config) -> None:
    config_path = tmp_path / "config.json"
    payload = sample_config.model_dump(mode="json")
    payload["llm"] = {"provider": "deepseek", "token_key": "sk-test", "model": ""}
    config_path.write_text(json.dumps(payload), encoding="utf-8")

    store = FileStateStore(tmp_path)
    session = ChatSession(id="chat-model")
    session_store = ChatSessionStore(store)

    app = MagicMock()
    app.state_store = store
    app.config = sample_config.model_copy(
        update={"llm": sample_config.llm.model_copy(update={"provider": "deepseek", "model": ""})}
    )
    app.secrets = MagicMock()
    app.llm_gateway = MagicMock()
    app.set_llm_model = MagicMock(return_value="deepseek-v4-pro")
    app.executor = MagicMock()
    app.event_bus.history = []

    loop = MagicMock()
    repl = ChatRepl(
        app=app,
        session=session,
        session_store=session_store,
        registry=MagicMock(),
        loop=loop,
        config_path=config_path,
        stdin=io.StringIO("/model\n/model 2\n/exit\n"),
        stdout=io.StringIO(),
    )

    repl.run()
    out = repl.stdout.getvalue()
    assert "deepseek-v4-flash" in out
    assert "deepseek-v4-pro" in out
    app.set_llm_model.assert_called_once_with("deepseek-v4-pro")
    saved = json.loads(config_path.read_text(encoding="utf-8"))
    assert saved["llm"]["model"] == "deepseek-v4-pro"
