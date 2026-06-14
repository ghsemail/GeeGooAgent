# coding=utf-8
import json
from pathlib import Path

import requests

cfg = json.loads(
    Path(r"C:/Users/ghsemail/.cursor/skills/geegoo/config.json").read_text(encoding="utf-8")
)
resp = requests.post(
    f"{cfg['base_url'].rstrip('/')}/getAllGRIDBots",
    json={"mcp_token": cfg["mcp_token"]},
    headers={
        "Authorization": f"Bearer {cfg['api_key']}",
        "Content-Type": "application/json",
    },
    timeout=30,
)
data = resp.json()
print(json.dumps(data, ensure_ascii=False, indent=2))
