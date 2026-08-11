#!/usr/bin/env python3
from __future__ import annotations

import json
from pathlib import Path

import paramiko

DEPLOY = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")
REMOTE = r'''
import os, json
from bson import ObjectId
from pymongo import MongoClient
os.chdir("/home/ubuntu/apps/GeeGooBot")
for line in open(".env"):
    line = line.strip()
    if line and not line.startswith("#") and "=" in line:
        k, v = line.split("=", 1)
        os.environ.setdefault(k.strip(), v.strip().strip('"').strip("'"))
db = MongoClient(os.environ["GEEGOO_BOT_MONGO_URI"])[os.environ["GEEGOO_BOT_MONGO_DB"]]
for bid in ["6908395ac968cf04c9115041", "67f3d77d63b48cbc08a00b77"]:
    oid = ObjectId(bid)
    b = db.grid_bot.find_one({"_id": oid}, {"botname": 1, "code": 1, "grid": 1})
    info = db.grid_info.find_one(
        {"bot_id": oid}, {"buy_grid": 1, "sell_grid": 1, "current_grid": 1, "buy_position": 1}
    )
    print(json.dumps({"bot": b, "info": info}, default=str, ensure_ascii=False))
n = 0
samples = []
for info in db.grid_info.find({}, {"bot_id": 1, "buy_grid": 1, "sell_grid": 1}):
    buy = list(info.get("buy_grid") or [])
    sell = list(info.get("sell_grid") or [])
    if (not buy and not sell) or (buy and not sell and all(v == 0 for v in buy)):
        n += 1
        if len(samples) < 5:
            samples.append({"bot_id": str(info.get("bot_id")), "buy": buy, "sell": sell})
print(json.dumps({"suspicious": n, "samples": samples}, ensure_ascii=False))
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
