#!/usr/bin/env python3
"""Tests for diagnose_macro_pool workflow pack/parse (crypto fixture, no production)."""

from __future__ import annotations

import json
import unittest

from macro_pool_diagnose_skill import (
    PoolSkillPackError,
    build_macro_pool_skill_body,
    parse_pool_choice_distribution,
    pool_jev_from_diagnose_macro_pool_skill_response,
    validate_pool_skill_pack,
)

SAMPLE_CHOICES = [
    {"label": "看多", "probability": 0.72},
    {"label": "看空", "probability": 0.18},
    {"label": "不明确", "probability": 0.1},
]


CRYPTO_FIXTURE_ROWS = [
    {
        "symbol": "BTC-USD.CRYPTO",
        "success": True,
        "jev_direction": "long",
        "jev_reason": "BTC 测试理由：结构偏多且量能配合，足够长度用于池 skill 校验。",
        "confidence": 0.82,
    },
    {
        "symbol": "ETH-USD.CRYPTO",
        "success": True,
        "jev_direction": "short",
        "jev_reason": "ETH 测试理由：动能转弱与指数压制，足够长度用于池 skill 校验。",
        "confidence": 0.71,
    },
]

CRYPTO_EXPECTED = ["BTC-USD.CRYPTO", "ETH-USD.CRYPTO"]

RID = {
    "BTC-USD.CRYPTO": "sym:2026-11-20:crypto:BTC-USD.CRYPTO:batch-demo",
    "ETH-USD.CRYPTO": "sym:2026-11-20:crypto:ETH-USD.CRYPTO:batch-demo",
}


class TestMacroPoolDiagnoseSkillPack(unittest.TestCase):
    def test_crypto_fixture_body(self) -> None:
        body = build_macro_pool_skill_body(
            batch_date="2026-11-20",
            pool_workflow="crypto",
            source_batch_id="batch-demo",
            title="加密",
            expected_symbols=CRYPTO_EXPECTED,
            symbol_rows=CRYPTO_FIXTURE_ROWS,
            symbol_result_ids=RID,
        )
        self.assertEqual(body["skill"], "diagnose_macro_pool")
        mp = body["macro_pool"]
        self.assertEqual(mp["date"], "2026-11-20")
        self.assertEqual(set(mp["expected_symbols"]), set(CRYPTO_EXPECTED))
        self.assertEqual(len(mp["symbols"]), 2)
        syms = {s["symbol"] for s in mp["symbols"]}
        self.assertEqual(syms, set(CRYPTO_EXPECTED))

    def test_rejects_only_btc_when_eth_expected(self) -> None:
        with self.assertRaises(PoolSkillPackError) as ctx:
            validate_pool_skill_pack(
                batch_date="2026-11-20",
                pool_workflow="crypto",
                source_batch_id="batch-demo",
                expected_symbols=CRYPTO_EXPECTED,
                symbol_rows=CRYPTO_FIXTURE_ROWS[:1],
                symbol_result_ids=RID,
                min_reason_len=20,
            )
        self.assertIn("ETH", str(ctx.exception))

    def test_rejects_duplicate_symbol(self) -> None:
        rows = [CRYPTO_FIXTURE_ROWS[0], {**CRYPTO_FIXTURE_ROWS[0]}]
        with self.assertRaises(PoolSkillPackError):
            validate_pool_skill_pack(
                batch_date="2026-11-20",
                pool_workflow="crypto",
                source_batch_id="batch-demo",
                expected_symbols=["BTC-USD.CRYPTO"],
                symbol_rows=rows,
                symbol_result_ids=RID,
                min_reason_len=20,
            )

    def test_rejects_forged_result_id_gap(self) -> None:
        with self.assertRaises(PoolSkillPackError):
            validate_pool_skill_pack(
                batch_date="2026-11-20",
                pool_workflow="crypto",
                source_batch_id="batch-demo",
                expected_symbols=CRYPTO_EXPECTED,
                symbol_rows=[
                    {**CRYPTO_FIXTURE_ROWS[0], "result_id": ""},
                    CRYPTO_FIXTURE_ROWS[1],
                ],
                symbol_result_ids={},
                min_reason_len=20,
            )

    def test_rejects_invalid_batch_date(self) -> None:
        with self.assertRaises(PoolSkillPackError):
            build_macro_pool_skill_body(
                batch_date="2026/11/20",
                pool_workflow="crypto",
                source_batch_id="batch-demo",
                title="加密",
                expected_symbols=CRYPTO_EXPECTED,
                symbol_rows=CRYPTO_FIXTURE_ROWS,
                symbol_result_ids=RID,
            )

    def test_rejects_wrong_pool_workflow_symbol_suffix(self) -> None:
        """Mixed pool: crypto expected set inside cn_a workflow pack must fail workflow field validation."""
        with self.assertRaises(PoolSkillPackError):
            validate_pool_skill_pack(
                batch_date="2026-11-20",
                pool_workflow="not_a_pool",
                source_batch_id="batch-demo",
                expected_symbols=CRYPTO_EXPECTED,
                symbol_rows=CRYPTO_FIXTURE_ROWS,
                symbol_result_ids=RID,
                min_reason_len=20,
            )

    def test_parse_success_response(self) -> None:
        resp = {
            "status": "ok",
            "session_id": "sess-pool-1",
            "skill": "diagnose_macro_pool",
            "result": {
                "jev_direction": "long",
                "jev_reason": "## 整体结论\n看多，置信 0.72。\n1. BTC 强\n2. 综合权衡",
                "confidence": 0.72,
                "conflict": True,
                "choices": SAMPLE_CHOICES,
                "detail_ref": "diagnose_macro_pool",
                "input_refs": [
                    {"symbol": "BTC-USD.CRYPTO", "result_id": RID["BTC-USD.CRYPTO"]},
                ],
            },
        }
        out = pool_jev_from_diagnose_macro_pool_skill_response(resp, min_reason_len=20)
        self.assertTrue(out.get("ok"))
        self.assertEqual(out.get("jev_direction"), "long")
        self.assertEqual(out.get("confidence"), 0.72)
        self.assertTrue(out.get("conflict"))
        self.assertEqual(out.get("choices"), SAMPLE_CHOICES)

    def test_parse_choices_rejects_bad_shape(self) -> None:
        self.assertIsNone(parse_pool_choice_distribution(None))
        self.assertIsNone(parse_pool_choice_distribution([{"label": "up", "probability": 0.5}]))
        self.assertIsNone(parse_pool_choice_distribution("not-a-list"))

    def test_parse_success_requires_choices(self) -> None:
        resp = {
            "status": "ok",
            "session_id": "s2",
            "result": {
                "jev_direction": "long",
                "jev_reason": "## 整体结论\n" + "x" * 40,
                "confidence": 0.7,
            },
        }
        out = pool_jev_from_diagnose_macro_pool_skill_response(resp, min_reason_len=20)
        self.assertFalse(out.get("ok"))
        self.assertIn("choices", str(out.get("error")))

    def test_skill_failure_not_neutral(self) -> None:
        resp = {"status": "failed", "error": "jev down", "session_id": "s1"}
        out = pool_jev_from_diagnose_macro_pool_skill_response(resp)
        self.assertFalse(out.get("ok"))
        self.assertNotIn("jev_direction", out)

    def test_source_batch_id_required_at_pack(self) -> None:
        with self.assertRaises(PoolSkillPackError):
            build_macro_pool_skill_body(
                batch_date="2026-11-20",
                pool_workflow="crypto",
                source_batch_id="",
                title="加密",
                expected_symbols=CRYPTO_EXPECTED,
                symbol_rows=CRYPTO_FIXTURE_ROWS,
                symbol_result_ids=RID,
            )

    def test_minimal_io_json_documented(self) -> None:
        body = build_macro_pool_skill_body(
            batch_date="2026-11-20",
            pool_workflow="crypto",
            source_batch_id="batch-demo",
            title="加密",
            expected_symbols=CRYPTO_EXPECTED[:1],
            symbol_rows=CRYPTO_FIXTURE_ROWS[:1],
            symbol_result_ids=RID,
        )
        self.assertTrue(json.dumps(body, ensure_ascii=False))


class TestPoolSkillPackBondExpectedSet(unittest.TestCase):
    """bond/pool-jev-only: expected set vs rows mismatch fails in workflow without silent partial synthesis."""

    def test_expected_set_mismatch_before_skill_http(self) -> None:
        rows = [CRYPTO_FIXTURE_ROWS[0]]
        with self.assertRaises(PoolSkillPackError):
            build_macro_pool_skill_body(
                batch_date="2026-11-20",
                pool_workflow="crypto",
                source_batch_id="batch-run-pool-only",
                title="加密",
                expected_symbols=CRYPTO_EXPECTED,
                symbol_rows=rows,
                symbol_result_ids=RID,
            )


if __name__ == "__main__":
    unittest.main()