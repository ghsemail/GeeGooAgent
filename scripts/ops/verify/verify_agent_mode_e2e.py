#!/usr/bin/env python3
"""E2E verify Agent mode data load (ops console, no MCP token)."""
from __future__ import annotations

import json
import urllib.error
import urllib.request
from pathlib import Path

import paramiko

DEPLOY = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")
BASE = "http://146.56.225.252:8088/op_agent"
KEY = "sk-7a2b3c4d5e6f7g8h9i0j1k2l3m4n5o6p7q8r9s0t1u2v3w4x5y6z7a8b9c0d1e2"


def get_ghsemail_user_id() -> str:
    cfg = json.loads(DEPLOY.read_text(encoding="utf-8"))
    sc = cfg["targets"]["geegoo-signal"]["ssh"]
    c = paramiko.SSHClient()
    c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    c.connect(sc["host"], username=sc["user"], password=sc.get("password"), timeout=60)
    py = r"""
import json, subprocess
mongo = next(n for n in subprocess.check_output("docker ps --format '{{.Names}}'", shell=True, text=True).splitlines() if 'mongo' in n.lower())
raw = subprocess.check_output(['docker','exec',mongo,'mongosh','Signal_DB','--quiet','--eval',
 "JSON.stringify(db.admin.findOne({username:'ghsemail'},{_id:1})._id)"], text=True).strip()
print(raw)
"""
    _, o, _ = c.exec_command(f"python3 <<'PY'\n{py}\nPY", timeout=60)
    raw = o.read().decode().strip()
    c.close()
    if raw.startswith('{'):
        return json.loads(raw).get("$oid", "")
    return raw.strip('"')


def probe(name: str, path: str, uid: str, *, with_mcp: str = "") -> dict:
    headers = {
        "Authorization": f"Bearer {KEY}",
        "X-User-Id": uid,
        "X-Client-Source": "trading_operation",
    }
    if with_mcp:
        headers["X-MCP-Token"] = with_mcp
    req = urllib.request.Request(f"{BASE}{path}", headers=headers)
    try:
        with urllib.request.urlopen(req, timeout=25) as r:
            body = r.read(400)
            return {"name": name, "ok": True, "status": r.status, "preview": body[:200]}
    except urllib.error.HTTPError as e:
        return {"name": name, "ok": False, "status": e.code, "preview": e.read()[:200]}


def main() -> None:
    uid = get_ghsemail_user_id()
    print(f"ops user_id: {uid[:12]}...")
    endpoints = [
        ("dashboard/data", "/v1/dashboard/data"),
        ("dashboard/events", "/v1/dashboard/events"),
        ("sessions", "/v1/sessions?limit=20"),
        ("metrics/overview", "/v1/metrics/overview"),
        ("doctor", "/v1/doctor?skip_connectivity=true"),
        ("data/overview", "/v1/data/overview"),
    ]
    print("\n=== Ops console (no MCP) ===")
    ok = 0
    for name, path in endpoints:
        r = probe(name, path, uid)
        mark = "OK" if r["ok"] else "FAIL"
        print(f"  [{mark}] {name}: {r['status']} {r['preview']!r}")
        if r["ok"]:
            ok += 1
    print(f"\n{ok}/{len(endpoints)} endpoints OK (no MCP)")

    # dashboard payload sanity
    r = probe("dashboard", "/v1/dashboard/data", uid)
    if r["ok"]:
        full = urllib.request.urlopen(
            urllib.request.Request(
                f"{BASE}/v1/dashboard/data",
                headers={
                    "Authorization": f"Bearer {KEY}",
                    "X-User-Id": uid,
                    "X-Client-Source": "trading_operation",
                },
            ),
            timeout=25,
        )
        data = json.loads(full.read().decode())
        stats = data.get("stats") or {}
        print("\n=== Dashboard payload ===")
        print(f"  provider: {data.get('provider')}")
        print(f"  model: {data.get('model')}")
        print(f"  sessions: {len(data.get('sessions') or [])}")
        print(f"  turns: {stats.get('turns')}")
        print(f"  facts: {len(data.get('facts') or [])}")
        print(f"  doctor_ok: {data.get('doctor_ok')}")

    if ok < len(endpoints):
        raise SystemExit(1)


if __name__ == "__main__":
    main()
