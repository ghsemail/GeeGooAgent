#!/usr/bin/env python3
"""Backfill null buy/sell in grid_log from grid cfg + log.current_grid."""
from __future__ import annotations

import json
from pathlib import Path

import paramiko

DEPLOY = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")
REMOTE = r'''
import json, os
from pymongo import MongoClient

os.chdir("/home/ubuntu/apps/GeeGooBot")
for line in open(".env"):
    line = line.strip()
    if line and not line.startswith("#") and "=" in line:
        k, v = line.split("=", 1)
        os.environ.setdefault(k.strip(), v.strip().strip('"').strip("'"))
db = MongoClient(os.environ["GEEGOO_BOT_MONGO_URI"])[os.environ["GEEGOO_BOT_MONGO_DB"]]


def linspace(upper, lower, n):
    if n < 2:
        n = 2
    step = (upper - lower) / (n - 1)
    return [round((lower + step * i) * 100) / 100 for i in range(n)]


def split_grids(grid_list, price):
    buy, sell = [], []
    for g in grid_list:
        (buy if price >= g else sell).append(g)
    current = 0.0
    if buy and sell:
        if price - buy[-1] >= sell[0] - price:
            current = sell[0]
            sell = sell[1:]
        else:
            current = buy[-1]
            buy = buy[:-1]
    elif not buy and sell:
        current = sell[0]
        sell = sell[1:]
    elif buy:
        current = buy[-1]
        buy = buy[:-1]
    sell = list(reversed(sell))
    return buy, sell, current


def rebuild(grid_cfg, current_grid):
    if not grid_cfg:
        return None
    upper = float(grid_cfg.get("upper_limit_price") or 0)
    lower = float(grid_cfg.get("lower_limit_price") or 0)
    num = int(grid_cfg.get("grid_num") or 0)
    if upper <= 0 or lower <= 0 or upper <= lower:
        return None
    if num < 2:
        num = 6
    anchor = float(current_grid or 0)
    if anchor <= 0:
        anchor = (upper + lower) / 2
    return split_grids(linspace(upper, lower, num), anchor)


fixed = 0
bots = 0
for bot in db.grid_bot.find({}, {"grid": 1, "botname": 1, "code": 1}):
    q = {
        "bot_id": bot["_id"],
        "$or": [
            {"log.buy_grid": None},
            {"log.sell_grid": None},
            {"log.buy_grid": []},
            {"log.sell_grid": []},
        ],
    }
    n = db.grid_log.count_documents(q)
    if n == 0:
        continue
    bots += 1
    for doc in db.grid_log.find(q, {"log": 1}):
        log = doc.get("log") or {}
        buy = log.get("buy_grid")
        sell = log.get("sell_grid")
        if isinstance(buy, list) and buy and isinstance(sell, list) and sell:
            continue
        rebuilt = rebuild(bot.get("grid") or {}, log.get("current_grid") or 0)
        if not rebuilt:
            continue
        nbuy, nsell, ncur = rebuilt
        cur = log.get("current_grid") or ncur
        db.grid_log.update_one(
            {"_id": doc["_id"]},
            {"$set": {"log.buy_grid": nbuy, "log.sell_grid": nsell, "log.current_grid": cur}},
        )
        fixed += 1

print(json.dumps({"bots_touched": bots, "logs_fixed": fixed}, ensure_ascii=False))
'''


def main() -> None:
    cfg = json.loads(DEPLOY.read_text(encoding="utf-8"))
    s = cfg["targets"]["geegoo-bot"]["ssh"]
    c = paramiko.SSHClient()
    c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    c.connect(s["host"], username=s["user"], password=s.get("password"), timeout=30)
    try:
        _, stdout, stderr = c.exec_command(f"python3 - <<'PY'\n{REMOTE}\nPY", timeout=180)
        print(stdout.read().decode("utf-8", errors="replace"))
        err = stderr.read().decode("utf-8", errors="replace")
        if err.strip():
            print("STDERR:", err)
    finally:
        c.close()


if __name__ == "__main__":
    main()
