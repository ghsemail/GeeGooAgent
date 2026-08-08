#!/usr/bin/env python3
import json, os, urllib.request

cfg = json.load(open(os.path.expanduser("~/.geegoo/config.json"), encoding="utf-8"))
token = cfg.get("mcp_token", "")
api_key = cfg.get("api_key", "")

def post(path, body):
    data = json.dumps(body).encode()
    req = urllib.request.Request(
        "http://118.195.135.97:3120" + path,
        data=data,
        headers={"Content-Type": "application/json", "Authorization": "Bearer " + api_key},
        method="POST",
    )
    with urllib.request.urlopen(req, timeout=30) as r:
        return json.loads(r.read().decode())

users = post("/listReportUsers", {"mcp_token": token, "market": "CN"})
items = (users.get("data") or users.get("users") or users)
print("list keys", users.keys() if isinstance(users, dict) else type(users))
if isinstance(users, dict) and "data" in users:
    items = users["data"]
print("count", len(items) if isinstance(items, list) else items)
for u in (items or [])[:5]:
    if not isinstance(u, dict):
        continue
    ut = u.get("mcp_token") or u.get("mcpToken") or ""
    print("user", u.get("user_id") or u.get("userId"), "token", ut[:16])
    try:
        g = post("/getMarketPremarketReport", {"mcp_token": ut or token, "market": "CN", "report_date": "2026-08-07"})
        print("  get ok", bool((g.get("data") or {}).get("report")))
    except Exception as e:
        print("  get err", e)
