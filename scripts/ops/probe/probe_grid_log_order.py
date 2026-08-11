#!/usr/bin/env python3
"""Sample recent GRID log buy/sell order for held bots."""
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
    "6781cb8309a2189f26d8866e",
    "67f3d77d63b48cbc08a00b77",
    "6908395ac968cf04c9115041",
]
out = {}
for hid in ids:
    oid = ObjectId(hid)
    bot = db.grid_bot.find_one({"_id": oid}, {"botname": 1, "code": 1})
    logs = list(db.grid_log.find({"bot_id": oid}, {"time": 1, "log.buy_grid": 1, "log.sell_grid": 1, "log.current_grid": 1, "log.next_opt": 1}).sort("time", -1).limit(8))
    bad = 0
    total = db.grid_log.count_documents({"bot_id": oid})
    for doc in db.grid_log.find({"bot_id": oid}, {"log.buy_grid": 1, "log.sell_grid": 1}):
        buy = list((doc.get("log") or {}).get("buy_grid") or [])
        sell = list((doc.get("log") or {}).get("sell_grid") or [])
        corrupt = False
        for i in range(1, len(buy)):
            if float(buy[i]) < float(buy[i-1]):
                corrupt = True
        for i in range(1, len(sell)):
            if float(sell[i]) > float(sell[i-1]):
                corrupt = True
        if corrupt:
            bad += 1
    out[hid] = {
        "botname": (bot or {}).get("botname"),
        "code": (bot or {}).get("code"),
        "total_logs": total,
        "corrupt_order_logs": bad,
        "recent": [
            {
                "time": str(d.get("time")),
                "next_opt": (d.get("log") or {}).get("next_opt"),
                "buy": (d.get("log") or {}).get("buy_grid"),
                "cur": (d.get("log") or {}).get("current_grid"),
                "sell": (d.get("log") or {}).get("sell_grid"),
            }
            for d in logs
        ],
    }
print(json.dumps(out, ensure_ascii=False, indent=2, default=str))
'''


def main() -> None:
    cfg = json.loads(DEPLOY.read_text(encoding="utf-8"))
    s = cfg["targets"]["geegoo-bot"]["ssh"]
    c = paramiko.SSHClient()
    c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    c.connect(s["host"], username=s["user"], password=s.get("password"), timeout=30)
    try:
        _, stdout, stderr = c.exec_command(f"python3 - <<'PY'\n{REMOTE}\nPY", timeout=90)
        print(stdout.read().decode("utf-8", errors="replace"))
        err = stderr.read().decode("utf-8", errors="replace")
        if err.strip():
            print("STDERR:", err)
    finally:
        c.close()


if __name__ == "__main__":
    main()
