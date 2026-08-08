#!/usr/bin/env python3
import json, os, urllib.request

cfg = json.load(open(os.path.expanduser("~/.geegoo/config.json"), encoding="utf-8"))
if not str(cfg.get("mcp_token", "")).strip():
    cfg["mcp_token"] = "mcp_rbWb0H-oiGPDAuZhM-5o2v2l50z1uG10"
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
        return r.status, json.loads(r.read().decode())

for d in ["2026-08-07", "2026-08-08"]:
    try:
        c, r = post("/getMarketPremarketReport", {"mcp_token": token, "market": "CN", "report_date": d})
        data = r.get("data") or {}
        print(d, "http", c, "code", r.get("code"), "msg", r.get("message"), "found", bool(data.get("report")), "len", len(data.get("report") or ""))
    except urllib.error.HTTPError as e:
        print(d, "http", e.code, e.read()[:200])
