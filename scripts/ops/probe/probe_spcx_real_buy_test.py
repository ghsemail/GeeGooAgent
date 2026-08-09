#!/usr/bin/env python3
"""One-off REAL buy test: 1 share SPCX.US via TradingServer (password unlock)."""
from __future__ import annotations

import json
from pathlib import Path

import paramiko

REMOTE = r'''
import json
import urllib.error
import urllib.request
from datetime import datetime

from bson import ObjectId
from pymongo import MongoClient

USER_ID = "6366170502d5c175fd586fe8"
CODE = "SPCX.US"


def env(k):
    for line in open("/home/ubuntu/apps/GeeGooBot/.env"):
        if line.startswith(k + "="):
            return line.strip().split("=", 1)[1]
    return ""


def post(url, body, headers):
    req = urllib.request.Request(
        url,
        data=json.dumps(body).encode(),
        headers=headers,
        method="POST",
    )
    with urllib.request.urlopen(req, timeout=60) as r:
        return json.loads(r.read().decode())


db = MongoClient(env("GEEGOO_BOT_MONGO_URI"))[env("GEEGOO_BOT_MONGO_DB") or "QT_DB"]
user = db.user.find_one({"_id": ObjectId(USER_ID)})
trade = user.get("trade") or {}
token = str(trade.get("trade_api_token") or "").strip()
host = str(trade.get("bot_host") or "43.134.94.87").strip()
port = int(trade.get("bot_port") or 7000)
trade_env = str(trade.get("trade_env") or "REAL").upper()
mcp_tok = str((user.get("mcp") or {}).get("mcp_token") or "").strip()
api_key = env("GEEGOO_BOT_APP_API_KEY")

print("trade_target", host, port, "env", trade_env, "token_set", bool(token))

price_resp = post(
    "http://127.0.0.1:3120/getCurrentPrice",
    {"mcp_token": mcp_tok, "code": CODE},
    {"Content-Type": "application/json", "Authorization": "Bearer " + api_key},
)
price = float(price_resp.get("price") or 0)
print("quote", json.dumps(price_resp, ensure_ascii=False))
if price <= 0:
    raise SystemExit("no price for " + CODE)

trade_body = {
    "user_id": USER_ID,
    "opt": "buy",
    "code": CODE,
    "qty": 1,
    "price": price,
    "trade_env": trade_env,
    "futu_host": "127.0.0.1",
    "futu_port": 11111,
    "request_time": datetime.now().strftime("%Y-%m-%d %H:%M:%S"),
}
print("trade_req", json.dumps({**trade_body, "user_id": USER_ID}, ensure_ascii=False))

url = f"http://{host}:{port}/v1/futu/trade"
try:
    trade_resp = post(
        url,
        trade_body,
        {"Content-Type": "application/json", "x-trading-token": token},
    )
    print("trade_resp", json.dumps(trade_resp, ensure_ascii=False))
except urllib.error.HTTPError as e:
    print("trade_http_err", e.code, e.read().decode("utf-8", "replace")[:2000])
'''


def main() -> None:
    cfg = json.loads(
        Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json").read_text(
            encoding="utf-8"
        )
    )
    s = cfg["targets"]["geegoo-bot"]["ssh"]
    c = paramiko.SSHClient()
    c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    c.connect(s["host"], username=s["user"], password=s.get("password"), timeout=30)
    remote_path = "/tmp/probe_spcx_real_buy_test.py"
    with c.open_sftp().file(remote_path, "w") as f:
        f.write(REMOTE)
    _, o, e = c.exec_command(f"python3 {remote_path}", timeout=120)
    print((o.read() + e.read()).decode("utf-8", "replace"))
    c.close()


if __name__ == "__main__":
    main()
