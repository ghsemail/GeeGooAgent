#!/usr/bin/env python3
from __future__ import annotations

import json
import tempfile
import unittest
from pathlib import Path

from macro_jev_orchestration import (
    ledger_path,
    load_pool_ledger,
    overall_inputs_from_ledger,
    record_pool_ledger,
    synthesis_pool_jev_valid,
)


class TestOrchestrationLedger(unittest.TestCase):
    def setUp(self) -> None:
        self._td = tempfile.TemporaryDirectory()
        self.addCleanup(self._td.cleanup)
        self._orig = None
        import macro_jev_orchestration as mod

        self._orig = mod.os.environ.get("MACRO_JEV_LEDGER_DIR")
        mod.os.environ["MACRO_JEV_LEDGER_DIR"] = self._td.name

    def test_record_seven_pool_slots_distinct_cron(self) -> None:
        day = "2026-11-17"
        for wf, cron in [
            ("us", "us"),
            ("bond", "bond_finalize"),
        ]:
            record_pool_ledger(
                day,
                wf,
                result_id=f"pool:{wf}:x",
                source_batch_id=f"run-{wf}",
                batch_started_at=f"{day}T05:30:00+08:00",
                status="success",
                cron_workflow=cron,
            )
        led = load_pool_ledger(day)
        self.assertEqual(led["us"]["cron_workflow"], "us")
        self.assertEqual(led["bond"]["source_batch_id"], "run-bond")

    def test_overall_rejects_non_success_ledger_entry(self) -> None:
        day = "2026-11-18"
        ledger = {}
        for wf in ["cn_a", "hk"]:
            ledger[wf] = {
                "result_id": f"pool:{wf}",
                "source_batch_id": "r",
                "batch_started_at": f"{day}T07:00:00+08:00",
                "status": "success" if wf == "cn_a" else "failed",
            }

        def http_json(*_a, **_k):
            return 200, {}, ""

        refs, err = overall_inputs_from_ledger("http://x", "k", http_json, day, ledger)
        self.assertIsNone(refs)
        self.assertTrue("hk" in err or "ledger pools" in err)

    def test_synthesis_requires_identity(self) -> None:
        ok, err = synthesis_pool_jev_valid(
            {"ok": True, "jev_direction": "long", "jev_reason": "x" * 25},
            min_reason_len=20,
        )
        self.assertFalse(ok)
        self.assertIn("session", err)

    def test_ledger_file_is_per_pipeline_date(self) -> None:
        p = ledger_path("2026-11-19")
        self.assertIn("macro_jev_pool_ledger_2026-11-19.json", str(p))
        record_pool_ledger(
            "2026-11-19",
            "crypto",
            result_id="pool:crypto:1",
            source_batch_id="run-c",
            batch_started_at="2026-11-19T16:15:00-05:00",
        )
        data = json.loads(Path(p).read_text(encoding="utf-8"))
        self.assertEqual(data["orchestration_window"], "2026-11-19")


if __name__ == "__main__":
    unittest.main()