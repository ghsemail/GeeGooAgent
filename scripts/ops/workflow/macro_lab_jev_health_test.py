#!/usr/bin/env python3
import unittest

from macro_lab_jev_health import lab_detail_four_lane_failed, lab_episode_summary_failed


class MacroLabJevHealthTest(unittest.TestCase):
    def test_happy_path(self) -> None:
        summary = {
            "jev_unified": {"ok": True},
            "technical_analysis_jev": {"ok": True},
            "index_trend_jev": {"ok": True},
            "price_analysis_jev": {"ok": True},
            "kline_jev": {"ok": True},
        }
        failed, _ = lab_episode_summary_failed(summary)
        self.assertFalse(failed)

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


if __name__ == "__main__":
    unittest.main()