#!/usr/bin/env python3
"""When did grid logs start writing null buy/sell; sample bad vs good."""
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

out = {}
for hid, name in [
    ("6908395ac968cf04c9115041", "zhaojin"),
    ("67f3d77d63b48cbc08a00b77", "xiaomi"),
    ("6781cb8309a2189f26d8866e", "tencent"),
]:
    oid = ObjectId(hid)
    first_null = db.grid_log.find_one(
        {"bot_id": oid, "log.buy_grid": None},
        sort=[("time", 1)],
    )
    last_good = db.grid_log.find_one(
        {"bot_id": oid, "log.buy_grid.0": {"$exists": True}},
        sort=[("time", -1)],
    )
    # also empty array
    empty = db.grid_log.count_documents({"bot_id": oid, "log.buy_grid": []})
    null_n = db.grid_log.count_documents({"bot_id": oid, "log.buy_grid": None})
    # zero arrays
    sample_zero = db.grid_log.find_one({"bot_id": oid, "log.buy_grid": [0, 0, 0, 0, 0]}, sort=[("time", -1)])
    # API-visible last 100: how many null
    last100 = list(db.grid_log.find({"bot_id": oid}).sort("time", -1).limit(100))
    null100 = sum(1 for d in last100 if (d.get("log") or {}).get("buy_grid") is None)
    empty100 = sum(1 for d in last100 if (d.get("log") or {}).get("buy_grid") == [])
    good100 = sum(1 for d in last100 if isinstance((d.get("log") or {}).get("buy_grid"), list) and (d.get("log") or {}).get("buy_grid"))

    def slim(d):
        if not d:
            return None
        log = d.get("log") or {}
        return {
            "time": str(d.get("time")),
            "buy": log.get("buy_grid"),
            "sell": log.get("sell_grid"),
            "cur": log.get("current_grid"),
            "buy_pos": log.get("buy_position"),
        }

    out[name] = {
        "null_total": null_n,
        "empty_array_total": empty,
        "last100_null": null100,
        "last100_empty": empty100,
        "last100_good": good100,
        "first_null": slim(first_null),
        "last_good": slim(last_good),
        "sample_zero": slim(sample_zero),
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
        _, stdout, stderr = c.exec_command(f"python3 - <<'PY'\n{REMOTE}\nPY", timeout=120)
        print(stdout.read().decode("utf-8", errors="replace"))
        err = stderr.read().decode("utf-8", errors="replace")
        if err.strip():
            print("STDERR:", err)
    finally:
        c.close()


if __name__ == "__main__":
    main()
