#!/usr/bin/env python3
"""Delete 601766 postmarket by report_id, rerun skill, verify."""
from __future__ import annotations

import json
from pathlib import Path

import paramiko

DEPLOY = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")
REPORT_DATE = "2026-08-07"
CODE = "601766.SH"
USER = "6366170502d5c175fd586fe8"
BOT_KEY = "sk-7a2b3c4d5e6f7g8h9i0j1k2l3m4n5o6p7q8r9s0t1u2v3w4x5y6z7a8b9c0d1e2"
KNOWN_REPORT_ID = "6a7734d672677663fa007c35"


def ssh_run(cmd: str, timeout: int = 900) -> str:
    cfg = json.loads(DEPLOY.read_text(encoding="utf-8"))
    ssh = cfg["targets"]["geegoo-agent"]["ssh"]
    c = paramiko.SSHClient()
    c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    c.connect(ssh["host"], username=ssh["user"], password=ssh.get("password"), timeout=30)
    _, o, e = c.exec_command(cmd, timeout=timeout)
    out = (o.read() + e.read()).decode("utf-8", errors="replace")
    c.close()
    return out


REMOTE = f'''
import json, os, sys, urllib.request, urllib.error, subprocess

CODE = "{CODE}"
REPORT_DATE = "{REPORT_DATE}"
USER = "{USER}"
BOT = "{BOT_KEY}"
FALLBACK_ID = "{KNOWN_REPORT_ID}"

cfg = json.load(open(os.path.expanduser("~/.geegoo/config.json")))
tok = cfg.get("mcp_token", "")
key = cfg.get("geegoo_api_key") or cfg.get("api_key", "")
MCP = "http://118.195.135.97:3120"

def post(path, payload):
    req = urllib.request.Request(
        MCP + path,
        data=json.dumps(payload).encode(),
        headers={{"Content-Type": "application/json", "Authorization": "Bearer " + key}},
        method="POST",
    )
    try:
        with urllib.request.urlopen(req, timeout=90) as r:
            return r.status, json.loads(r.read().decode())
    except urllib.error.HTTPError as e:
        body = e.read().decode(errors="replace")
        try:
            return e.code, json.loads(body)
        except Exception:
            return e.code, {{"raw": body[:400]}}

print("=== find report_id ===")
c, r = post("/getStockDailyReports", {{
    "mcp_token": tok,
    "code": CODE,
    "report_date": REPORT_DATE,
}})
pm = (r.get("data") or {{}}).get("stock_postmarket") or []
report_id = None
for row in pm:
    rid = str(row.get("report_id") or row.get("_id") or "")
    if rid:
        report_id = rid
        print("found", rid, "report_date", row.get("report_date"))
if not report_id:
    report_id = FALLBACK_ID
    print("use fallback report_id", report_id)

print("\\n=== delete by report_id ===")
c, r = post("/deleteStockPostmarketReport", {{
    "mcp_token": tok,
    "report_id": report_id,
}})
print("delete", c, r.get("code"), r.get("message"), r.get("data"))

c, r = post("/getStockDailyReports", {{
    "mcp_token": tok,
    "code": CODE,
    "report_date": REPORT_DATE,
}})
pm2 = (r.get("data") or {{}}).get("stock_postmarket") or []
print("postmarket count after delete", len(pm2))
if pm2:
    print("still exists:", [x.get("report_id") for x in pm2])
    sys.exit(2)

print("\\n=== run postmarket_stock ===")
cmd = [
    os.path.expanduser("~/.geegoo/bin/geegoo"), "run",
    "--config", os.path.expanduser("~/.geegoo/config.json"),
    "--report-date", REPORT_DATE,
    "postmarket_stock",
]
proc = subprocess.run(cmd, capture_output=True, text=True, timeout=3600)
print(proc.stdout[-6000:])
if proc.stderr:
    print("STDERR", proc.stderr[-2000:])
print("exit", proc.returncode)
if proc.returncode != 0:
    sys.exit(proc.returncode)

print("\\n=== verify API ===")
body = json.dumps({{"user_id": USER, "phases": ["stock_postmarket"], "limit_per_phase": 20}}).encode()
req = urllib.request.Request(
    "http://127.0.0.1:3140/reports/daily",
    data=body,
    headers={{"Content-Type": "application/json", "Authorization": "Bearer " + BOT}},
    method="POST",
)
data = json.loads(urllib.request.urlopen(req, timeout=60).read())
rows = (data.get("data") or {{}}).get("stock_postmarket") or []
hit = [r for r in rows if r.get("code") == CODE and REPORT_DATE in str(r.get("report_date") or "")]
print("api_code", data.get("code"), "hit", len(hit))
if not hit:
    print("MISSING row")
    sys.exit(1)
r = hit[0]
print("report_id", r.get("report_id"))
print("change_pct", r.get("change_pct"), "session_bias", r.get("session_bias"))
ms = str(r.get("market_summary") or "")
ts = str(r.get("trade_summary") or "")
print("market_summary", ms[:360])
print("trade_summary", ts[:200])
rep = str(r.get("report") or "")
print("report_len", len(rep))
print("report_tail", rep[-320:])
bad = ["$1", "map[]", "多空严", "| 指标 |"]
print("bad_tokens", {{b: (b in ms or b in rep or b in ts) for b in bad}})
'''


def main() -> int:
    print(ssh_run(f"python3 <<'PY'\n{REMOTE}\nPY", timeout=3700), flush=True)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
