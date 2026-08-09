#!/usr/bin/env python3
"""Probe bot log/profit APIs for ghsemail user bots."""
from __future__ import annotations

import json
from pathlib import Path

import paramiko

KEY = "sk-7a2b3c4d5e6f7g8h9i0j1k2l3m4n5o6p7q8r9s0t1u2v3w4x5y6z7a8b9c0d1e2"
BOT_HOST = "118.195.135.97"
BASE = f"http://{BOT_HOST}:3100"


def curl(client, path: str, body: dict) -> tuple[int, str]:
    payload = json.dumps(body, ensure_ascii=False).replace("'", "'\\''")
    cmd = (
        f"curl -s -m 30 -w '\\nHTTP=%{{http_code}}' -X POST {BASE}/{path} "
        f"-H 'Content-Type: application/json' -H 'Authorization: Bearer {KEY}' "
        f"-d '{payload}'"
    )
    _, o, _ = client.exec_command(cmd, timeout=45)
    text = o.read().decode("utf-8", errors="replace")
    if "HTTP=" in text:
        body_text, _, code = text.rpartition("HTTP=")
        return int(code.strip()), body_text.strip()
    return 0, text


def main() -> None:
    cfg = json.loads(Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json").read_text(encoding="utf-8"))
    ssh = cfg["targets"]["geegoo-bot"]["ssh"]
    c = paramiko.SSHClient()
    c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    c.connect(ssh["host"], 22, ssh["user"], ssh["password"], timeout=20)

    code, bots_raw = curl(c, "getUserBot", {"user_id": "6366170502d5c175fd586fe8"})
    print("getUserBot HTTP", code)
    print(bots_raw[:1200])
    try:
        bots = json.loads(bots_raw)
    except json.JSONDecodeError:
        c.close()
        return

    bot_list = []
    if isinstance(bots, list):
        bot_list = bots
    elif isinstance(bots, dict):
        if bots.get("code") not in (None, 100) and "DCA" not in bots:
            print("getUserBot error:", bots)
            c.close()
            return
        for key in ("DCA", "GRID", "HDG", "SMARTTRADE", "TRADE"):
            for item in bots.get(key) or []:
                item = dict(item)
                item.setdefault("type", key)
                bot_list.append(item)

    for b in (bot_list or [])[:8]:
        bot_id = b.get("bot_id") or b.get("_id")
        btype = b.get("type") or b.get("bot_type")
        print(f"\n=== bot {bot_id} type={btype} name={b.get('name','')} ===")
        endpoints = []
        if str(btype).upper() in ("DCA", "1") or btype == 1:
            endpoints = [("getDCABotLog", {"bot_id": bot_id, "hold": "all"}), ("getDCABotProfit", {"bot_id": bot_id})]
        elif str(btype).upper() in ("GRID", "2") or btype == 2:
            endpoints = [("getGRIDBotLog", {"bot_id": bot_id, "hold": True}), ("getGRIDBotProfit", {"bot_id": bot_id})]
        elif str(btype).upper() in ("HDG", "3") or btype == 3:
            endpoints = [("getHDGBotLog", {"bot_id": bot_id}), ("getHDGBotProfit", {"bot_id": bot_id})]
        elif str(btype).upper() in ("SMARTTRADE", "TRADE", "4") or btype == 4:
            endpoints = [("getSmartTradeLog", {"bot_id": bot_id, "hold": True}), ("getSmartTradeProfit", {"bot_id": bot_id})]
        else:
            continue
        for ep, body in endpoints:
                hc, raw = curl(c, ep, body)
                print(f"--- {ep} HTTP {hc} ---")
                try:
                    data = json.loads(raw)
                    if ep.endswith("Log"):
                        info = data.get("info") or {}
                        print("info pl_val/pl_ratio:", info.get("pl_val"), info.get("pl_ratio"), "qty:", info.get("qty"))
                        logs = data.get("log") or []
                        if logs:
                            pos = (logs[0] or {}).get("position") or {}
                            print("log[0] position pl_val:", pos.get("pl_val"), "time:", (logs[0] or {}).get("time"))
                    else:
                        tp = data.get("total_profit") or {}
                        pl = data.get("profit_list") or []
                        print("total_profit:", tp)
                        if pl:
                            print("profit_list[0]:", pl[0])
                        else:
                            print("profit_list empty, keys:", list(data.keys()), "raw:", raw[:300])
                except Exception as ex:
                    print(raw[:800], ex)

    c.close()


if __name__ == "__main__":
    main()
