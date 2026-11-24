#!/usr/bin/env python3
from __future__ import annotations

import unittest

from macro_lab_batch_health import (
    batch_item_from_diagnose_skill_response,
    build_diagnose_episode_summary_from_detail,
    build_diagnose_episode_summary_from_skill_result,
    direction_skill_to_ledger,
    item_refs_complete,
    lab_detail_four_lane_failed,
    diagnose_episode_summary_failed,
)


class TestMacroLabBatchHealth(unittest.TestCase):
    def test_summary_ok(self) -> None:
        detail = {
            "ref": "buy:1",
            "jev_unified": {"ok": True, "choice": "long"},
            "technical_analysis_jev": {"ok": True},
            "index_trend_jev": {"ok": True},
            "price_analysis_jev": {"ok": True},
            "kline_jev": {"ok": True},
        }
        summary = build_diagnose_episode_summary_from_detail(detail, "buy:1")
        assert summary is not None
        failed, _ = diagnose_episode_summary_failed(summary)
        self.assertFalse(failed)

    def test_summary_timeout_lane_fails(self) -> None:
        detail = {
            "ref": "buy:1",
            "jev_unified": {"ok": True, "choice": "neutral"},
            "kline_jev": {"ok": False, "error": "context deadline exceeded"},
        }
        summary = build_diagnose_episode_summary_from_detail(detail, "buy:1")
        failed, reason = diagnose_episode_summary_failed(summary)
        self.assertTrue(failed)
        self.assertIn("kline_jev", reason)

    def test_trading_day_degraded(self) -> None:
        detail = {
            "jev_unified": {"ok": False, "error": "macro_trading_day_check_failed"},
            "technical_analysis_jev": {"ok": True},
            "index_trend_jev": {"ok": True},
            "price_analysis_jev": {"ok": True},
            "kline_jev": {"ok": True},
            "degraded": ["macro_trading_day_check_failed"],
        }
        failed, reason = lab_detail_four_lane_failed(detail)
        self.assertTrue(failed)
        self.assertIn("jev_unified", reason)

    def test_item_refs_complete(self) -> None:
        ok, _ = item_refs_complete("x" * 20, "buy:9", "s", "r", "long", min_reason_len=20)
        self.assertTrue(ok)
        ok2, msg = item_refs_complete("short", "buy:9", "s", "r", "long", min_reason_len=20)
        self.assertFalse(ok2)
        self.assertIn("jev_reason", msg)

    def test_item_refs_complete_skill_detail_ref(self) -> None:
        reason = "宏观 skill 解读：" + "x" * 20
        ok, msg = item_refs_complete(
            reason, "diagnose_macro", "sess-1", "sess-1", "long", min_reason_len=20
        )
        self.assertTrue(ok, msg)
        ok2, msg2 = item_refs_complete(reason, "diagnose_macro", "", "sess-1", "long", min_reason_len=20)
        self.assertFalse(ok2)
        self.assertIn("session", msg2)

    def test_direction_skill_to_ledger(self) -> None:
        self.assertEqual(direction_skill_to_ledger({"direction": "bull", "confidence": 0.8})[0], "long")
        self.assertEqual(direction_skill_to_ledger({"direction": "bear"})[0], "short")
        self.assertEqual(direction_skill_to_ledger({"direction": "unclear", "ok": True})[0], "neutral")

    def test_build_diagnose_episode_summary_from_skill_result(self) -> None:
        result = {
            "monitor_session_close": "2026-01-01T08:00:00Z",
            "jev_unified": {"ok": True, "direction": "bull", "recommended_reason": "r" * 25},
            "technical_analysis_jev": {"ok": True},
            "index_trend_jev": {"ok": True},
            "price_analysis_jev": {"ok": True},
            "kline_jev": {"ok": True},
        }
        summary = build_diagnose_episode_summary_from_skill_result(result, "diagnose_macro")
        assert summary is not None
        failed, _ = diagnose_episode_summary_failed(summary)
        self.assertFalse(failed)
        self.assertEqual(summary.get("detail_ref"), "diagnose_macro")

    def test_batch_item_from_diagnose_skill_response(self) -> None:
        lane_ok = {"ok": True, "direction": "bull", "recommended_reason": "看多主线" + "x" * 20}
        response = {
            "status": "ok",
            "session_id": "sk-sess-abc",
            "skill": "diagnose_macro",
            "result": {
                "jev_unified": {**lane_ok, "confidence": 0.85},
                "technical_analysis_jev": {"ok": True},
                "index_trend_jev": {"ok": True},
                "price_analysis_jev": {"ok": True},
                "kline_jev": {"ok": True},
            },
        }
        out = batch_item_from_diagnose_skill_response(response, min_reason_len=20)
        self.assertTrue(out.get("ok"))
        self.assertEqual(out.get("direction"), "long")
        self.assertEqual(out.get("detail_ref"), "diagnose_macro")
        self.assertEqual(out.get("session_id"), "sk-sess-abc")
        self.assertEqual(out.get("run_id"), "sk-sess-abc")
        self.assertIn("diagnose_episode_summary", out)


if __name__ == "__main__":
    unittest.main()