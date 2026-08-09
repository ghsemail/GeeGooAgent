#!/usr/bin/env python3
"""Deep API smoke test using ghsemail user (6366170502d5c175fd586fe8)."""
from __future__ import annotations

import json
from pathlib import Path

import paramiko

USER_ID = "6366170502d5c175fd586fe8"
BOT_KEY = "sk-7a2b3c4d5e6f7g8h9i0j1k2l3m4n5o6p7q8r9s0t1u2v3w4x5y6z7a8b9c0d1e2"
CAT_KEY = "850367bc6d5fe8a4a53f267f5c308ac6d2ec1474d1764fe9"
SIG_KEY = "a76e2d4b4aa8b8eb154f3f2e8feff49a2c34c0f94d012402"
ANA_KEY = "aac157767ebdc8889b83b268852cc8ac09f4f360b67b36d7"
DEPLOY = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")

REMOTE_PY = r'''
import json, urllib.request, sys
USER = "''' + USER_ID + r'''"
BOT = "''' + BOT_KEY + r'''"
CAT = "''' + CAT_KEY + r'''"
SIG = "''' + SIG_KEY + r'''"
ANA = "''' + ANA_KEY + r'''"

def post(url, body, key=None, timeout=60):
    data = json.dumps(body).encode()
    h = {"Content-Type": "application/json"}
    if key:
        h["Authorization"] = f"Bearer {key}"
    req = urllib.request.Request(url, data=data, headers=h, method="POST")
    try:
        with urllib.request.urlopen(req, timeout=timeout) as r:
            raw = r.read()
            try:
                return r.status, json.loads(raw)
            except Exception:
                return r.status, raw[:200]
    except urllib.error.HTTPError as e:
        raw = e.read()
        try:
            return e.code, json.loads(raw)
        except Exception:
            return e.code, raw[:200]

def get(url, key=None, timeout=30):
    h = {}
    if key:
        h["Authorization"] = f"Bearer {key}"
    req = urllib.request.Request(url, headers=h, method="GET")
    try:
        with urllib.request.urlopen(req, timeout=timeout) as r:
            return r.status, json.loads(r.read())
    except urllib.error.HTTPError as e:
        try:
            return e.code, json.loads(e.read())
        except Exception:
            return e.code, str(e)

def show(label, code, data, ok_codes=(100,)):
    ok = False
    if code == 200:
        if isinstance(data, list):
            ok = True
            extra = f"list={len(data)}"
        elif isinstance(data, dict):
            c = data.get("code")
            if c is None or c in ok_codes or c == 100:
                ok = True
            extra = f"code={c} keys={list(data.keys())[:6]}"
        else:
            ok = True
            extra = str(data)[:60]
    else:
        extra = str(data)[:100]
    mark = "OK" if ok else "FAIL"
    print(f"[{mark}] {label} http={code} {extra}")

# load user meta from mongo
from pymongo import MongoClient
uri = open("/home/ubuntu/apps/GeeGooBot/.env").read()
mongo_uri = "mongodb://127.0.0.1:27017"
dbn = "QT_DB"
for line in open("/home/ubuntu/apps/GeeGooBot/.env"):
    if line.startswith("GEEGOO_BOT_MONGO_URI="):
        mongo_uri = line.strip().split("=",1)[1]
    if line.startswith("GEEGOO_BOT_MONGO_DB="):
        dbn = line.strip().split("=",1)[1]
from bson import ObjectId
user = MongoClient(mongo_uri)[dbn]["user"].find_one({"_id": ObjectId(USER)})
version_name = (user or {}).get("product", {}).get("version", "slot")
mcp_token = ((user or {}).get("mcp") or {}).get("token", "")
print("user", USER, "version", version_name, "mcp", bool(mcp_token))

# --- trading_app bot-api ---
for name, body in [
    ("getUserStock", {"user_id": USER, "language": "cn"}),
    ("getUserInfo", {"user_id": USER}),
    ("getNewsStock", {"user_id": USER}),
    ("usedCheck", {"user_id": USER, "name": "testbot"}),
    ("hdgCount", {"user_id": USER}),
    ("getBindingBot", {"user_id": USER}),
]:
    c,d = post(f"http://127.0.0.1:3100/{name}", body, BOT)
    show(name, c, d, ok_codes=(100,101))

# get first stock code
stocks = MongoClient(mongo_uri)[dbn]["user_security"].find({"user_id": ObjectId(USER)}).limit(3)
codes = [s.get("code") for s in stocks if s.get("code")]
print("watchlist codes", codes)
if codes:
    code = codes[0]
    c,d = post("http://127.0.0.1:3100/getCurrentPrice", {"code": code}, BOT)
    show("getCurrentPrice", c, d)
    c,d = post("http://127.0.0.1:3100/getUserReminder", {"user_id": USER, "code": code}, BOT)
    show("getUserReminder", c, d)
    c,d = post("http://127.0.0.1:3100/getUserBot", {"user_id": USER, "code": code}, BOT)
    show("getUserBot", c, d)

# --- agent-api ---
c,d = post("http://127.0.0.1:3110/getUserAgents", {"user_id": USER}, BOT)
show("getUserAgents", c, d, ok_codes=(100,101))

# --- service-api ---
c,d = post("http://127.0.0.1:3140/reports/daily", {"user_id": USER, "limit_per_phase": 3}, BOT)
show("reports/daily", c, d)

# --- catalog ---
c,d = post("http://146.56.225.252:3210/queryVersion", {"name": version_name}, CAT)
show("queryVersion", c, d, ok_codes=(100,))
c,d = post("http://146.56.225.252:3210/getSinglePromptTemplate", {"type": "tech", "user_id": USER}, CAT)
show("getSinglePromptTemplate(tech)", c, d)
c,d = post("http://146.56.225.252:3210/getAttitudePromptList", {"user_id": USER}, CAT)
show("getAttitudePromptList", c, d)
if codes:
    c,d = post("http://146.56.225.252:3210/getAISignal", {"code_list": codes[:3], "month": "2026-08"}, CAT)
    show("getAISignal", c, d, ok_codes=(100,101))

# --- signal-api ---
if codes:
    code = codes[0]
    c,d = post("http://146.56.225.252:3200/getDashboardKline", {"code": code, "language": "cn"}, SIG)
    show("getDashboardKline", c, d)
    c,d = post("http://146.56.225.252:3200/getDashboardSignal", {"code": code, "frequency": "1d", "type": "stock", "signal_index_list": [], "language": "cn"}, SIG)
    show("getDashboardSignal", c, d)
    c,d = post("http://146.56.225.252:3200/getSupportingPrice", {"code": code}, SIG)
    ok = c == 200 and isinstance(d, dict) and "data" in d
    print(f"[{'OK' if ok else 'FAIL'}] getSupportingPrice http={c} stock={d.get('code') if isinstance(d,dict) else d}")

# --- analyze-api ---
c,d = post("http://146.56.225.252:3230/getSingleAnalysisHistory", {"user_id": USER, "type": "single"}, ANA)
show("getSingleAnalysisHistory", c, d)

# --- news ---
news_stocks = MongoClient(mongo_uri)[dbn]["news_security"].find({"user_id": ObjectId(USER)}).limit(5)
news_list = [{"code": s.get("code"), "name": {"init": "t"}} for s in news_stocks if s.get("code")]
print("news watch", [x["code"] for x in news_list])
if news_list:
    hk_us = [x for x in news_list if not (x["code"].endswith(".SZ") or x["code"].endswith(".SH"))]
    cn = [x for x in news_list if x["code"].endswith(".SZ") or x["code"].endswith(".SH")]
    if hk_us:
        c,d = post("http://47.80.14.120:3300/getStockNews", {"stock_list": hk_us[:2]})
        show("getStockNews HK", c, d)
    if cn:
        c,d = post("http://82.157.97.76:3300/getStockNews", {"stock_list": cn[:2]})
        show("getStockNews CN", c, d)

# --- agent BFF with mcp ---
if mcp_token:
    req = urllib.request.Request("http://127.0.0.1:3110/op_agent/v1/metrics/overview",
        headers={"Authorization": f"Bearer {BOT}", "X-MCP-Token": mcp_token, "X-User-Id": USER})
    try:
        with urllib.request.urlopen(req, timeout=30) as r:
            show("op_agent metrics/overview", r.status, json.loads(r.read()))
    except urllib.error.HTTPError as e:
        show("op_agent metrics/overview", e.code, e.read()[:100])
else:
    print("[SKIP] op_agent metrics (no mcp token on user)")
'''

def main() -> int:
    cfg = json.loads(DEPLOY.read_text(encoding="utf-8"))
    ssh = cfg["targets"]["geegoo-bot"]["ssh"]
    c = paramiko.SSHClient()
    c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    c.connect(ssh["host"], username=ssh["user"], password=ssh["password"], timeout=60)
    cmd = f"python3 - <<'PY'\n{REMOTE_PY}\nPY"
    _, o, e = c.exec_command(cmd, timeout=300)
    out = (o.read() + e.read()).decode("utf-8", errors="replace")
    Path(r"D:\Geegoo\GeeGooAgent\scripts\probe_ghsemail_result.txt").write_text(out, encoding="utf-8")
    print(out.encode("ascii", errors="replace").decode())
    c.close()
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
