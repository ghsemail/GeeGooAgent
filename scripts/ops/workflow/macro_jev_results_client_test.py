#!/usr/bin/env python3
from __future__ import annotations

import json
import unittest
from typing import Any

from macro_jev_results_client import (
    MacroJevResultsWriter,
    PutOutcome,
    collect_overall_pool_inputs,
    jev_stable_source_batch_id,
    local_symbol_result_id,
    make_source_batch_id,
    resolve_pool_day_symbol_result_ids,
)


def mock_http(registry: dict[str, Any]):
    def http_json(
        method: str,
        url: str,
        headers: dict[str, str],
        body: dict[str, Any] | None = None,
        timeout: int = 120,
    ) -> tuple[int, Any, str]:
        key = f"{method} {url}"
        reg = registry.get(key)
        if callable(reg):
            return reg(method, url, body, headers)
        if isinstance(reg, tuple):
            return reg
        return 404, {"message": "not found"}, "{}"

    return http_json


class TestMacroJevResultsClient(unittest.TestCase):
    def test_local_symbol_result_id_revision(self) -> None:
        rid0 = local_symbol_result_id("2026-11-17", "cn_a", "000001.SH", "run-1", 0)
        rid1 = local_symbol_result_id("2026-11-17", "cn_a", "000001.SH", "run-1", 1)
        self.assertEqual(rid0, "sym:2026-11-17:cn_a:000001.SH:run-1")
        self.assertEqual(rid1, "sym:2026-11-17:cn_a:000001.SH:run-1@r1")

    def test_put_bumps_revision_on_immutable_conflict(self) -> None:
        calls: list[dict[str, Any]] = []

        def put_symbol(_m, _url, body, _h):
            calls.append(body or {})
            rev = int((body or {}).get("revision") or 0)
            if rev == 1:
                return (
                    200,
                    {
                        "code": 100,
                        "data": {
                            "accepted": False,
                            "reason": "immutable_version_conflict",
                            "result_id": "sym:x",
                        },
                    },
                    "{}",
                )
            return (
                200,
                {"code": 100, "data": {"accepted": True, "result_id": "sym:y"}},
                "{}",
            )

        reg = {"PUT http://sig/macro/jev/results/symbols": put_symbol}
        w = MacroJevResultsWriter(
            "http://sig",
            "key",
            "2026-11-17",
            "cn_a",
            "run-abc",
            "2026-11-17T07:00:00+08:00",
            mock_http(reg),
        )
        w.symbol_revision_final = 1
        out = w.put_symbol(symbol="000001.SH", status="success", jev_direction="long")
        self.assertTrue(out.ok)
        self.assertEqual(calls[0]["revision"], 1)
        self.assertEqual(calls[1]["revision"], 2)
        self.assertEqual(w.symbol_result_ids.get("000001.SH"), "sym:y")
        self.assertFalse(w.has_errors())

    def test_bind_batch_run_matches_macro_api_run_id(self) -> None:
        calls: list[dict[str, Any]] = []

        def put_symbol(_m, url, body, _h):
            calls.append(body or {})
            return (
                200,
                {"code": 100, "data": {"accepted": True, "result_id": "sym:1"}},
                "{}",
            )

        reg = {"PUT http://sig/macro/jev/results/symbols": put_symbol}
        w = MacroJevResultsWriter(
            "http://sig",
            "key",
            "2026-11-17",
            "us",
            "",
            "2026-11-17T05:30:00+08:00",
            mock_http(reg),
        )
        w.bind_batch_run("mongo-run-id-42")
        w.put_symbol(symbol="SPY.US", status="running", revision=0)
        self.assertEqual(calls[0]["source_batch_id"], "mongo-run-id-42")
        self.assertEqual(calls[0]["batch_started_at"], "2026-11-17T05:30:00+08:00")
        self.assertEqual(calls[0]["revision"], 0)

    def test_put_pool_success_includes_choices(self) -> None:
        captured: dict[str, Any] = {}

        def put_pool(_m, _u, body, _h):
            captured.update(body or {})
            return (
                200,
                {"code": 100, "data": {"accepted": True, "result_id": "pool:ch"}},
                "{}",
            )

        reg = {"PUT http://sig/macro/jev/results/pools/crypto": put_pool}
        w = MacroJevResultsWriter(
            "http://sig",
            "key",
            "2026-11-20",
            "crypto",
            "run-pool",
            "2026-11-20T09:00:00+08:00",
            mock_http(reg),
        )
        choices = [
            {"label": "看多", "probability": 0.75},
            {"label": "看空", "probability": 0.15},
            {"label": "不明确", "probability": 0.1},
        ]
        out = w.put_pool(
            status="success",
            completed_at="2026-11-20T09:10:00+08:00",
            jev_direction="long",
            jev_reason="y" * 30,
            confidence=0.75,
            choices=choices,
        )
        self.assertTrue(out.ok)
        self.assertEqual(captured.get("choices"), choices)

    def test_put_pool_from_synthesis_forwards_choices(self) -> None:
        captured: list[dict[str, Any]] = []

        def put_pool(_m, _u, body, _h):
            captured.append(body or {})
            return (
                200,
                {"code": 100, "data": {"accepted": True, "result_id": "pool:syn"}},
                "{}",
            )

        reg = {"PUT http://sig/macro/jev/results/pools/crypto": put_pool}
        w = MacroJevResultsWriter(
            "http://sig",
            "key",
            "2026-11-20",
            "crypto",
            "run-pool",
            "2026-11-20T09:00:00+08:00",
            mock_http(reg),
        )
        w.bind_batch_run("run-pool")
        w.symbol_result_ids["BTC-USD.CRYPTO"] = "sym:btc"
        w.symbol_result_ids["ETH-USD.CRYPTO"] = "sym:eth"
        choices = [
            {"label": "看多", "probability": 0.6},
            {"label": "看空", "probability": 0.25},
            {"label": "不明确", "probability": 0.15},
        ]
        pool_jev = {
            "success": True,
            "jev_direction": "long",
            "jev_reason": "z" * 30,
            "confidence": 0.6,
            "choices": choices,
        }
        out = w.put_pool_from_synthesis(
            pool_jev,
            symbol_items=[
                {"symbol": "BTC-USD.CRYPTO", "success": True},
                {"symbol": "ETH-USD.CRYPTO", "success": True},
            ],
            all_symbols_ok=True,
            completed_at="2026-11-20T09:11:00+08:00",
        )
        self.assertIsNotNone(out)
        assert out is not None
        self.assertTrue(out.ok)
        self.assertEqual(captured[-1].get("choices"), choices)

    def test_pool_incomplete_when_expected_symbols_missing(self) -> None:
        sym_put: dict[str, Any] = {}

        def put_symbol(_m, _u, body, _h):
            sym_put.update(body or {})
            return (
                200,
                {"code": 100, "data": {"accepted": True, "result_id": "sym:a"}},
                "{}",
            )

        pool_calls: list[dict[str, Any]] = []

        def put_pool(_m, _u, body, _h):
            pool_calls.append(body or {})
            return (
                200,
                {"code": 100, "data": {"accepted": True, "result_id": "pool:1"}},
                "{}",
            )

        reg = {
            "PUT http://sig/macro/jev/results/symbols": put_symbol,
            "PUT http://sig/macro/jev/results/pools/cn_a": put_pool,
        }
        w = MacroJevResultsWriter(
            "http://sig",
            "key",
            "2026-11-17",
            "cn_a",
            "run-id",
            "2026-11-17T07:00:00+08:00",
            mock_http(reg),
        )
        w.put_symbol_from_batch_item(
            {
                "symbol": "000001.SH",
                "success": True,
                "jev_direction": "long",
                "jev_reason": "x" * 30,
                "confidence": 0.7,
                "agent_session_id": "s",
                "agent_run_id": "r",
                "detail_ref": "buy:1",
            },
            completed_at="2026-11-17T07:10:00+08:00",
        )
        out = w.put_pool_from_synthesis(
            {
                "success": True,
                "jev_direction": "long",
                "jev_reason": "y" * 30,
                "choices": [
                    {"label": "看多", "probability": 0.8},
                    {"label": "看空", "probability": 0.1},
                    {"label": "不明确", "probability": 0.1},
                ],
            },
            symbol_items=[{"symbol": "000001.SH", "success": True}],
            all_symbols_ok=True,
            completed_at="2026-11-17T07:11:00+08:00",
            expected_symbols=["000001.SH", "399001.SZ"],
        )
        self.assertIsNotNone(out)
        assert out is not None
        self.assertTrue(out.ok)
        self.assertEqual(pool_calls[-1]["status"], "incomplete")

    def test_collect_overall_seven_pools(self) -> None:
        workflows = [
            "cn_a",
            "hk",
            "us",
            "commodity_fx",
            "bond",
            "crypto",
            "volatility",
        ]

        def get_pool(_m, url, _b, _h):
            wf = url.rsplit("/", 1)[-1].split("?")[0]
            return (
                200,
                {
                    "code": 100,
                    "data": {
                        "result": {
                            "result_id": f"pool:{wf}",
                            "status": "success",
                            "pool_workflow": wf,
                        }
                    },
                },
                "{}",
            )

        reg = {}
        for wf in workflows:
            reg[f"GET http://sig/macro/jev/results/pools/{wf}?date=2026-11-17&source=canonical"] = (
                get_pool
            )
        rows, err = collect_overall_pool_inputs(
            "http://sig", "key", mock_http(reg), "2026-11-17"
        )
        self.assertEqual(err, "")
        self.assertIsNotNone(rows)
        assert rows is not None
        self.assertEqual(len(rows), 7)
        self.assertEqual(rows[0]["pool_workflow"], "cn_a")

    def test_put_network_retries_then_ok(self) -> None:
        calls = {"n": 0}

        def flaky_put(_m, _u, body, _h):
            calls["n"] += 1
            if calls["n"] < 3:
                return 503, {"message": "unavailable"}, "{}"
            return (
                200,
                {"code": 100, "data": {"accepted": True, "result_id": "sym:retry-ok"}},
                "{}",
            )

        reg = {"PUT http://sig/macro/jev/results/symbols": flaky_put}
        w = MacroJevResultsWriter(
            "http://sig",
            "key",
            "2026-11-17",
            "us",
            "run-1",
            "2026-11-17T05:30:00+08:00",
            mock_http(reg),
            put_network_retries=3,
        )
        out = w.put_symbol(
            symbol="SPY.US",
            status="success",
            revision=1,
            jev_direction="long",
            jev_reason="network retry integration reason",
            agent_session_id="s",
            agent_run_id="r",
            detail_ref="buy:1",
        )
        self.assertTrue(out.ok, out.error)
        self.assertEqual(calls["n"], 3)

    def test_put_blocked_without_batch_run_id(self) -> None:
        w = MacroJevResultsWriter(
            "http://sig",
            "key",
            "2026-11-17",
            "bond",
            "",
            "2026-11-17T05:30:00+08:00",
            mock_http({}),
        )
        out = w.put_symbol(symbol="HYG.US", status="running", revision=0)
        self.assertFalse(out.ok)
        self.assertEqual(out.error, "jev_source_batch_id_missing")
        self.assertTrue(w.has_errors())

    def test_jev_stable_source_batch_id_uses_date_not_mongo(self) -> None:
        self.assertEqual(
            jev_stable_source_batch_id("cn_a", "2026-11-23"),
            "macro-jev-cn_a-2026-11-23",
        )

    def test_resolve_pool_day_symbol_result_ids_per_symbol(self) -> None:
        ok_sym = {
            "code": 100,
            "data": {
                "result": {
                    "result_id": "sym:legacy:000001",
                    "status": "success",
                }
            },
        }
        reg = {
            "GET http://sig/macro/jev/results/symbols?date=2026-11-23&pool_workflow=cn_a&source=canonical": (
                200,
                {"code": 100, "data": {"items": []}},
                "{}",
            ),
            "GET http://sig/macro/jev/results/symbols/000001.SH?date=2026-11-23&pool_workflow=cn_a&source=canonical": (
                200,
                ok_sym,
                "{}",
            ),
            "GET http://sig/macro/jev/results/symbols/000001.SH?date=2026-11-23&pool_workflow=cn_a&source=auto": (
                200,
                ok_sym,
                "{}",
            ),
        }
        http = mock_http(reg)
        ids, err = resolve_pool_day_symbol_result_ids(
            "http://sig",
            "key",
            http,
            "2026-11-23",
            "cn_a",
            ["000001.SH", "399001.SZ"],
        )
        self.assertIn("399001.SZ", err or "")
        self.assertEqual(ids.get("000001.SH"), "sym:legacy:000001")


if __name__ == "__main__":
    unittest.main()