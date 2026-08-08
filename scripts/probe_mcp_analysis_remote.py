#!/usr/bin/env python3
"""Probe getMCPAnalysis from agent server with real config."""
import json, os, urllib.request, urllib.error

cfg = json.load(open(os.path.expanduser("~/.geegoo/config.json"), encoding="utf-8"))
api_key = cfg.get("api_key", "")
mcp_token = cfg.get("mcp_token", "")
# analyze URL from typical config
analyze_url = cfg.get("signal_analyze_url") or cfg.get("analyze_url") or "http://146.56.225.252:3211"
if isinstance(cfg.get("mcp"), dict):
    analyze_url = cfg["mcp"].get("analyze_url", analyze_url)

body = {
    "mcp_token": mcp_token,
    "name": "贵州茅台",
    "code": "600519.SH",
    "prompt_id": "663e5ac904f98788e502fab7",
    "period": "weekly",
    "language": "cn",
}
data = json.dumps(body).encode()
for base in [analyze_url.rstrip("/"), "http://146.56.225.252:3211", "http://146.56.225.252:3210"]:
    url = base + "/getMCPAnalysis"
    req = urllib.request.Request(
        url, data=data,
        headers={"Content-Type": "application/json", "Authorization": "Bearer " + api_key},
        method="POST",
    )
    try:
        with urllib.request.urlopen(req, timeout=120) as r:
            resp = json.loads(r.read().decode())
            print(url, "OK", r.status, "code", resp.get("code"), "len", len(str(resp.get("data", ""))))
    except urllib.error.HTTPError as e:
        print(url, "HTTP", e.code, e.read()[:300])
    except Exception as e:
        print(url, "ERR", e)
