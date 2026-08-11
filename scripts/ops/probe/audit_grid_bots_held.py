#!/usr/bin/env python3
"""Deep audit: GRID bots with real holdings (position.qty / can_sell_qty)."""
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


def f(v, d=0.0):
    try:
        return float(v)
    except Exception:
        return d


def order_corrupt(buy, sell):
    for i in range(1, len(buy or [])):
        if f(buy[i]) < f(buy[i - 1]):
            return "buy_not_ascending"
    for i in range(1, len(sell or [])):
        if f(sell[i]) > f(sell[i - 1]):
            return "sell_not_descending"
    return None


held = []
planned_only = []
all_switch_on = []

for bot in db.grid_bot.find({}):
    info = db.grid_info.find_one({"bot_id": bot["_id"]}) or {}
    pos = info.get("position") if isinstance(info.get("position"), dict) else {}
    qty = f(pos.get("qty"))
    can_sell = f(pos.get("can_sell_qty"))
    buy = list(info.get("buy_grid") or [])
    sell = list(info.get("sell_grid") or [])
    bp = list(info.get("buy_position") or [])
    cur = f(info.get("current_grid"))
    grid = bot.get("grid") or {}
    upper, lower = f(grid.get("upper_limit_price")), f(grid.get("lower_limit_price"))
    issues = []
    oc = order_corrupt(buy, sell)
    if oc:
        issues.append(oc)
    if not buy and not sell:
        issues.append("empty_grids")
    if buy and all(f(x) == 0 for x in buy) and not sell:
        issues.append("all_zero_buy")
    if cur <= 0:
        issues.append("current_le_0")
    if len(bp) != len(buy):
        issues.append(f"buy_pos_len:{len(bp)}!={len(buy)}")
    if upper > 0 and lower > 0:
        for g in buy + sell + ([cur] if cur > 0 else []):
            if g < lower - 1e-6 or g > upper + 1e-6:
                issues.append(f"out_of_range:{g}")
                break
        if buy and cur and f(buy[-1]) > cur + 1e-6:
            issues.append("nearest_buy_gt_current")
        if sell and cur and f(sell[-1]) < cur - 1e-6:
            issues.append("nearest_sell_lt_current")
    # held shares but no sell ladder: may be stuck unable to scale out (warn)
    if (qty > 0 or can_sell > 0) and not sell and cur > 0:
        if upper > 0 and cur >= upper - 1e-6:
            issues.append("soft:at_upper_no_sell_levels")
        else:
            issues.append("soft:held_but_no_sell_grid")

    row = {
        "bot_id": str(bot["_id"]),
        "botname": bot.get("botname"),
        "code": bot.get("code"),
        "bot_switch": info.get("switch"),
        "status": info.get("status"),
        "qty": qty,
        "can_sell_qty": can_sell,
        "pl_val": pos.get("pl_val"),
        "pl_ratio": pos.get("pl_ratio"),
        "price": pos.get("price"),
        "opt": pos.get("opt"),
        "order_status": pos.get("order_status"),
        "buy_grid": buy,
        "sell_grid": sell,
        "current_grid": cur,
        "buy_position": bp,
        "buy_pos_sum": sum(f(x) for x in bp),
        "grid": grid,
        "issues": issues,
    }
    sw = info.get("switch")
    if sw is True or str(sw).lower() == "true":
        all_switch_on.append({
            "botname": row["botname"], "code": row["code"],
            "qty": qty, "can_sell": can_sell,
            "buy": buy, "cur": cur, "sell": sell, "issues": issues,
        })
    if qty > 0 or can_sell > 0:
        held.append(row)
    elif sum(f(x) for x in bp) > 0:
        planned_only.append({
            "botname": row["botname"], "code": row["code"],
            "buy_pos_sum": row["buy_pos_sum"],
            "buy": buy, "cur": cur, "sell": sell, "issues": issues,
            "note": "buy_position only (not position.qty)",
        })

print(json.dumps({
    "held_count": len(held),
    "held_problem_count": sum(1 for r in held if any(not i.startswith("soft:") for i in r["issues"])),
    "held_soft_count": sum(1 for r in held if any(i.startswith("soft:") for i in r["issues"])),
    "held": held,
    "planned_buy_levels_count": len(planned_only),
    "planned_only": planned_only,
    "switch_on_count": len(all_switch_on),
    "switch_on": all_switch_on,
}, ensure_ascii=False, indent=2, default=str))
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
