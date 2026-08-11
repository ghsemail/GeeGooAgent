#!/usr/bin/env python3
"""Probe 招金矿业 GRID bot grids — why all zeros."""
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

q = {
    "$or": [
        {"stock_name": {"$regex": "招金", "$options": "i"}},
        {"botname": {"$regex": "招金", "$options": "i"}},
        {"code": {"$regex": "1818|01818", "$options": "i"}},
    ]
}
bots = list(db["grid_bot"].find(q))
print("grid_bot matches", len(bots))
for b in bots:
    bid = b["_id"]
    print("\n=== BOT", bid, b.get("botname"), b.get("code"), b.get("stock_name"), "===")
    grid = b.get("grid") or {}
    print("param.grid keys", list(grid.keys()) if isinstance(grid, dict) else type(grid))
    for side in ("buy_grid", "sell_grid", "buy", "sell"):
        if isinstance(grid, dict) and side in grid:
            v = grid[side]
            print(" param", side, "type", type(v).__name__, "len", len(v) if hasattr(v, "__len__") else None, "sample", list(v)[:8] if hasattr(v, "__iter__") and not isinstance(v, (str, dict)) else v)

    info = db["grid_info"].find_one({"bot_id": bid}) or db["grid_info"].find_one({"bot_id": str(bid)})
    if info:
        print("info.status", info.get("status"), "switch", info.get("switch"))
        pos = info.get("position") or {}
        print("info.position keys sample", {k: pos.get(k) for k in list(pos)[:8]})
        # some schemas store grids on info
        for side in ("buy_grid", "sell_grid"):
            if side in info:
                v = info[side]
                print(" info", side, type(v).__name__, list(v)[:10] if hasattr(v, "__iter__") and not isinstance(v, (str, dict)) else v)

    # recent logs
    logs = list(db["grid_log"].find({"$or": [{"bot_id": bid}, {"bot_id": str(bid)}]}).sort("time", -1).limit(5))
    print("recent logs", len(logs))
    for row in logs:
        lg = row.get("log") or {}
        bg, sg = lg.get("buy_grid"), lg.get("sell_grid")
        def summarize(v):
            if v is None:
                return None
            if isinstance(v, list):
                return {"type": "list", "len": len(v), "head": v[:6], "all_zero": all((x == 0 or x == 0.0) for x in v)}
            return {"type": type(v).__name__, "val": str(v)[:120]}
        print(" ", row.get("time"), "next", lg.get("next_opt"),
              "buy", summarize(bg), "sell", summarize(sg),
              "cur", lg.get("current_grid"))

# also search reminder
rems = list(db["grid_reminder"].find(q))
print("\ngrid_reminder matches", len(rems))
for r in rems:
    print(r.get("_id"), r.get("botname") or r.get("reminder_name"), r.get("code"), r.get("stock_name"))
    g = r.get("grid") or {}
    for side in ("buy_grid", "sell_grid"):
        if side in g:
            v = g[side]
            print(" ", side, type(v).__name__, list(v)[:8] if hasattr(v,"__iter__") and not isinstance(v,(str,dict)) else v)
'''


def main() -> int:
    cfg = json.loads(DEPLOY.read_text(encoding="utf-8"))
    s = cfg["targets"]["geegoo-bot"]["ssh"]
    c = paramiko.SSHClient()
    c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    c.connect(s["host"], username=s["user"], password=s.get("password"), timeout=30)
    p = "/tmp/probe_zhaojin_grid.py"
    with c.open_sftp().file(p, "w") as f:
        f.write(REMOTE)
    _, o, e = c.exec_command(f"python3 {p}", timeout=120)
    print((o.read() + e.read()).decode("utf-8", errors="replace"))
    c.close()
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
