#!/usr/bin/env python3
"""Fix scheduler health: attitude.switch on report bots + GEEGOO_PG_DSN in agent.env."""
from __future__ import annotations

import json
import re
from pathlib import Path

import paramiko

DEPLOY = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")
cfg = json.loads(DEPLOY.read_text(encoding="utf-8"))

BOT_PY = r'''python3 - <<'PY'
import json, re, urllib.request
from bson import ObjectId
from pymongo import MongoClient

uri = dbn = None
for line in open("/home/ubuntu/apps/GeeGooBot/.env"):
    line = line.strip()
    if line.startswith("GEEGOO_BOT_MONGO_URI="):
        uri = line.split("=", 1)[1]
    if line.startswith("GEEGOO_BOT_MONGO_DB="):
        dbn = line.split("=", 1)[1]
db = MongoClient(uri, serverSelectionTimeoutMS=8000)[dbn or "QT_DB"]
user = db.user.find_one({"username": "ghsemail"})
if not user:
    print("USER_NOT_FOUND"); raise SystemExit(1)
uid = user["_id"]
mcp = (user.get("mcp") or {}).get("mcp_token", "")
print("user_id", str(uid), "mcp", mcp[:10] + "...")

pairs = [
    ("dca_bot", "dca_info"),
    ("grid_bot", "grid_info"),
    ("trade_bot", "trade_info"),
]
updated = 0
for bot_coll, info_coll in pairs:
    if bot_coll not in db.list_collection_names():
        continue
    for doc in db[bot_coll].find({"user_id": uid}):
        info = db[info_coll].find_one({"bot_id": doc["_id"]}) if info_coll in db.list_collection_names() else {}
        att = doc.get("attitude") or {}
        active = bool(info.get("switch", True))
        if att.get("switch") is True:
            print("ok", bot_coll, doc.get("code"), doc.get("botname"))
            continue
        if not active and not att.get("controll_switch"):
            print("skip_inactive", bot_coll, doc.get("code"))
            continue
        db[bot_coll].update_one({"_id": doc["_id"]}, {"$set": {"attitude.switch": True}})
        updated += 1
        print("enabled", bot_coll, doc.get("code"), doc.get("botname"), "switch", active, "ctrl", att.get("controll_switch"))
print("UPDATED", updated)

key = open("/home/ubuntu/apps/GeeGooBot/.env").read()
import re as _re
m = _re.search(r"GEEGOO_BOT_API_KEY=(.+)", key)
api_key = m.group(1).strip() if m else ""
body = json.dumps({"mcp_token": mcp}).encode()
req = urllib.request.Request(
    "http://127.0.0.1:3120/getReportBotCodes", data=body, method="POST",
    headers={"Authorization": "Bearer " + api_key, "Content-Type": "application/json"},
)
with urllib.request.urlopen(req, timeout=15) as r:
    print("getReportBotCodes", r.read().decode()[:800])
PY'''

AGENT_PY = r'''python3 - <<'PY'
import os, re, subprocess

def proc_env(pattern):
    p = subprocess.run(["bash", "-lc", f"pgrep -f '{pattern}' | head -1"], capture_output=True, text=True)
    pid = (p.stdout or "").strip()
    if not pid:
        return {}
    env = {}
    for item in open(f"/proc/{pid}/environ", "rb").read().split(b"\0"):
        if b"=" in item:
            k, v = item.split(b"=", 1)
            env[k.decode()] = v.decode()
    return env

rt = proc_env("agentRuntimeServer")
dsn = rt.get("GEEGOO_PG_DSN", "")
print("runtime_dsn", "set" if dsn else "missing")
path = "/home/ubuntu/.geegoo/agent.env"
lines = open(path, encoding="utf-8").read().splitlines() if os.path.isfile(path) else []

def set_kv(lines, key, val):
    pat = re.compile(rf"^{re.escape(key)}=")
    out, done = [], False
    for ln in lines:
        if pat.match(ln):
            out.append(f'{key}="{val}"'); done = True
        else:
            out.append(ln)
    if not done:
        out.append(f'{key}="{val}"')
    return out

changed = False
for key, val in [("GEEGOO_PG_DSN", dsn), ("GEEGOO_SESSION_STORE", "postgres")]:
    if key == "GEEGOO_PG_DSN" and not val:
        continue
    new = set_kv(lines, key, val)
    if new != lines:
        lines = new; changed = True
        print("set", key)
if changed:
    open(path, "w", encoding="utf-8").write("\n".join(lines) + "\n")
print("env_changed", changed)
PY'''


def run(ssh_cfg, script, title):
    print(f"\n===== {title} =====")
    c = paramiko.SSHClient()
    c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    c.connect(ssh_cfg["host"], username=ssh_cfg["user"], password=ssh_cfg.get("password"), timeout=30)
    _, o, e = c.exec_command(script, timeout=120)
    print(o.read().decode("utf-8", errors="replace"))
    err = e.read().decode("utf-8", errors="replace").strip()
    if err:
        print("ERR:", err[:1500])
    c.close()


def main():
    run(cfg["targets"]["geegoo-bot"]["ssh"], BOT_PY, "attitude.switch")
    run(cfg["targets"]["geegoo-agent"]["ssh"], AGENT_PY, "agent.env PG")


if __name__ == "__main__":
    main()
