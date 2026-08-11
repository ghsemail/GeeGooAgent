#!/usr/bin/env python3
"""Compare grid_info current vs latest grid_log current for key bots."""
from __future__ import annotations

import json
from pathlib import Path

import paramiko

DEPLOY = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")
REMOTE = r'''
import json, os
from bson import ObjectId
from pymongo import MongoClient

os.chdir("/home/ubuntu/apps/GeeGooBot")
for line in open(".env"):
    line = line.strip()
    if line and not line.startswith("#") and "=" in line:
        k, v = line.split("=", 1)
        os.environ.setdefault(k.strip(), v.strip().strip('"').strip("'"))
db = MongoClient(os.environ["GEEGOO_BOT_MONGO_URI"])[os.environ["GEEGOO_BOT_MONGO_DB"]]

ids = [
    ("6781cb8309a2189f26d8866e", "tencent"),
    ("67f3d77d63b48cbc08a00b77", "xiaomi"),
    ("6908395ac968cf04c9115041", "zhaojin"),
]
out = []
for hid, name in ids:
    oid = ObjectId(hid)
    bot = db.grid_bot.find_one({"_id": oid}, {"botname": 1, "code": 1, "grid": 1})
    info = db.grid_info.find_one({"bot_id": oid}, {"buy_grid": 1, "sell_grid": 1, "current_grid": 1, "switch": 1})
    log = db.grid_log.find_one({"bot_id": oid}, sort=[("time", -1)])
    logbody = (log or {}).get("log") or {}
    out.append({
        "name": name,
        "code": (bot or {}).get("code"),
        "card": {
            "buy": (info or {}).get("buy_grid"),
            "cur": (info or {}).get("current_grid"),
            "sell": (info or {}).get("sell_grid"),
        },
        "latest_log": {
            "time": str((log or {}).get("time")),
            "buy": logbody.get("buy_grid"),
            "cur": logbody.get("current_grid"),
            "sell": logbody.get("sell_grid"),
            "next_opt": logbody.get("next_opt"),
            "price": logbody.get("current_price"),
        },
        "match_cur": (info or {}).get("current_grid") == logbody.get("current_grid"),
    })
print(json.dumps(out, ensure_ascii=False, indent=2, default=str))
'''


def main() -> None:
    cfg = json.loads(DEPLOY.read_text(encoding="utf-8"))
    s = cfg["targets"]["geegoo-bot"]["ssh"]
    c = paramiko.SSHClient()
    c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    c.connect(s["host"], username=s["user"], password=s.get("password"), timeout=30)
    try:
        _, stdout, stderr = c.exec_command(f"python3 - <<'PY'\n{REMOTE}\nPY", timeout=60)
        print(stdout.read().decode("utf-8", errors="replace"))
        err = stderr.read().decode("utf-8", errors="replace")
        if err.strip():
            print("STDERR:", err)
    finally:
        c.close()


if __name__ == "__main__":
    main()
