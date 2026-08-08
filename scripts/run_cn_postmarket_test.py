#!/usr/bin/env python3
"""Deploy GeeGooAgent, run A-share postmarket_stock backfill, verify report."""
from __future__ import annotations

import json
from pathlib import Path

import paramiko

DEPLOY = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")
REPORT_DATE = "2026-08-07"
CODE = "601766.SH"
USER = "6366170502d5c175fd586fe8"
BOT_KEY = "sk-7a2b3c4d5e6f7g8h9i0j1k2l3m4n5o6p7q8r9s0t1u2v3w4x5y6z7a8b9c0d1e2"


def ssh_run(target: str, cmd: str, timeout: int = 900) -> str:
    cfg = json.loads(DEPLOY.read_text(encoding="utf-8"))
    ssh = cfg["targets"][target]["ssh"]
    c = paramiko.SSHClient()
    c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    c.connect(ssh["host"], username=ssh["user"], password=ssh.get("password"), timeout=30)
    _, o, e = c.exec_command(cmd, timeout=timeout)
    out = (o.read() + e.read()).decode("utf-8", errors="replace")
    c.close()
    return out


def main() -> int:
    print("=== 1) Install/update GeeGooAgent ===")
    install = json.loads(DEPLOY.read_text(encoding="utf-8"))["targets"]["geegoo-agent"]["install_cmd"]
    print(ssh_run("geegoo-agent", install, timeout=900))

    print(f"\n=== 2) Delete existing postmarket for {CODE} @ {REPORT_DATE} (idempotency) ===")
    delete_py = f'''
import json, os
from pymongo import MongoClient
cfg = json.load(open(os.path.expanduser("~/.geegoo/config.json")))
uri = cfg.get("mongo_uri") or cfg.get("mongodb_uri") or ""
client = MongoClient(uri)
db = client.get_default_database()
coll = db["stock_postmarket_report"]
q = {{"code": "{CODE}", "report_date": "{REPORT_DATE}"}}
res = coll.delete_many(q)
print("deleted", res.deleted_count, "from stock_postmarket_report")
'''
    print(ssh_run("geegoo-agent", f"python3 <<'PY'\n{delete_py}\nPY", timeout=60))

    print(f"\n=== 3) Run postmarket_stock for {REPORT_DATE} ===")
    run_cmd = (
        "export PATH=$HOME/.geegoo/bin:$PATH; "
        f"timeout 1200 $HOME/.geegoo/bin/geegoo run "
        f"--config $HOME/.geegoo/config.json --report-date {REPORT_DATE} postmarket_stock 2>&1"
    )
    print(ssh_run("geegoo-agent", run_cmd, timeout=1220))

    print(f"\n=== 4) Mongo check {CODE} ===")
    mongo_py = f'''
import json, os
from pymongo import MongoClient
cfg = json.load(open(os.path.expanduser("~/.geegoo/config.json")))
client = MongoClient(cfg.get("mongo_uri") or cfg.get("mongodb_uri") or "")
db = client.get_default_database()
doc = db["stock_postmarket_report"].find_one({{"code": "{CODE}", "report_date": "{REPORT_DATE}"}})
if not doc:
    print("MISSING mongo doc")
else:
  slim = {{k: doc.get(k) for k in ["code","report_date","result","change_pct","confidence","summary","vs_premarket"]}}
  print("mongo", json.dumps(slim, ensure_ascii=False, default=str))
  rep = str(doc.get("report") or "")
  print("report_len", len(rep))
  print(rep[:1500])
'''
    print(ssh_run("geegoo-agent", f"python3 <<'PY'\n{mongo_py}\nPY", timeout=60))

    print("\n=== 5) App API /reports/daily stock_postmarket ===")
    api_py = f'''
import json, urllib.request
USER = "{USER}"
BOT = "{BOT_KEY}"
body = json.dumps({{"user_id": USER, "phases": ["stock_postmarket"], "limit_per_phase": 10}}).encode()
req = urllib.request.Request(
    "http://127.0.0.1:3140/reports/daily",
    data=body,
    headers={{"Content-Type": "application/json", "Authorization": "Bearer " + BOT}},
    method="POST",
)
data = json.loads(urllib.request.urlopen(req, timeout=60).read())
rows = (data.get("data") or {{}}).get("stock_postmarket") or []
hit = [r for r in rows if r.get("code") == "{CODE}" and "{REPORT_DATE}" in str(r.get("report_date") or "")]
print("api_code", data.get("code"), "total_postmarket", len(rows), "hit", len(hit))
if hit:
    r = hit[0]
    print(json.dumps({{k: r.get(k) for k in ["code","report_date","result","change_pct","confidence","summary","vs_premarket"]}}, ensure_ascii=False, default=str))
    rep = str(r.get("report") or "")
    print("report_len", len(rep))
    print(rep[:1200])
else:
    print("no matching row for {CODE}")
'''
    print(ssh_run("geegoo-agent", f"python3 <<'PY'\n{api_py}\nPY", timeout=90))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
