#!/usr/bin/env python3
"""Tests for diagnose_macro_pool workflow pack/parse (crypto fixture, no production)."""

from __future__ import annotations

import json
import unittest

from macro_pool_diagnose_skill import (
    PoolSkillPackError,
    build_macro_pool_skill_body,
    pool_jev_from_diagnose_macro_pool_skill_response,
    validate_pool_skill_pack,
)


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
            symbol_rows=CRYPTO_FIXTURE_ROWS,
            symbol_result_ids=RID,
        )
        self.assertEqual(body["skill"], "diagnose_macro_pool")
        mp = body["macro_pool"]
        self.assertEqual(mp["date"], "2026-11-20")
        self.assertEqual(len(mp["symbols"]), 2)
        syms = {s["symbol"] for s in mp["symbols"]}
        self.assertEqual(syms, {"BTC-USD.CRYPTO", "ETH-USD.CRYPTO"})

    def test_rejects_duplicate_symbol(self) -> None:
        rows = [CRYPTO_FIXTURE_ROWS[0], {**CRYPTO_FIXTURE_ROWS[0]}]
        with self.assertRaises(PoolSkillPackError):
            validate_pool_skill_pack(
                batch_date="2026-11-20",
                pool_workflow="crypto",
                source_batch_id="batch-demo",
                symbol_rows=rows,
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
                "detail_ref": "diagnose_macro_pool",
            },
        }
        out = pool_jev_from_diagnose_macro_pool_skill_response(resp, min_reason_len=20)
        self.assertTrue(out.get("ok"))
        self.assertEqual(out.get("jev_direction"), "long")
        self.assertEqual(out.get("confidence"), 0.72)
        self.assertEqual(out.get("detail_ref"), "diagnose_macro_pool")

    def test_skill_failure_not_neutral(self) -> None:
        resp = {"status": "failed", "error": "jev down", "session_id": "s1"}
        out = pool_jev_from_diagnose_macro_pool_skill_response(resp)
        self.assertFalse(out.get("ok"))
        self.assertNotIn("jev_direction", out)

    def test_minimal_io_json_documented(self) -> None:
        body = build_macro_pool_skill_body(
            batch_date="2026-11-20",
            pool_workflow="crypto",
            source_batch_id="batch-demo",
            title="加密",
            symbol_rows=CRYPTO_FIXTURE_ROWS[:1],
            symbol_result_ids=RID,
        )
        self.assertTrue(json.dumps(body, ensure_ascii=False))


if __name__ == "__main__":
    unittest.main()