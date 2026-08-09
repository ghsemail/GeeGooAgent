#!/usr/bin/env python3
import json, os, urllib.request

TOKEN = "mcp_rbWb0H-oiGPDAuZhM-5o2v2l50z1uG10"
API_KEY = "sk-7a2b3c4d5e6f7g8h9i0j1k2l3m4n5o6p7q8r9s0t1u2v3w4x5y6z7a8b9c0d1e2"
cfg_path = os.path.expanduser("~/.geegoo/config.json")
cfg = json.load(open(cfg_path, encoding="utf-8"))
cfg["mcp_token"] = TOKEN
if not cfg.get("api_key"):
    cfg["api_key"] = API_KEY
json.dump(cfg, open(cfg_path, "w", encoding="utf-8"), indent=2, ensure_ascii=False)

def post(path, body):
    data = json.dumps(body).encode()
    req = urllib.request.Request(
        "http://118.195.135.97:3120" + path,
        data=data,
        headers={"Content-Type": "application/json", "Authorization": "Bearer " + cfg["api_key"]},
        method="POST",
    )
    with urllib.request.urlopen(req, timeout=30) as r:
        return r.status, json.loads(r.read().decode())

for d in ["2026-08-07"]:
    c, r = post("/getMarketPremarketReport", {"mcp_token": TOKEN, "market": "CN", "report_date": d})
    data = r.get("data") or {}
    print(d, "code", r.get("code"), "msg", r.get("message"), "found", bool(data.get("report")), "len", len(data.get("report") or ""))
