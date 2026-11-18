#!/usr/bin/env python3
from __future__ import annotations

import unittest
from unittest.mock import patch

from macro_jev_orchestration import (
    MACRO_JEV_LEGACY_WEEKDAY_CRON_SHANGHAI,
    MACRO_JEV_WEEKDAY_CRON_SHANGHAI,
    expected_pool_macro_date,
    macro_batch_date_for_cron,
    pools_needing_rerun,
    previous_us_trading_day,
    resolve_run_dates,
)
from macro_jev_results_client import JEV_PIPELINE_WORKFLOWS

_D = "2026-11-18"  # Tuesday Shanghai


class TestMacroSchedule(unittest.TestCase):
    def test_confirmed_shanghai_cron_matrix(self) -> None:
        self.assertEqual(MACRO_JEV_WEEKDAY_CRON_SHANGHAI["us"], "30 5")
        self.assertEqual(MACRO_JEV_WEEKDAY_CRON_SHANGHAI["crypto"], "38 5")
        self.assertEqual(MACRO_JEV_WEEKDAY_CRON_SHANGHAI["cn_a"], "30 15")
        self.assertEqual(MACRO_JEV_WEEKDAY_CRON_SHANGHAI["overall"], "0 17")
        self.assertNotIn("America/New_York", str(MACRO_JEV_WEEKDAY_CRON_SHANGHAI.values()))

    def test_legacy_cron_docs_differ_from_current(self) -> None:
        self.assertEqual(MACRO_JEV_LEGACY_WEEKDAY_CRON_SHANGHAI["cn_a"], "0 7")
        self.assertEqual(MACRO_JEV_LEGACY_WEEKDAY_CRON_SHANGHAI["overall"], "10 9")
        self.assertNotEqual(
            MACRO_JEV_LEGACY_WEEKDAY_CRON_SHANGHAI["overall"],
            MACRO_JEV_WEEKDAY_CRON_SHANGHAI["overall"],
        )

    def test_previous_us_trading_day_tuesday(self) -> None:
        self.assertEqual(previous_us_trading_day(_D), "2026-11-17")

    def test_previous_us_trading_day_monday_shanghai(self) -> None:
        self.assertEqual(previous_us_trading_day("2026-11-16"), "2026-11-13")

    def test_macro_batch_dates_morning_and_afternoon(self) -> None:
        self.assertEqual(macro_batch_date_for_cron("us", _D), "2026-11-17")
        self.assertEqual(macro_batch_date_for_cron("crypto", _D), "2026-11-17")
        self.assertEqual(macro_batch_date_for_cron("bond_us", _D), "2026-11-17")
        self.assertEqual(macro_batch_date_for_cron("commodity_fx", _D), "2026-11-17")
        self.assertEqual(macro_batch_date_for_cron("volatility", _D), "2026-11-17")
        self.assertEqual(macro_batch_date_for_cron("cn_a", _D), _D)
        self.assertEqual(macro_batch_date_for_cron("bond_sh", _D), _D)
        self.assertEqual(macro_batch_date_for_cron("hk", _D), _D)
        self.assertEqual(macro_batch_date_for_cron("overall", _D), _D)

    def test_resolve_run_dates_auto_orchestration(self) -> None:
        orch, macro = resolve_run_dates("us", _D)
        self.assertEqual(orch, _D)
        self.assertEqual(macro, "2026-11-17")

    def test_expected_pool_macro_dates_for_overall_validation(self) -> None:
        self.assertEqual(expected_pool_macro_date("us", _D), "2026-11-17")
        self.assertEqual(expected_pool_macro_date("cn_a", _D), _D)
        self.assertEqual(expected_pool_macro_date("bond", _D), _D)

    def test_pools_needing_rerun(self) -> None:
        ledger = {wf: {"status": "success"} for wf in JEV_PIPELINE_WORKFLOWS}
        ledger["hk"] = {"status": "failed"}
        self.assertEqual(pools_needing_rerun(ledger), ["hk"])


class TestOverallRerun(unittest.TestCase):
    def test_refresh_reruns_missing_pool_before_overall(self) -> None:
        from run_macro_monitor_lab_batch import refresh_ledger_before_overall

        calls: list[str] = []

        def fake_run_batch(wf, *a, **k):
            calls.append(wf)
            return 0, None, None

        with patch("run_macro_monitor_lab_batch.run_batch", side_effect=fake_run_batch):
            with patch(
                "run_macro_monitor_lab_batch.load_pool_ledger",
                side_effect=[
                    {"us": {"status": "success"}, "hk": {"status": "failed"}},
                    {"us": {"status": "success"}, "hk": {"status": "success"}},
                ],
            ):
                with patch(
                    "run_macro_monitor_lab_batch.pools_needing_rerun",
                    return_value=["hk"],
                ):
                    refresh_ledger_before_overall("http://m", "key", "http://a", _D, False)
        self.assertEqual(calls, ["hk"])


class TestSymbolRetryDelayDefault(unittest.TestCase):
    def test_retry_delay_default_600_seconds(self) -> None:
        import run_macro_monitor_lab_batch as batch

        self.assertEqual(batch.RETRY_DELAY_SEC, 600)
        self.assertGreaterEqual(batch.MAX_DIAGNOSE_RETRIES, 1)


if __name__ == "__main__":
    unittest.main()