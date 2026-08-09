#!/usr/bin/env python3
"""Test trading_operation login endpoints from local machine."""
import json
import urllib.error
import urllib.request

API_KEY = "850367bc6d5fe8a4a53f267f5c308ac6d2ec1474d1764fe9"
BODY = json.dumps({"username": "test", "password": "test"}).encode()
HEADERS = {
    "Content-Type": "application/json",
    "Authorization": f"Bearer {API_KEY}",
}

URLS = [
    "http://146.56.225.252:3210/login",
    "http://146.56.225.252:8088/op_catalog/login",
    "http://146.56.225.252:8066/op_catalog/login",
]

for url in URLS:
    req = urllib.request.Request(url, data=BODY, headers=HEADERS, method="POST")
    try:
        with urllib.request.urlopen(req, timeout=10) as resp:
            print(url, "->", resp.status, resp.read()[:200])
    except urllib.error.HTTPError as e:
        print(url, "-> HTTP", e.code, e.read()[:200])
    except Exception as e:
        print(url, "-> ERROR", type(e).__name__, e)
