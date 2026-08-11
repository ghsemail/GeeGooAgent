#!/usr/bin/env python3
"""Audit all GRID bots with non-empty position for order/consistency issues."""
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


def fnum(v, default=0.0):
    try:
        return float(v)
    except Exception:
        return default


def position_qty(pos):
    if not isinstance(pos, dict):
        return 0.0
    for k in ("quantity", "qty", "amount", "size", "stock_num", "volume"):
        if k in pos and pos[k] is not None:
            q = fnum(pos[k])
            if q:
                return q
    # nested common shapes
    for nest in ("long", "hold", "stock"):
        if isinstance(pos.get(nest), dict):
            q = position_qty(pos[nest])
            if q:
                return q
    # sum numeric leaves that look like qty
    total = 0.0
    for k, v in pos.items():
        if isinstance(v, (int, float)) and k.lower() in (
            "quantity", "qty", "amount", "size", "stock_num", "volume", "share", "shares"
        ):
            total += float(v)
    return total


def buy_pos_sum(bp):
    if not bp:
        return 0.0
    s = 0.0
    for x in bp:
        s += fnum(x)
    return s


def order_corrupt(buy, sell):
    buy = buy or []
    sell = sell or []
    for i in range(1, len(buy)):
        if fnum(buy[i]) < fnum(buy[i - 1]):
            return "buy_not_ascending"
    for i in range(1, len(sell)):
        if fnum(sell[i]) > fnum(sell[i - 1]):
            return "sell_not_descending"
    return None


def all_zero(xs):
    xs = xs or []
    return len(xs) > 0 and all(fnum(x) == 0 for x in xs)


def issues_for(bot, info):
    issues = []
    buy = list(info.get("buy_grid") or [])
    sell = list(info.get("sell_grid") or [])
    cur = fnum(info.get("current_grid"))
    bp = list(info.get("buy_position") or [])
    sp = list(info.get("sell_position") or [])
    pos = info.get("position") or {}
    qty = position_qty(pos if isinstance(pos, dict) else {})

    oc = order_corrupt(buy, sell)
    if oc:
        issues.append(oc)
    if (not buy and not sell) or (all_zero(buy) and not sell):
        issues.append("wiped_or_zero_grids")
    if cur <= 0:
        issues.append("current_grid_le_0")
    if len(bp) != len(buy):
        issues.append(f"buy_position_len_mismatch:{len(bp)}!={len(buy)}")
    if sp and len(sp) != len(sell):
        issues.append(f"sell_position_len_mismatch:{len(sp)}!={len(sell)}")

    grid = bot.get("grid") or {}
    upper = fnum(grid.get("upper_limit_price"))
    lower = fnum(grid.get("lower_limit_price"))
    if upper > 0 and lower > 0:
        for g in buy + sell + ([cur] if cur > 0 else []):
            if g < lower - 1e-6 or g > upper + 1e-6:
                issues.append(f"level_out_of_range:{g}")
                break
        # nearest buy should be <= current <= nearest sell (when present)
        if buy and cur > 0 and fnum(buy[-1]) > cur + 1e-6:
            issues.append("nearest_buy_above_current")
        if sell and cur > 0 and fnum(sell[-1]) < cur - 1e-6:
            issues.append("nearest_sell_below_current")

    # with real holdings, buy_position sum or position qty should be >0; flag if position says held but grids empty
    if qty > 0 and not buy and not sell and cur <= 0:
        issues.append("has_position_but_no_grids")

    # buy_position all zero while position qty > 0 can be ok (already moved to current), soft warn
    if qty > 0 and bp and buy_pos_sum(bp) == 0 and not sell:
        issues.append("soft:position_qty_but_buy_pos_zero_no_sell")

    return {
        "bot_id": str(bot.get("_id")),
        "botname": bot.get("botname"),
        "code": bot.get("code"),
        "switch": info.get("switch"),
        "status": info.get("status"),
        "qty": qty,
        "position": pos,
        "buy_grid": buy,
        "sell_grid": sell,
        "current_grid": cur,
        "buy_position": bp,
        "sell_position": sp,
        "buy_pos_sum": buy_pos_sum(bp),
        "grid": grid,
        "issues": issues,
    }


rows = []
no_info = []
for bot in db.grid_bot.find({}):
    info = db.grid_info.find_one({"bot_id": bot["_id"]})
    if not info:
        no_info.append({"bot_id": str(bot["_id"]), "botname": bot.get("botname"), "code": bot.get("code")})
        continue
    pos = info.get("position") or {}
    bp = list(info.get("buy_position") or [])
    qty = position_qty(pos if isinstance(pos, dict) else {})
    bpsum = buy_pos_sum(bp)
    # "有持仓": position qty > 0 OR buy_position sum > 0 OR sell_position sum > 0
    sps = buy_pos_sum(list(info.get("sell_position") or []))
    if qty <= 0 and bpsum <= 0 and sps <= 0:
        continue
    rows.append(issues_for(bot, info))

problem = [r for r in rows if r["issues"]]
ok = [r for r in rows if not r["issues"]]

# compact ok summary
ok_brief = [
    {
        "botname": r["botname"],
        "code": r["code"],
        "qty": r["qty"],
        "buy_pos_sum": r["buy_pos_sum"],
        "buy": r["buy_grid"],
        "cur": r["current_grid"],
        "sell": r["sell_grid"],
    }
    for r in ok
]

print(json.dumps({
    "with_position_count": len(rows),
    "problem_count": len(problem),
    "ok_count": len(ok),
    "no_info_count": len(no_info),
    "problems": problem,
    "ok": ok_brief,
    "no_info": no_info[:20],
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
