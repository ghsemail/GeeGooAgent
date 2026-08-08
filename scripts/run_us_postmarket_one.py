#!/usr/bin/env python3
"""Run postmarket_stock for one US bot (background on agent server) and verify."""
from __future__ import annotations

import json
import time
from datetime import datetime
from pathlib import Path

import paramiko

DEPLOY = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")
USER = "6366170502d5c175fd586fe8"
BOT_KEY = "sk-7a2b3c4d5e6f7g8h9i0j1k2l3m4n5o6p7q8r9s0t1u2v3w4x5y6z7a8b9c0d1e2"
PREFER = ("TSLA.US", "AAPL.US", "NVDA.US", "SPCX.US")


def ssh(cmd: str, timeout: int = 120) -> str:
    cfg = json.loads(DEPLOY.read_text(encoding="utf-8"))
    s = cfg["targets"]["geegoo-agent"]["ssh"]
    c = paramiko.SSHClient()
    c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    c.connect(s["host"], username=s["user"], password=s.get("password"), timeout=30)
    _, o, e = c.exec_command(cmd, timeout=timeout)
    out = (o.read() + e.read()).decode("utf-8", errors="replace")
    c.close()
    return out


REMOTE_PROBE = r'''
import json, os, urllib.request
from datetime import datetime, timedelta
cfg = json.load(open(os.path.expanduser("~/.geegoo/config.json")))
tok = cfg.get("mcp_token", "")
key = cfg.get("geegoo_api_key") or cfg.get("api_key", "")
MCP = "http://118.195.135.97:3120"
PREFER = ["TSLA.US","AAPL.US","NVDA.US","SPCX.US"]

def post(path, payload):
    req = urllib.request.Request(
        MCP + path,
        data=json.dumps(payload).encode(),
        headers={"Content-Type": "application/json", "Authorization": "Bearer " + key},
        method="POST",
    )
    with urllib.request.urlopen(req, timeout=90) as r:
        return json.loads(r.read().decode())

r = post("/getReportBotCodes", {"mcp_token": tok})
bots = r.get("data") or []
us = [b for b in bots if str(b.get("code", "")).upper().endswith(".US")]
pick = None
for pref in PREFER:
    for b in us:
        if str(b.get("code", "")).upper() == pref:
            pick = b
            break
    if pick:
        break
if not pick and us:
    pick = us[0]
if not pick:
    print(json.dumps({"error": "no_us_bot", "total": len(bots)}))
    raise SystemExit(2)
code = pick["code"]
report_date = None
for delta in range(0, 10):
    d = (datetime.now() - timedelta(days=delta)).strftime("%Y-%m-%d")
    tr = post("/checkTradingDay", {"mcp_token": tok, "code": code, "date": d})
    if (tr.get("data") or {}).get("is_trading_day"):
        report_date = d
        break
if not report_date:
    report_date = (datetime.now() - timedelta(days=1)).strftime("%Y-%m-%d")
# delete existing postmarket for this code/date only
rep = post("/getStockDailyReports", {"mcp_token": tok, "code": code, "report_date": report_date})
for row in (rep.get("data") or {}).get("stock_postmarket") or []:
    rid = str(row.get("report_id") or row.get("_id") or "")
    if rid:
        post("/deleteStockPostmarketReport", {"mcp_token": tok, "report_id": rid})
print(json.dumps({"code": code, "stock_name": pick.get("stock_name"), "bot_id": pick.get("bot_id"),
                  "report_date": report_date, "us_bot_count": len(us)}, ensure_ascii=False))
'''


def main() -> int:
    print("=== probe US bot ===")
    probe_out = ssh(f"python3 <<'PY'\n{REMOTE_PROBE}\nPY", timeout=120).strip()
    print(probe_out)
    meta = json.loads(probe_out.splitlines()[-1])
    if meta.get("error"):
        print("ERROR:", meta)
        return 2
    code = meta["code"]
    report_date = meta["report_date"]
    print(f"target {code} ({meta.get('stock_name')}) date={report_date} us_bots={meta.get('us_bot_count')}")

    # Run only US market; if multiple US bots, mark others skipped via env hack:
    # temporarily restrict by patching working after phase A is not available — use market US.
    # When multiple US bots exist, pre-mark non-target bots as skipped in a wrapper script.
    run_wrapper = f'''
import json, os, subprocess, time, glob
code = {json.dumps(code)}
report_date = {json.dumps(report_date)}
log_path = os.path.expanduser("~/.geegoo/data/postmarket-us-one.log")
pid_path = log_path + ".pid"
cmd = [
    os.path.expanduser("~/.geegoo/bin/geegoo"), "run",
    "--config", os.path.expanduser("~/.geegoo/config.json"),
    "--market", "US",
    "--report-date", report_date,
    "postmarket_stock",
]
env = os.environ.copy()
proc = subprocess.Popen(cmd, stdout=open(log_path, "w"), stderr=subprocess.STDOUT, env=env)
open(pid_path, "w").write(str(proc.pid))
print("started", proc.pid, "log", log_path)
'''
    print("\n=== start postmarket_stock (US) background ===")
    start_out = ssh(f"python3 <<'PY'\n{run_wrapper}\nPY", timeout=60)
    print(start_out.strip())

    print("\n=== poll until done (max 25 min) ===")
    deadline = time.time() + 25 * 60
    last_line = ""
    while time.time() < deadline:
        poll = ssh(
            f"""python3 <<'PY'
import json, os, glob, time
code = {json.dumps(code)}
report_date = {json.dumps(report_date)}
log = os.path.expanduser('~/.geegoo/data/postmarket-us-one.log')
pidf = log + '.pid'
running = False
if os.path.exists(pidf):
    pid = int(open(pidf).read().strip() or '0')
    try:
        os.kill(pid, 0)
        running = True
    except OSError:
        running = False
runs = sorted(glob.glob(os.path.expanduser('~/.geegoo/data/working/*.json')), key=os.path.getmtime, reverse=True)
ws = {{}}
skill = phase = ''
for p in runs[:5]:
    d = json.load(open(p))
    if d.get('skill') != 'postmarket_stock':
        continue
    if str(d.get('report_date') or '')[:10] != report_date:
        continue
    ws = (d.get('stocks') or {{}}).get(code) or {{}}
    skill = d.get('skill','')
    phase = d.get('phase','')
    break
tail = ''
if os.path.exists(log):
    tail = open(log).read()[-800:]
print(json.dumps({{'running': running, 'phase': phase, 'status': ws.get('status'), 'change_pct': ws.get('change_pct'), 'report_id': ws.get('report_id'), 'tail': tail}}, ensure_ascii=False))
PY""",
            timeout=90,
        )
        line = poll.strip().splitlines()[-1] if poll.strip() else "{}"
        try:
            st = json.loads(line)
        except json.JSONDecodeError:
            st = {}
        if line != last_line:
            print(
                f"  running={st.get('running')} phase={st.get('phase')} "
                f"status={st.get('status')} change_pct={st.get('change_pct')} report_id={st.get('report_id')}"
            )
            last_line = line
        if st.get("status") in ("reported", "skipped", "failed") and not st.get("running"):
            break
        if not st.get("running") and st.get("status") in ("reported", "failed"):
            break
        time.sleep(30)

    print("\n=== log tail ===")
    print(ssh("tail -n 40 ~/.geegoo/data/postmarket-us-one.log 2>/dev/null || echo NO_LOG", timeout=30))

    print("\n=== API verify ===")
    verify = f'''python3 <<'PY'
import json, urllib.request
USER = "{USER}"
BOT = "{BOT_KEY}"
code = {json.dumps(code)}
report_date = {json.dumps(report_date)}
body = json.dumps({{"user_id": USER, "phases": ["stock_postmarket"], "limit_per_phase": 30}}).encode()
req = urllib.request.Request(
    "http://118.195.135.97:3140/reports/daily",
    data=body,
    headers={{"Content-Type": "application/json", "Authorization": "Bearer " + BOT}},
    method="POST",
)
api = json.loads(urllib.request.urlopen(req, timeout=60).read())
rows = (api.get("data") or {{}}).get("stock_postmarket") or []
hit = [x for x in rows if x.get("code") == code and report_date in str(x.get("report_date") or "")]
print("api_hit", len(hit))
if hit:
    r0 = hit[0]
    print("report_id", r0.get("report_id"))
    print("change_pct", r0.get("change_pct"), "session_bias", r0.get("session_bias"))
    print("report_len", len(str(r0.get("report") or "")))
    print("summary", str(r0.get("summary") or "")[:200])
else:
    us = [(x.get("code"), x.get("report_date")) for x in rows if str(x.get("code","")).endswith(".US")][:5]
    print("recent_us", us)
PY'''
    print(ssh(verify, timeout=90))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
