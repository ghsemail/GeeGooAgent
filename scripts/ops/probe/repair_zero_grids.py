#!/usr/bin/env python3
"""Rebuild all-zero GRID buy_grid sides from upper/lower/grid_num (招金/小米等)."""
from __future__ import annotations

import json
from pathlib import Path

import paramiko

DEPLOY = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")

REMOTE = r'''
import json, math, os
from bson import ObjectId
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
    out = [0.0] * n
    for i in range(n):
        out[n - 1 - i] = round((lower + step * i) * 100) / 100
    return out


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


def need_repair(buy, sell):
    buy = buy or []
    sell = sell or []
    if len(buy) == 0 and len(sell) == 0:
        return True
    if len(sell) > 0:
        return False
    return len(buy) > 0 and all(v == 0 for v in buy)


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


fixed = []
for info in db.grid_info.find({}, {"bot_id": 1, "buy_grid": 1, "sell_grid": 1, "current_grid": 1, "buy_position": 1}):
    buy = list(info.get("buy_grid") or [])
    sell = list(info.get("sell_grid") or [])
    if not need_repair(buy, sell):
        continue
    bid = info["bot_id"]
    bot = db.grid_bot.find_one({"_id": bid})
    if not bot:
        continue
    rebuilt = rebuild(bot.get("grid") or {}, info.get("current_grid") or 0)
    if not rebuilt:
        print("SKIP no cfg", bid, bot.get("botname"), bot.get("code"))
        continue
    nbuy, nsell, ncur = rebuilt
    order = bot.get("order_size") or {}
    base = float(order.get("base_order_size") or 0)
    scale = float(order.get("safety_orders_volume_scale") or 0) or 2
    set_doc = {"buy_grid": nbuy, "sell_grid": nsell, "current_grid": ncur}
    if base > 0 and nbuy:
        set_doc["buy_position"] = [
            round(base * (scale ** (math.floor(len(nbuy)) - i - 1))) for i in range(len(nbuy))
        ]
    db.grid_info.update_one({"bot_id": bid}, {"$set": set_doc})
    fixed.append({
        "bot_id": str(bid),
        "botname": bot.get("botname"),
        "code": bot.get("code"),
        "before_buy": buy,
        "after": set_doc,
    })

print(json.dumps({"fixed_count": len(fixed), "fixed": fixed}, ensure_ascii=False, indent=2))
'''


def main() -> None:
    cfg = json.loads(DEPLOY.read_text(encoding="utf-8"))
    s = cfg["targets"]["geegoo-bot"]["ssh"]
    c = paramiko.SSHClient()
    c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    c.connect(s["host"], username=s["user"], password=s.get("password"), timeout=30)
    try:
        _, stdout, stderr = c.exec_command(f"python3 - <<'PY'\n{REMOTE}\nPY", timeout=60)
        out = stdout.read().decode("utf-8", errors="replace")
        err = stderr.read().decode("utf-8", errors="replace")
        print(out)
        if err.strip():
            print("STDERR:", err)
    finally:
        c.close()


if __name__ == "__main__":
    main()
