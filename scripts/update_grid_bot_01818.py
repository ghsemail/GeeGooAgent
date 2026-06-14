# coding=utf-8
"""Update GRID bot 01818.HK names via geegoo mcp."""
import json
from pathlib import Path

import requests

cfg = json.loads(
    Path(r"C:/Users/ghsemail/.cursor/skills/geegoo/config.json").read_text(encoding="utf-8")
)
base = cfg["base_url"].rstrip("/")
headers = {
    "Authorization": f"Bearer {cfg['api_key']}",
    "Content-Type": "application/json",
}
bot_id = "6908395ac968cf04c9115041"
payload = {
    "mcp_token": cfg["mcp_token"],
    "bot_id": bot_id,
    "stock_name": "招金矿业",
    "botname": "招金矿业-GRID",
}

resp = requests.post(f"{base}/updateGRIDBot", json=payload, headers=headers, timeout=30)
print("update:", resp.status_code, json.dumps(resp.json(), ensure_ascii=False))

verify = requests.post(
    f"{base}/getAllGRIDBots",
    json={"mcp_token": cfg["mcp_token"]},
    headers=headers,
    timeout=30,
)
for bot in verify.json().get("data", []):
    if bot.get("bot_id") == bot_id:
        print(
            "verified:",
            bot.get("code"),
            bot.get("stock_name"),
            bot.get("botname"),
        )
        break
