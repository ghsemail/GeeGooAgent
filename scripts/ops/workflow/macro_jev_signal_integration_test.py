#!/usr/bin/env python3
"""Integration tests against Signal b3eb834 macro handlers (local Mongo + HTTP)."""

from __future__ import annotations

import json
import os
import socket
import subprocess
import sys
import time
import unittest
import urllib.error
import urllib.request
from typing import Any

_SCRIPT_DIR = os.path.dirname(os.path.abspath(__file__))
if _SCRIPT_DIR not in sys.path:
    sys.path.insert(0, _SCRIPT_DIR)

from macro_jev_orchestration import record_pool_ledger, load_pool_ledger, overall_inputs_from_ledger
from macro_jev_results_client import MacroJevResultsWriter, JEV_PIPELINE_WORKFLOWS


def _mongo_up() -> bool:
    try:
        s = socket.create_connection(("127.0.0.1", 27017), timeout=0.5)
        s.close()
        return True
    except OSError:
        return False


def http_json(
    method: str,
    url: str,
    headers: dict[str, str],
    body: dict[str, Any] | None = None,
    timeout: int = 60,
) -> tuple[int, Any, str]:
    data = json.dumps(body or {}, ensure_ascii=False).encode("utf-8") if body is not None else None
    req = urllib.request.Request(url, data=data, headers=headers, method=method)
    try:
        with urllib.request.urlopen(req, timeout=timeout) as resp:
            raw = resp.read().decode("utf-8", errors="replace")
            return resp.status, json.loads(raw) if raw.strip() else {}, raw
    except urllib.error.HTTPError as e:
        raw = e.read().decode("utf-8", errors="replace")
        try:
            parsed = json.loads(raw) if raw.strip() else {}
        except json.JSONDecodeError:
            parsed = {"raw": raw}
        return e.code, parsed, raw


def post_batch_run(base: str, key: str, body: dict[str, Any]) -> str:
    st, data, _ = http_json(
        "POST",
        f"{base.rstrip('/')}/macro/batch/runs",
        {"Content-Type": "application/json", "Authorization": f"Bearer {key}"},
        body,
    )
    if st != 200:
        return ""
    return str((data.get("data") or {}).get("run", {}).get("id") or data.get("id") or "")


@unittest.skipUnless(_mongo_up(), "local Mongo required on 127.0.0.1:27017")
class TestSignalMacroJevIntegration(unittest.TestCase):
    @classmethod
    def _pick_port(cls) -> int:
        s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        s.bind(("127.0.0.1", 0))
        port = int(s.getsockname()[1])
        s.close()
        return port

    @classmethod
    def setUpClass(cls) -> None:
        cls.port = cls._pick_port()
        cls.addr = f"127.0.0.1:{cls.port}"
        cls.base = f"http://{cls.addr}"
        cls.key = "jev-it-key"
        env = {
            **os.environ,
            "MACRO_JEV_IT_ADDR": cls.addr,
            "MACRO_JEV_IT_API_KEY": cls.key,
            "GEEGOO_SIGNAL_MONGO_URI": os.environ.get(
                "GEEGOO_SIGNAL_MONGO_URI", "mongodb://127.0.0.1:27017"
            ),
        }
        sig_root = os.environ.get("GEEGOO_SIGNAL_ROOT", "/agent/repos/GeeGooSignal")
        env.update(
            {
                "GEEGOO_SIGNAL_MACRO_API_PORT": str(cls.port),
                "GEEGOO_SIGNAL_MACRO_API_KEY": cls.key,
            }
        )
        cls._proc = subprocess.Popen(
            ["go", "run", "./cmd/macro-api"],
            cwd=sig_root,
            env=env,
            stdout=subprocess.PIPE,
            stderr=subprocess.STDOUT,
            text=True,
        )
        deadline = time.time() + 45
        cls._ready = False
        while time.time() < deadline:
            if cls._proc.poll() is not None:
                break
            try:
                st, _, _ = http_json(
                    "GET",
                    f"{cls.base}/macro/jev/results/symbols?date=2099-01-01&source=canonical",
                    {"Authorization": f"Bearer {cls.key}"},
                )
                if st in (200, 400):
                    cls._ready = True
                    break
            except (urllib.error.URLError, ConnectionError):
                pass
            time.sleep(0.3)
        if not cls._ready:
            out = cls._proc.stdout.read() if cls._proc.stdout else ""
            raise unittest.SkipTest(f"signaljevit server not ready: {out[:300]}")

    @classmethod
    def tearDownClass(cls) -> None:
        if getattr(cls, "_proc", None):
            cls._proc.terminate()
            try:
                cls._proc.wait(timeout=5)
            except subprocess.TimeoutExpired:
                cls._proc.kill()

    def test_running_to_final_revision_latest(self) -> None:
        date = "2026-11-25"
        started = f"{date}T05:30:00+08:00"
        run_id = post_batch_run(
            self.base,
            self.key,
            {
                "workflow": "us",
                "date": date,
                "started_at": started,
                "finished_at": started,
                "success": False,
                "pools": [
                    {
                        "name": "美股池",
                        "symbols": ["SPY.US"],
                        "items": [{"symbol": "SPY.US", "success": False, "error": "诊断中"}],
                    }
                ],
                "summary": "it",
            },
        )
        self.assertTrue(run_id)
        w = MacroJevResultsWriter(
            self.base, self.key, date, "us", run_id, started, http_json
        )
        o0 = w.put_symbol(symbol="SPY.US", status="running", revision=0)
        self.assertTrue(o0.ok, o0.reason or o0.error)
        o1 = w.put_symbol(
            symbol="SPY.US",
            status="success",
            revision=1,
            jev_direction="long",
            jev_reason="integration pass reason text",
            confidence=0.7,
            agent_session_id="chat-it",
            agent_run_id="flow-it",
            detail_ref="buy:1",
            completed_at=f"{date}T05:40:00+08:00",
        )
        self.assertTrue(o1.ok)
        st, data, _ = http_json(
            "GET",
            f"{self.base}/macro/jev/results/symbols/SPY.US?date={date}&pool_workflow=us&source=canonical",
            {"Authorization": f"Bearer {self.key}"},
        )
        self.assertEqual(st, 200)
        result = (data.get("data") or {}).get("result") or {}
        self.assertEqual(result.get("status"), "success")
        self.assertIn("@r1", str(result.get("result_id")))

    def test_immutable_conflict_accepted_false(self) -> None:
        date = "2026-11-26"
        started = f"{date}T06:00:00+08:00"
        run_id = post_batch_run(
            self.base,
            self.key,
            {
                "workflow": "hk",
                "date": date,
                "started_at": started,
                "finished_at": started,
                "success": False,
                "pools": [
                    {
                        "name": "港股池",
                        "symbols": ["800000.HK"],
                        "items": [{"symbol": "800000.HK", "success": False, "error": "诊断中"}],
                    }
                ],
            },
        )
        w = MacroJevResultsWriter(
            self.base, self.key, date, "hk", run_id, started, http_json
        )
        w.put_symbol(
            symbol="800000.HK",
            status="success",
            revision=0,
            jev_direction="neutral",
            jev_reason="first version reason ok",
            confidence=0.5,
            agent_session_id="s",
            agent_run_id="r",
            detail_ref="buy:1",
        )
        out = w.put_symbol(
            symbol="800000.HK",
            status="success",
            revision=0,
            jev_direction="short",
            jev_reason="mutated reason must fail",
            confidence=0.5,
            agent_session_id="s",
            agent_run_id="r",
            detail_ref="buy:1",
        )
        self.assertFalse(out.ok)
        self.assertEqual(out.reason, "immutable_version_conflict")

    def test_bond_dual_segment_manifest_pool(self) -> None:
        date = "2026-11-27"
        t0 = f"{date}T08:50:00+08:00"
        t1 = f"{date}T08:55:00+08:00"
        run_sh = post_batch_run(
            self.base,
            self.key,
            {
                "workflow": "bond",
                "date": date,
                "started_at": t0,
                "finished_at": t0,
                "success": True,
                "pools": [
                    {
                        "name": "债券利率池",
                        "symbols": ["HYG.US", "TLT.US"],
                        "items": [
                            {
                                "symbol": "HYG.US",
                                "success": True,
                                "jev_direction": "long",
                                "jev_reason": "x" * 24,
                                "detail_ref": "buy:1",
                                "agent_session_id": "s1",
                                "agent_run_id": "r1",
                            }
                        ],
                    }
                ],
            },
        )
        run_us = post_batch_run(
            self.base,
            self.key,
            {
                "workflow": "bond",
                "date": date,
                "started_at": t1,
                "finished_at": t1,
                "success": True,
                "pools": [
                    {
                        "name": "债券利率池",
                        "symbols": ["HYG.US", "TLT.US"],
                        "items": [
                            {
                                "symbol": "TLT.US",
                                "success": True,
                                "jev_direction": "short",
                                "jev_reason": "y" * 24,
                                "detail_ref": "buy:2",
                                "agent_session_id": "s2",
                                "agent_run_id": "r2",
                            }
                        ],
                    }
                ],
            },
        )
        self.assertTrue(run_sh and run_us)
        self.assertNotEqual(run_sh, run_us)
        w_sh = MacroJevResultsWriter(
            self.base, self.key, date, "bond", run_sh, t0, http_json
        )
        w_us = MacroJevResultsWriter(
            self.base, self.key, date, "bond", run_us, t1, http_json
        )
        s1 = w_sh.put_symbol(
            symbol="HYG.US",
            status="success",
            revision=0,
            jev_direction="long",
            jev_reason="bond sh leg reason ok",
            confidence=0.6,
            agent_session_id="s1",
            agent_run_id="r1",
            detail_ref="buy:1",
        )
        s2 = w_us.put_symbol(
            symbol="TLT.US",
            status="success",
            revision=0,
            jev_direction="short",
            jev_reason="bond us leg reason ok",
            confidence=0.6,
            agent_session_id="s2",
            agent_run_id="r2",
            detail_ref="buy:2",
        )
        self.assertTrue(s1.ok and s2.ok)
        started_fin = f"{date}T09:00:00+08:00"
        run_fin = post_batch_run(
            self.base,
            self.key,
            {
                "workflow": "bond",
                "date": date,
                "started_at": started_fin,
                "finished_at": started_fin,
                "success": True,
                "pools": [
                    {
                        "name": "债券利率池",
                        "symbols": ["HYG.US", "TLT.US"],
                        "items": [
                            {"symbol": "HYG.US", "success": True, "jev_direction": "long"},
                            {"symbol": "TLT.US", "success": True, "jev_direction": "short"},
                        ],
                    }
                ],
                "summary": "bond finalize",
            },
        )
        wf = MacroJevResultsWriter(
            self.base, self.key, date, "bond", run_fin, started_fin, http_json
        )
        wf.symbol_result_ids["HYG.US"] = s1.result_id
        wf.symbol_result_ids["TLT.US"] = s2.result_id
        pool = wf.put_pool(
            status="success",
            completed_at=started_fin,
            jev_direction="neutral",
            jev_reason="bond merged pool synthesis reason",
            confidence=0.55,
            agent_session_id="pool-s",
            agent_run_id="pool-r",
            inputs=[
                {"result_id": s1.result_id},
                {"result_id": s2.result_id},
            ],
        )
        self.assertTrue(pool.ok, pool.reason or pool.error)

    def test_backfill_phase_a_then_pools_overall_recorded_synthesis(self) -> None:
        import tempfile
        from pathlib import Path

        import macro_jev_backfill_common as bc
        import macro_jev_backfill_symbols as sym_mod
        import macro_jev_backfill_pools_overall as pool_mod
        import macro_jev_orchestration as orch_mod

        day = "2026-12-01"
        root = Path(__file__).resolve().parent
        export = root / "fixtures" / "backfill_export_it.json"
        synth = root / "fixtures" / "synthesis"
        with tempfile.TemporaryDirectory() as td:
            bc.os.environ["MACRO_JEV_BACKFILL_STATE_DIR"] = td
            orch_mod.os.environ["MACRO_JEV_LEDGER_DIR"] = td
            sym_mod.http_json = http_json  # type: ignore
            rc_a = sym_mod.run_phase_a(
                batch_date=day,
                export_path=export,
                macro_base=self.base,
                macro_key=self.key,
                apply=True,
                confirm_no_active_batch=True,
                allow_production_host=False,
                skip_running=False,
            )
            self.assertEqual(rc_a, 0, "phase A apply")
            state = bc.load_backfill_state(day)
            self.assertGreaterEqual(
                sum(1 for v in state.phase_a.values() if v.get("status") == "success"), 8
            )
            export_data = bc.parse_export_file(export)
            rc_p = 0
            for wf in JEV_PIPELINE_WORKFLOWS:
                rc_p = max(
                    rc_p,
                    pool_mod.run_one_pool(
                        wf=wf,
                        batch_date=day,
                        export=export_data,
                        state=bc.load_backfill_state(day),
                        macro_base=self.base,
                        macro_key=self.key,
                        agent_base="http://127.0.0.1:9",
                        apply=True,
                        recorded_dir=synth,
                    ),
                )
            self.assertEqual(rc_p, 0, "phase B pools")
            rc_o = pool_mod.run_overall(
                batch_date=day,
                macro_base=self.base,
                macro_key=self.key,
                agent_base="http://127.0.0.1:9",
                apply=True,
                recorded_dir=synth,
                state=bc.load_backfill_state(day),
            )
            self.assertEqual(rc_o, 0, "phase C overall")
            st, data, _ = http_json(
                "GET",
                f"{self.base}/macro/jev/results/overall?date={day}&source=canonical",
                {"Authorization": f"Bearer {self.key}"},
            )
            self.assertEqual(st, 200)
            result = (data.get("data") or {}).get("result") or {}
            self.assertEqual(result.get("status"), "success")

    def test_prod_shape_33_symbols_bond5_manifest_http(self) -> None:
        import json
        import tempfile
        from pathlib import Path

        import macro_jev_backfill_common as bc
        import macro_jev_backfill_fixtures as bfix
        import macro_jev_backfill_pools_overall as pool_mod
        import macro_jev_backfill_symbols as sym_mod
        import macro_jev_orchestration as orch_mod

        day = "2026-11-17"
        export_data = bfix.build_prod_shape_audit_export(day)
        synth = Path(__file__).resolve().parent / "fixtures" / "synthesis"
        with tempfile.TemporaryDirectory() as td:
            export_path = Path(td) / "export.json"
            export_path.write_text(json.dumps(export_data, ensure_ascii=False), encoding="utf-8")
            bc.os.environ["MACRO_JEV_BACKFILL_STATE_DIR"] = td
            orch_mod.os.environ["MACRO_JEV_LEDGER_DIR"] = td
            sym_mod.http_json = http_json  # type: ignore
            rc_a = sym_mod.run_phase_a(
                batch_date=day,
                export_path=export_path,
                macro_base=self.base,
                macro_key=self.key,
                apply=True,
                confirm_no_active_batch=True,
                allow_production_host=False,
                skip_running=False,
            )
            self.assertEqual(rc_a, 0)
            self.assertEqual(
                sum(
                    1
                    for v in bc.load_backfill_state(day).phase_a.values()
                    if v.get("status") == "success"
                ),
                33,
            )
            bond_manifest = bc.pool_manifest_symbols_from_export(export_data, "bond")
            self.assertEqual(len(bond_manifest), 5)
            self.assertTrue(any(s.endswith(".SH") for s in bond_manifest))
            rc_p = pool_mod.run_one_pool(
                wf="bond",
                batch_date=day,
                export=export_data,
                state=bc.load_backfill_state(day),
                macro_base=self.base,
                macro_key=self.key,
                agent_base="http://127.0.0.1:9",
                apply=True,
                recorded_dir=synth,
            )
            self.assertEqual(rc_p, 0)
            st, data, _ = http_json(
                "GET",
                f"{self.base}/macro/jev/results/pools/bond?date={day}&source=canonical",
                {"Authorization": f"Bearer {self.key}"},
            )
            self.assertEqual(st, 200)
            result = (data.get("data") or {}).get("result") or {}
            inputs = result.get("inputs") or []
            self.assertEqual(len(inputs), 5, "bond pool must freeze five child result_ids")

    def test_overall_blocked_writes_failed_not_old_success(self) -> None:
        import tempfile
        import macro_jev_orchestration as mod

        day = "2026-11-28"
        with tempfile.TemporaryDirectory() as td:
            mod.os.environ["MACRO_JEV_LEDGER_DIR"] = td
            for wf in JEV_PIPELINE_WORKFLOWS[:3]:
                record_pool_ledger(
                    day,
                    wf,
                    result_id=f"pool:{wf}:stub",
                    source_batch_id=f"run-{wf}",
                    batch_started_at=f"{day}T07:00:00+08:00",
                    status="success",
                )
            ledger = load_pool_ledger(day)
            refs, err = overall_inputs_from_ledger(
                self.base, self.key, http_json, day, ledger
            )
            self.assertIsNone(refs)
            self.assertIn("ledger pools", err)


if __name__ == "__main__":
    unittest.main()