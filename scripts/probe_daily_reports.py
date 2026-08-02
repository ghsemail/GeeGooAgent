#!/usr/bin/env python3
"""Test pre_market / post_market daily reports for ghsemail user."""
from __future__ import annotations

import json
from pathlib import Path

import paramiko

USER = "6366170502d5c175fd586fe8"
BOT_KEY = "sk-7a2b3c4d5e6f7g8h9i0j1k2l3m4n5o6p7q8r9s0t1u2v3w4x5y6z7a8b9c0d1e2"
DEPLOY = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")

REMOTE = f'''
import json, urllib.request, traceback
from bson import ObjectId
from pymongo import MongoClient

USER = "{USER}"
BOT = "{BOT_KEY}"

def post(url, body, key=None, timeout=60):
    data = json.dumps(body).encode()
    h = {{"Content-Type": "application/json"}}
    if key:
        h["Authorization"] = f"Bearer {{key}}"
    req = urllib.request.Request(url, data=data, headers=h, method="POST")
    try:
        with urllib.request.urlopen(req, timeout=timeout) as r:
            raw = r.read()
            try:
                return r.status, json.loads(raw)
            except Exception:
                return r.status, {{"raw": raw[:500].decode("utf-8", "replace")}}
    except urllib.error.HTTPError as e:
        raw = e.read()
        try:
            return e.code, json.loads(raw)
        except Exception:
            return e.code, {{"raw": raw[:500].decode("utf-8", "replace")}}

def show(label, code, data):
    ok = code == 200 and isinstance(data, dict) and data.get("code") == 100
    mark = "OK" if ok else "FAIL"
    extra = ""
    if isinstance(data, dict):
        d = data.get("data")
        if isinstance(d, dict):
            pre = d.get("pre_market") or []
            intra = d.get("intraday") or []
            postm = d.get("post_market") or []
            extra = f" pre={{len(pre)}} intra={{len(intra)}} post={{len(postm)}}"
            if pre:
                sample = pre[0]
                extra += f" pre0={{list(sample.keys())[:8]}}"
        else:
            extra = f" keys={{list(data.keys())[:6]}}"
    print(f"[{{mark}}] {{label}} http={{code}}{{extra}} msg={{data.get('message') if isinstance(data,dict) else data}}")

# --- service-api reports/daily ---
for label, body in [
    ("reports/daily all", {{"user_id": USER, "limit_per_phase": 5}}),
    ("reports/daily pre", {{"user_id": USER, "phases": ["pre_market"], "limit_per_phase": 5}}),
    ("reports/daily post", {{"user_id": USER, "phases": ["post_market"], "limit_per_phase": 5}}),
    ("reports/daily intraday", {{"user_id": USER, "phases": ["intraday"], "limit_per_phase": 5}}),
]:
    c,d = post("http://127.0.0.1:3140/reports/daily", body, BOT)
    show(label, c, d)

# --- MCP premarket endpoints (need mcp token) ---
uri = open("/home/ubuntu/apps/GeeGooBot/.env").read()
mongo_uri = "mongodb://127.0.0.1:27017"
dbn = "QT_DB"
for line in open("/home/ubuntu/apps/GeeGooBot/.env"):
    if line.startswith("GEEGOO_BOT_MONGO_URI="):
        mongo_uri = line.strip().split("=",1)[1]
    if line.startswith("GEEGOO_BOT_MONGO_DB="):
        dbn = line.strip().split("=",1)[1]
user = MongoClient(mongo_uri)[dbn]["user"].find_one({{"_id": ObjectId(USER)}})
mcp_token = ((user or {{}}).get("mcp") or {{}}).get("token", "")
print("mcp_token", bool(mcp_token))

if mcp_token:
    for path, body in [
        ("getReportBotCodes", {{"mcp_token": mcp_token}}),
        ("getPreMarketReports", {{"mcp_token": mcp_token, "code": "07552.HK", "period": "daily"}}),
        ("getStockDailyReports", {{"mcp_token": mcp_token, "code": "000858.SZ"}}),
    ]:
        c,d = post(f"http://127.0.0.1:3120/{{path}}", body, BOT)
        ok = c == 200 and isinstance(d, dict) and d.get("code") in (100, None)
        if isinstance(d, dict) and d.get("code") == 100:
            ok = True
        print(f"[{{'OK' if ok else 'FAIL'}}] mcp {{path}} http={{c}} code={{d.get('code') if isinstance(d,dict) else d}} keys={{list(d.keys())[:6] if isinstance(d,dict) else d}}")
else:
    print("[SKIP] mcp premarket tools (no token)")

# --- mongo counts ---
db = MongoClient(mongo_uri)[dbn]
uid = ObjectId(USER)
for coll in ["pre_market_report", "post_market_report", "intraday_report", "pre_market_reports", "post_market_reports"]:
    try:
        n = db[coll].count_documents({{"user_id": uid}})
        if n:
            print(f"mongo {{coll}} count={{n}}")
            row = db[coll].find_one({{"user_id": uid}}, sort=[("updated_at", -1)])
            if row:
                print(f"  latest keys={{list(row.keys())[:12]}} code={{row.get('code')}} updated={{row.get('updated_at')}}")
    except Exception as e:
        pass

# service log tail
import subprocess
out = subprocess.run("tail -n 30 /home/ubuntu/apps/GeeGooBot/service-api.out 2>/dev/null | grep -iE 'report|error|panic' || tail -n 8 /home/ubuntu/apps/GeeGooBot/service-api.out", shell=True, capture_output=True, text=True)
print('--- service-api.out (tail) ---')
print(out.stdout[-1500:])
'''


def main() -> int:
    cfg = json.loads(DEPLOY.read_text(encoding="utf-8"))
    ssh = cfg["targets"]["geegoo-bot"]["ssh"]
    c = paramiko.SSHClient()
    c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    c.connect(ssh["host"], username=ssh["user"], password=ssh["password"], timeout=60)
    _, o, e = c.exec_command(f"python3 - <<'PY'\n{REMOTE}\nPY", timeout=120)
    out = (o.read() + e.read()).decode("utf-8", errors="replace")
    Path(r"D:\Geegoo\GeeGooAgent\scripts\probe_daily_reports_result.txt").write_text(out, encoding="utf-8")
    print(out.encode("ascii", errors="replace").decode())
    c.close()
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
