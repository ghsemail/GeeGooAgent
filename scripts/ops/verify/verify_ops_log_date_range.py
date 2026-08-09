#!/usr/bin/env python3
import json
from pathlib import Path
import paramiko
import urllib.request

cfg = json.loads(Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json").read_text(encoding="utf-8"))
body = json.dumps({"limit": 2, "run_date_from": "2026-07-01", "run_date_to": "2026-07-31"}).encode()

def curl_local(url, headers=None):
    req = urllib.request.Request(url, data=body, method="POST", headers=headers or {"Content-Type": "application/json"})
    with urllib.request.urlopen(req, timeout=15) as r:
        data = r.read()[:300]
        print(url, "->", r.status, data)

print("=== external catalog strategy ===")
curl_local(
    "http://146.56.225.252:3210/getStrategyGenerationLogs",
    {"Content-Type": "application/json", "Authorization": "Bearer 850367bc6d5fe8a4a53f267f5c308ac6d2ec1474d1764fe9"},
)
print("=== external catalog news ===")
curl_local(
    "http://146.56.225.252:3210/getNewsRefreshLogs",
    {"Content-Type": "application/json", "Authorization": "Bearer 850367bc6d5fe8a4a53f267f5c308ac6d2ec1474d1764fe9"},
)
print("=== external news server ===")
curl_local("http://47.80.14.120:5800/getNewsRefreshLogs")
