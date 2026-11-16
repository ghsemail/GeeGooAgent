#!/usr/bin/env python3
"""Tests for agent_chat_stream SSE parsing."""

from __future__ import annotations

import unittest

from agent_chat_stream import last_assistant_text, parse_sse_payload, session_turn_complete


class TestAgentChatStream(unittest.TestCase):
    def test_parse_sse_connected_and_turn_end(self) -> None:
        raw = (
            "event: connected\n"
            'data: {"session_id":"chat-abc"}\n\n'
            "event: progress\n"
            'data: {"schema_version":1}\n\n'
            "event: turn_end\n"
            'data: {"session_id":"chat-abc","failed":false,"assistant_text":"结论 long"}\n\n'
            "event: done\n"
            'data: {"session_id":"chat-abc"}\n\n'
        )
        sid, run_id, turn_end, events = parse_sse_payload(raw)
        self.assertEqual(sid, "chat-abc")
        self.assertEqual(turn_end.get("assistant_text"), "结论 long")
        self.assertIn("turn_end", events)

    def test_parse_partial_sse_after_incomplete_read(self) -> None:
        raw = (
            "event: connected\n"
            'data: {"session_id":"chat-partial"}\n\n'
            "event: tool_end\n"
            'data: {"tool_name":"x"}\n\n'
        )
        sid, _, turn_end, _ = parse_sse_payload(raw)
        self.assertEqual(sid, "chat-partial")
        self.assertEqual(turn_end, {})

    def test_session_turn_complete(self) -> None:
        st = {
            "busy": False,
            "messages": [{"role": "user", "content": "hi"}, {"role": "assistant", "content": "宏观偏空理由" * 2}],
        }
        self.assertTrue(session_turn_complete(st, min_reply_len=10))
        self.assertEqual(last_assistant_text(st), "宏观偏空理由" * 2)


if __name__ == "__main__":
    unittest.main()