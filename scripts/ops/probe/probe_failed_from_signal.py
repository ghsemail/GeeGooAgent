#!/usr/bin/env python3
"""Re-probe failed endpoints from signal host (internal network)."""
from __future__ import annotations

import json
from pathlib import Path

import paramiko

DEPLOY = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")

TESTS = [
    ("GET", "http://47.80.14.120:3300/health", None, None),
    ("GET", "http://82.157.97.76:3300/health", None, None),
    ("POST", "http://47.80.14.120:3300/getStockNews", None, '{"stock_list":[{"code":"TSLA.US","name":{"init":"T"}}]}'),
    ("POST", "http://82.157.97.76:3300/getStockNews", None, '{"stock_list":[{"code":"000858.SZ","name":{"init":"w"}}]}'),
    ("GET", "http://43.134.94.87:7000/health", None, None),
    ("GET", "http://119.45.16.112:3400/health", None, None),
    ("POST", "http://146.56.225.252:3210/getCustomSignal", "850367bc6d5fe8a4a53f267f5c308ac6d2ec1474d1764fe9", "{}"),
    ("POST", "http://118.195.135.97:3100/getUserList", "sk-7a2b3c4d5e6f7g8h9i0j1k2l3m4n5o6p7q8r9s0t1u2v3w4x5y6z7a8b9c0d1e2", "{}"),
    ("POST", "http://146.56.225.252:3210/getLLMResult", "850367bc6d5fe8a4a53f267f5c308ac6d2ec1474d1764fe9", '{"prompt":"ping","dict":{}}'),
    ("POST", "http://146.56.225.252:3210/queryVersion", "850367bc6d5fe8a4a53f267f5c308ac6d2ec1474d1764fe9", '{"name":"slot"}'),
]

cmd = "python3 - <<'PY'\nimport json,urllib.request\n"
for method, url, token, body in TESTS:
    cmd += f"print('--- {url}')\n"
    cmd += "try:\n"
    cmd += f"  req=urllib.request.Request({url!r}, method={method!r}"
    if body:
        cmd += f", data={body!r}.encode(), headers={{'Content-Type':'application/json'"
        if token:
            cmd += f", 'Authorization':'Bearer {token}'"
        cmd += "}"
    elif token:
        cmd += f", headers={{'Authorization':'Bearer {token}'}}"
    cmd += ")\n"
    cmd += "  with urllib.request.urlopen(req,timeout=30) as r: print(r.status, r.read()[:120])\n"
    cmd += "except Exception as e: print('ERR', e)\n"
cmd += "PY"

cfg = json.loads(DEPLOY.read_text(encoding="utf-8"))
ssh = cfg["targets"]["geegoo-signal"]["ssh"]
c = paramiko.SSHClient()
c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
c.connect(ssh["host"], username=ssh["user"], password=ssh["password"], timeout=60)
_, o, e = c.exec_command(cmd, timeout=180)
print((o.read() + e.read()).decode())
c.close()
