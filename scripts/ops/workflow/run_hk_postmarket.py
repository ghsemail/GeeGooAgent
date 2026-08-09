#!/usr/bin/env python3
"""Run HK postmarket_stock for latest session and verify."""
from __future__ import annotations

import json
from datetime import datetime, timedelta
from pathlib import Path

import paramiko

DEPLOY = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")
USER = "6366170502d5c175fd586fe8"
BOT_KEY = "sk-7a2b3c4d5e6f7g8h9i0j1k2l3m4n5o6p7q8r9s0t1u2v3w4x5y6z7a8b9c0d1e2"
HK_CODE = "00700.HK"


def ssh(cmd: str, timeout: int = 3700) -> str:
    cfg = json.loads(DEPLOY.read_text(encoding="utf-8"))
    s = cfg["targets"]["geegoo-agent"]["ssh"]
    c = paramiko.SSHClient()
    c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    c.connect(s["host"], username=s["user"], password=s.get("password"), timeout=30)
    _, o, e = c.exec_command(cmd, timeout=timeout)
    out = (o.read() + e.read()).decode("utf-8", errors="replace")
    c.close()
    return out


REMOTE = f'''
import json, os, sys, urllib.request, urllib.error, subprocess
from datetime import datetime, timedelta

USER = "{USER}"
BOT = "{BOT_KEY}"
HK_CODE = "{HK_CODE}"
MCP = "http://118.195.135.97:3120"

cfg = json.load(open(os.path.expanduser("~/.geegoo/config.json")))
tok = cfg.get("mcp_token", "")
key = cfg.get("geegoo_api_key") or cfg.get("api_key", "")

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

# pick HK trading date: try yesterday then walk back
report_date = None
for delta in range(0, 8):
    d = (datetime.now() - timedelta(days=delta)).strftime("%Y-%m-%d")
    c, r = post("/checkTradingDay", {{"mcp_token": tok, "code": HK_CODE, "date": d}})
    data = r.get("data") or {{}}
    if data.get("is_trading_day"):
        report_date = d
        print("trading_day", d, "market", data.get("market"))
        break
if not report_date:
    report_date = (datetime.now() - timedelta(days=1)).strftime("%Y-%m-%d")
    print("fallback report_date", report_date)

print("\\n=== existing HK postmarket ===")
c, r = post("/getStockDailyReports", {{"mcp_token": tok, "code": HK_CODE, "report_date": report_date}})
pm = (r.get("data") or {{}}).get("stock_postmarket") or []
print("count", len(pm), "for", HK_CODE, report_date)
for row in pm:
    rid = str(row.get("report_id") or row.get("_id") or "")
    print(" delete", rid)
    if rid:
        c2, r2 = post("/deleteStockPostmarketReport", {{"mcp_token": tok, "report_id": rid}})
        print("  ", c2, r2.get("code"), r2.get("message"))

print("\\n=== run postmarket_stock", report_date, "===")
cmd = [
    os.path.expanduser("~/.geegoo/bin/geegoo"), "run",
    "--config", os.path.expanduser("~/.geegoo/config.json"),
    "--report-date", report_date,
    "postmarket_stock",
]
proc = subprocess.run(cmd, capture_output=True, text=True, timeout=3600)
print(proc.stdout[-4000:])
if proc.stderr:
    print("STDERR", proc.stderr[-1500:])
print("exit", proc.returncode)

print("\\n=== working state for", HK_CODE, "===")
from pathlib import Path
runs = sorted(Path(os.path.expanduser("~/.geegoo/data/working")).glob("*.json"), key=lambda p: p.stat().st_mtime, reverse=True)
if runs:
    d = json.loads(runs[0].read_text())
    ws = (d.get("stocks") or {{}}).get(HK_CODE) or {{}}
    print("status", ws.get("status"), "change_pct", ws.get("change_pct"), "report_id", ws.get("report_id"))

print("\\n=== API verify ===")
body = json.dumps({{"user_id": USER, "phases": ["stock_postmarket"], "limit_per_phase": 20}}).encode()
req = urllib.request.Request(
    "http://118.195.135.97:3140/reports/daily",
    data=body,
    headers={{"Content-Type": "application/json", "Authorization": "Bearer " + BOT}},
    method="POST",
)
api = json.loads(urllib.request.urlopen(req, timeout=60).read())
rows = (api.get("data") or {{}}).get("stock_postmarket") or []
hit = [x for x in rows if x.get("code") == HK_CODE and report_date in str(x.get("report_date") or "")]
print("api hit", len(hit), "report_date", report_date)
if hit:
    r0 = hit[0]
    print("report_id", r0.get("report_id"))
    print("change_pct", r0.get("change_pct"), "session_bias", r0.get("session_bias"))
    ms = str(r0.get("market_summary") or "")
    print("market_summary", ms[:200])
    print("report_len", len(str(r0.get("report") or "")))
else:
  # list HK rows
  hk = [x for x in rows if str(x.get("code","")).endswith(".HK")]
  print("hk_rows", [(x.get("code"), x.get("report_date"), x.get("report_id")) for x in hk[:5]])
  sys.exit(1)
'''


def main() -> int:
    print(ssh(f"python3 <<'PY'\n{REMOTE}\nPY"), flush=True)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
