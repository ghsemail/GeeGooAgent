#!/usr/bin/env python3
from __future__ import annotations

import json
import tempfile
import unittest
from pathlib import Path
from typing import Any

from macro_jev_backfill_common import (
    audit_export,
    guard_apply_target,
    parse_export_file,
    require_confirm_no_active_batch,
    validate_export_dates,
)
from macro_jev_backfill_symbols import run_phase_a


def _mock_http_batch_empty():
    def http_json(method: str, url: str, headers: dict, body: Any = None, timeout: int = 60):
        if method == "GET" and "/macro/batch/runs?" in url:
            return 200, {"data": {"items": []}}, "{}"
        return 404, {}, "{}"

    return http_json


class TestBackfillGuards(unittest.TestCase):
    def test_refuse_non_local_apply(self) -> None:
        ok, msg = guard_apply_target(
            apply=True, macro_base="https://signal.example.com", allow_production_host=False
        )
        self.assertFalse(ok)
        self.assertIn("refusing", msg)

    def test_confirm_flag_required_helper(self) -> None:
        ok, msg = require_confirm_no_active_batch(False)
        self.assertFalse(ok)
        self.assertIn("confirm-no-active-batch", msg)

    def test_export_date_strict(self) -> None:
        root = Path(__file__).resolve().parent
        export = parse_export_file(root / "fixtures" / "backfill_export_it.json")
        errs = validate_export_dates(export, "2026-12-01")
        self.assertEqual(errs, [])
        errs2 = validate_export_dates(export, "2026-12-02")
        self.assertTrue(any("!=" in e for e in errs2))

    def test_audit_seven_distinct_run_ids(self) -> None:
        root = Path(__file__).resolve().parent
        export = parse_export_file(root / "fixtures" / "backfill_export_it.json")
        rep = audit_export(export, "2026-12-01")
        self.assertEqual(rep.pool_runs, 7)
        self.assertTrue(rep.canonical_import_symbols_ok)
        self.assertTrue(any(p.needs_resynthesis for p in rep.pool_rows if p.pool_workflow == "bond"))

    def test_phase_a_dry_run(self) -> None:
        root = Path(__file__).resolve().parent
        export_path = root / "fixtures" / "backfill_export_it.json"
        with tempfile.TemporaryDirectory() as td:
            import macro_jev_backfill_common as bc
            import macro_jev_backfill_symbols as sym_mod

            bc.os.environ["MACRO_JEV_BACKFILL_STATE_DIR"] = td
            sym_mod.http_json = _mock_http_batch_empty()  # type: ignore
            rc = run_phase_a(
                batch_date="2026-12-01",
                export_path=export_path,
                macro_base="http://127.0.0.1:19999",
                macro_key="k",
                apply=False,
                confirm_no_active_batch=True,
                allow_production_host=False,
                skip_running=False,
            )
        self.assertEqual(rc, 0)


if __name__ == "__main__":
    unittest.main()