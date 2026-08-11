#!/usr/bin/env python3
"""Re-anchor mismatched GRID cards from live quote (not midpoint)."""
from __future__ import annotations

import json
import urllib.request
from pathlib import Path

import paramiko

DEPLOY = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")
REMOTE = r'''
import json, math, os, urllib.request
from bson import ObjectId
from pymongo import MongoClient

os.chdir("/home/ubuntu/apps/GeeGooBot")
for line in open(".env"):
    line = line.strip()
    if line and not line.startswith("#") and "=" in line:
        k, v = line.split("=", 1)
        os.environ.setdefault(k.strip(), v.strip().strip('"').strip("'"))

db = MongoClient(os.environ["GEEGOO_BOT_MONGO_URI"])[os.environ["GEEGOO_BOT_MONGO_DB"]]
base = (os.environ.get("GEEGOO_DATA_HTTP_URL") or "").rstrip("/")
token = os.environ.get("GEEGOO_DATA_SERVICE_TOKEN") or ""


def quote(code):
    req = urllib.request.Request(
        base + "/v1/market/quote",
        data=json.dumps({"code": code}).encode(),
        headers={"Content-Type": "application/json", "Authorization": f"Bearer {token}"},
        method="POST",
    )
    with urllib.request.urlopen(req, timeout=15) as r:
        body = json.loads(r.read().decode())
    d = body.get("data") if isinstance(body.get("data"), dict) else body
    return float(d.get("price") or 0)


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


targets = [
    ObjectId("67f3d77d63b48cbc08a00b77"),
    ObjectId("6908395ac968cf04c9115041"),
]
out = []
for oid in targets:
    bot = db.grid_bot.find_one({"_id": oid})
    info = db.grid_info.find_one({"bot_id": oid})
    code = bot.get("code")
    price = quote(code)
    grid = bot.get("grid") or {}
    upper = float(grid.get("upper_limit_price") or 0)
    lower = float(grid.get("lower_limit_price") or 0)
    num = int(grid.get("grid_num") or 0)
    buy, sell, cur = split_grids(linspace(upper, lower, num), price)
    order = bot.get("order_size") or {}
    base_sz = float(order.get("base_order_size") or 0)
    scale = float(order.get("safety_orders_volume_scale") or 0) or 2
    set_doc = {"buy_grid": buy, "sell_grid": sell, "current_grid": cur}
    if base_sz > 0 and buy:
        set_doc["buy_position"] = [
            round(base_sz * (scale ** (math.floor(len(buy)) - i - 1))) for i in range(len(buy))
        ]
    before = {
        "buy": (info or {}).get("buy_grid"),
        "cur": (info or {}).get("current_grid"),
        "sell": (info or {}).get("sell_grid"),
    }
    db.grid_info.update_one({"bot_id": oid}, {"$set": set_doc})
    # also align latest log snapshot so UI matches immediately
    latest = db.grid_log.find_one({"bot_id": oid}, sort=[("time", -1)])
    if latest:
        db.grid_log.update_one(
            {"_id": latest["_id"]},
            {"$set": {
                "log.buy_grid": buy,
                "log.sell_grid": sell,
                "log.current_grid": cur,
                "log.current_price": price,
            }},
        )
    out.append({
        "botname": bot.get("botname"),
        "code": code,
        "price": price,
        "before": before,
        "after": set_doc,
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
        _, stdout, stderr = c.exec_command(f"python3 - <<'PY'\n{REMOTE}\nPY", timeout=90)
        print(stdout.read().decode("utf-8", errors="replace"))
        err = stderr.read().decode("utf-8", errors="replace")
        if err.strip():
            print("STDERR:", err)
    finally:
        c.close()


if __name__ == "__main__":
    main()
