#!/usr/bin/env python3
"""Inspect raw grid_log documents for missing buy/sell grids."""
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

oid = ObjectId("6908395ac968cf04c9115041")
# raw recent
recent = list(db.grid_log.find({"bot_id": oid}).sort("time", -1).limit(3))
# stats
pipeline = [
    {"$match": {"bot_id": oid}},
    {"$project": {
        "has_buy": {"$cond": [{"$isArray": "$log.buy_grid"}, {"$gt": [{"$size": "$log.buy_grid"}, 0]}, False]},
        "has_sell": {"$cond": [{"$isArray": "$log.sell_grid"}, {"$gt": [{"$size": "$log.sell_grid"}, 0]}, False]},
        "buy_null": {"$eq": [{"$type": "$log.buy_grid"}, "null"]},
        "buy_missing": {"$eq": [{"$type": "$log.buy_grid"}, "missing"]},
    }},
    {"$group": {
        "_id": None,
        "total": {"$sum": 1},
        "has_buy": {"$sum": {"$cond": ["$has_buy", 1, 0]}},
        "has_sell": {"$sum": {"$cond": ["$has_sell", 1, 0]}},
        "buy_null": {"$sum": {"$cond": ["$buy_null", 1, 0]}},
        "buy_missing": {"$sum": {"$cond": ["$buy_missing", 1, 0]}},
    }},
]
stats = list(db.grid_log.aggregate(pipeline))
# find one with buy present and one without
with_buy = db.grid_log.find_one({"bot_id": oid, "log.buy_grid.0": {"$exists": True}}, sort=[("time", -1)])
without = db.grid_log.find_one({"bot_id": oid, "$or": [{"log.buy_grid": None}, {"log.buy_grid": {"$exists": False}}, {"log.buy_grid": []}]}, sort=[("time", -1)])

def slim(d):
    if not d:
        return None
    log = d.get("log") or {}
    return {
        "time": str(d.get("time")),
        "log_keys": sorted(list(log.keys())),
        "buy_grid": log.get("buy_grid"),
        "sell_grid": log.get("sell_grid"),
        "current_grid": log.get("current_grid"),
        "next_opt": log.get("next_opt"),
        "buy_type": type(log.get("buy_grid")).__name__,
    }

print(json.dumps({
    "stats": stats,
    "recent_slim": [slim(x) for x in recent],
    "latest_with_buy": slim(with_buy),
    "latest_without_buy": slim(without),
}, ensure_ascii=False, indent=2, default=str))
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
