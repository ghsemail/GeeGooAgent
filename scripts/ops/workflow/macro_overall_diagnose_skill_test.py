#!/usr/bin/env python3
"""Tests for diagnose_macro_overall pack/parse (single crypto, no production)."""

from __future__ import annotations

import unittest

from macro_overall_diagnose_skill import (
    OverallSkillPackError,
    build_macro_overall_skill_body,
    overall_jev_from_skill_response,
    parse_overall_choice_distribution,
    validate_overall_pools,
)

CRYPTO_REASON = (
    "BTC 空头信号一致性强。日线呈 LH+LL 下行结构并出现 BOS，EMA13/26 斜率、MACD 柱及 Force Index 均偏弱，"
    "ADX 与 SAR 也给出空头信号。ETH 同样由模型判定为空头占优：日线趋势汇总为空头占优（ADX、SAR 均为空，net=-2），"
    "且 EMA13/26 斜率均为负、MACD 柱与 Force Index fi13 均为负，显示下行压力。"
)


class TestMacroOverallDiagnoseSkill(unittest.TestCase):
    def test_single_crypto_body(self) -> None:
        body = build_macro_overall_skill_body(
            orchestration_date="2026-11-22",
            pools=[{
                "pool_workflow": "crypto",
                "jev_direction": "short",
                "jev_reason": CRYPTO_REASON,
            }],
        )
        self.assertEqual(body["skill"], "diagnose_macro_overall")
        mo = body["macro_overall"]
        self.assertEqual(mo["date"], "2026-11-22")
        self.assertEqual(len(mo["pools"]), 1)
        self.assertEqual(mo["pools"][0]["pool_workflow"], "crypto")

    def test_rejects_empty_pools(self) -> None:
        with self.assertRaises(OverallSkillPackError):
            validate_overall_pools([], min_reason_len=20)

    def test_rejects_duplicate_pool(self) -> None:
        pools = [
            {"pool_workflow": "crypto", "jev_direction": "short", "jev_reason": CRYPTO_REASON},
            {"pool_workflow": "crypto", "jev_direction": "long", "jev_reason": CRYPTO_REASON},
        ]
        with self.assertRaises(OverallSkillPackError) as ctx:
            validate_overall_pools(pools, min_reason_len=20)
        self.assertIn("duplicate", str(ctx.exception))

    def test_skill_response_maps_choices(self) -> None:
        raw = {
            "status": "ok",
            "session_id": "s-overall",
            "skill": "diagnose_macro_overall",
            "result": {
                "date": "2026-11-22",
                "detail_ref": "diagnose_macro_overall",
                "jev_direction": "short",
                "jev_reason": "## 整体结论\n偏空。\n" + ("x" * 30),
                "confidence": 0.62,
                "choices": [
                    {"label": "看多", "probability": 0.2},
                    {"label": "看空", "probability": 0.62},
                    {"label": "不明确", "probability": 0.18},
                ],
            },
        }
        syn = overall_jev_from_skill_response(raw)
        self.assertTrue(syn.get("ok"))
        self.assertEqual(syn.get("jev_direction"), "short")
        self.assertAlmostEqual(syn.get("confidence"), 0.62)
        self.assertEqual(syn.get("detail_ref"), "diagnose_macro_overall")
        self.assertNotIn("conflict", syn)

    def test_skill_failure_no_fake_success(self) -> None:
        syn = overall_jev_from_skill_response({"status": "failed", "error": "jev down"})
        self.assertFalse(syn.get("ok"))
        self.assertNotIn("jev_direction", syn)

    def test_parse_choices_rejects_invalid(self) -> None:
        self.assertIsNone(parse_overall_choice_distribution([{"label": "bad", "probability": 0.5}]))