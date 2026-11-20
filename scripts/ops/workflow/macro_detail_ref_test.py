#!/usr/bin/env python3
"""detail_ref selection for macro batch (start_idx semantics)."""

from __future__ import annotations

import unittest

from run_macro_monitor_lab_batch import (
    _detail_ref_from_signal_eval_episode,
    pick_detail_ref,
)


class TestDetailRef(unittest.TestCase):
    def test_pick_prefers_lab_default(self) -> None:
        ref = pick_detail_ref(
            {
                "lab": {
                    "lab_default_detail_ref": "buy:190",
                    "buy:7": {"start_idx": 7},
                }
            }
        )
        self.assertEqual(ref, "buy:190")

    def test_pick_summaries_start_idx_not_key(self) -> None:
        ref = pick_detail_ref(
            {
                "lab": {
                    "lab_episode_summaries": {
                        "buy:7": {"start_idx": 190},
                    }
                }
            }
        )
        self.assertEqual(ref, "buy:190")

    def test_signal_eval_start_idx(self) -> None:
        self.assertEqual(
            _detail_ref_from_signal_eval_episode({"side": "sell", "start_idx": 43}),
            "sell:43",
        )

    def test_no_episode_key_fallback(self) -> None:
        ref = pick_detail_ref({"lab": {"buy:7": {"foo": 1}}, "signal_eval": {}})
        self.assertEqual(ref, "")


if __name__ == "__main__":
    unittest.main()