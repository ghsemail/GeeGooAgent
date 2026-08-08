#!/usr/bin/env python3
"""Run postmarket_stock backfill and poll working state."""
from __future__ import annotations

import json
import time
from pathlib import Path

import paramiko

DEPLOY = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")
REPORT_DATE = "2026-08-07"
CODE = "601766.SH"


def ssh(target: str, cmd: str, timeout: int = 60) -> str:
    cfg = json.loads(DEPLOY.read_text(encoding="utf-8"))
    s = cfg["targets"][target]["ssh"]
    c = paramiko.SSHClient()
    c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    c.connect(s["host"], username=s["user"], password=s.get("password"), timeout=30)
    _, o, e = c.exec_command(cmd, timeout=timeout)
    out = (o.read() + e.read()).decode("utf-8", "replace")
    c.close()
    return out


def main() -> int:
    print("=== run postmarket_stock", REPORT_DATE, "===")
    run_cmd = (
        "export PATH=$HOME/.geegoo/bin:$PATH; "
        f"timeout 3600 $HOME/.geegoo/bin/geegoo run "
        f"--config $HOME/.geegoo/config.json --report-date {REPORT_DATE} postmarket_stock 2>&1"
    )
    out = ssh("geegoo-agent", run_cmd, timeout=3620)
    print(out[-4000:])

    print("\n=== latest working json ===")
    latest = ssh(
        "geegoo-agent",
        "ls -t ~/.geegoo/data/working/*.json 2>/dev/null | head -1",
        timeout=30,
    ).strip()
    if latest:
        raw = ssh("geegoo-agent", f"cat {latest}", timeout=30)
        data = json.loads(raw)
        print("file", latest.split("/")[-1])
        print("skill", data.get("skill"), "phase", data.get("phase"), "report_date", data.get("report_date"))
        ws = (data.get("stocks") or {}).get(CODE) or {}
        print(
            CODE,
            "status=", ws.get("status"),
            "change_pct=", ws.get("change_pct"),
            "report_id=", ws.get("report_id"),
        )
        for code, s in (data.get("stocks") or {}).items():
            print(code, s.get("status"), s.get("change_pct"))

    print("\n=== local report file ===")
    rep = ssh(
        "geegoo-agent",
        f"cat ~/.geegoo/data/reports/20260807/{CODE}-postmarket.md 2>/dev/null | head -c 2000 || echo MISSING",
        timeout=30,
    )
    print(rep)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
