#!/usr/bin/env python3
"""Simulate browser CORS: Origin 8088 -> catalog-api :3210."""
import json
import urllib.request

API_KEY = "850367bc6d5fe8a4a53f267f5c308ac6d2ec1474d1764fe9"
ORIGIN = "http://146.56.225.252:8088"

# Preflight
req = urllib.request.Request(
    "http://146.56.225.252:3210/login",
    method="OPTIONS",
    headers={
        "Origin": ORIGIN,
        "Access-Control-Request-Method": "POST",
        "Access-Control-Request-Headers": "authorization,content-type",
    },
)
with urllib.request.urlopen(req, timeout=10) as resp:
    print("OPTIONS", resp.status)
    for k, v in resp.headers.items():
        if "access-control" in k.lower():
            print(f"  {k}: {v}")

# POST with Origin (browser cross-port)
body = json.dumps({"username": "test", "password": "test"}).encode()
req2 = urllib.request.Request(
    "http://146.56.225.252:3210/login",
    data=body,
    method="POST",
    headers={
        "Origin": ORIGIN,
        "Content-Type": "application/json",
        "Authorization": f"Bearer {API_KEY}",
    },
)
with urllib.request.urlopen(req2, timeout=10) as resp:
    print("POST", resp.status, resp.read()[:120])

# localhost dev scenario
ORIGIN2 = "http://localhost:54321"
req3 = urllib.request.Request(
    "http://146.56.225.252:3210/login",
    method="OPTIONS",
    headers={
        "Origin": ORIGIN2,
        "Access-Control-Request-Method": "POST",
    },
)
with urllib.request.urlopen(req3, timeout=10) as resp:
    print("\nOPTIONS localhost origin", resp.status)
    for k, v in resp.headers.items():
        if "access-control" in k.lower():
            print(f"  {k}: {v}")
