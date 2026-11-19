#!/usr/bin/env python3
from __future__ import annotations

import unittest

from macro_lab_batch_health import (
    build_lab_episode_summary_from_detail,
    item_refs_complete,
    lab_detail_four_lane_failed,
    lab_episode_summary_failed,
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
        summary = build_lab_episode_summary_from_detail(detail, "buy:1")
        assert summary is not None
        failed, _ = lab_episode_summary_failed(summary)
        self.assertFalse(failed)

    def test_summary_timeout_lane_fails(self) -> None:
        detail = {
            "ref": "buy:1",
            "jev_unified": {"ok": True, "choice": "neutral"},
            "kline_jev": {"ok": False, "error": "context deadline exceeded"},
        }
        summary = build_lab_episode_summary_from_detail(detail, "buy:1")
        failed, reason = lab_episode_summary_failed(summary)
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


if __name__ == "__main__":
    unittest.main()