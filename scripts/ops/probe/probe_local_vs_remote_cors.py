#!/usr/bin/env python3
"""Compare remote same-origin vs localhost dev CORS on :8088 nginx."""
from __future__ import annotations

import json
import urllib.error
import urllib.request
from pathlib import Path

BASE = "http://146.56.225.252:8088"
API_KEY = "sk-7a2b3c4d5e6f7g8h9i0j1k2l3m4n5o6p7q8r9s0t1u2v3w4x5y6z7a8b9c0d1e2"
PATHS = ["/op_agent/v1/doctor", "/op_catalog/login", "/op_bot/queryTradingDate", "/op_news/getNewsRefreshLogs"]


def cors_preflight(path: str, origin: str) -> dict:
    req = urllib.request.Request(
        f"{BASE}{path}",
        method="OPTIONS",
        headers={
            "Origin": origin,
            "Access-Control-Request-Method": "POST",
            "Access-Control-Request-Headers": "authorization,content-type,x-client-source",
        },
    )
    try:
        with urllib.request.urlopen(req, timeout=15) as resp:
            hdrs = {k: v for k, v in resp.headers.items() if "access-control" in k.lower()}
            return {"status": resp.status, "headers": hdrs, "error": None}
    except urllib.error.HTTPError as e:
        hdrs = {k: v for k, v in e.headers.items() if "access-control" in k.lower()}
        return {"status": e.code, "headers": hdrs, "error": e.read()[:120].decode("utf-8", "replace")}
    except Exception as e:
        return {"status": None, "headers": {}, "error": str(e)}


def get_probe(path: str, origin: str) -> dict:
    req = urllib.request.Request(
        f"{BASE}{path}",
        headers={
            "Origin": origin,
            "Authorization": f"Bearer {API_KEY}",
            "X-Client-Source": "trading_operation",
        },
    )
    try:
        with urllib.request.urlopen(req, timeout=15) as resp:
            body = resp.read(200)
            hdrs = {k: v for k, v in resp.headers.items() if "access-control" in k.lower()}
            return {"status": resp.status, "headers": hdrs, "body": body[:120]}
    except urllib.error.HTTPError as e:
        hdrs = {k: v for k, v in e.headers.items() if "access-control" in k.lower()}
        return {"status": e.code, "headers": hdrs, "body": e.read()[:120]}
    except Exception as e:
        return {"status": None, "headers": {}, "body": str(e).encode()}


def main() -> int:
    origins = [
        ("remote-same-origin", "http://146.56.225.252:8088"),
        ("localhost-flutter", "http://localhost:52341"),
        ("localhost-8080", "http://localhost:8080"),
        ("127-flutter", "http://127.0.0.1:52341"),
    ]
    print("=== CORS preflight (OPTIONS) ===")
    for label, origin in origins:
        print(f"\n-- {label} origin={origin}")
        for path in PATHS:
            r = cors_preflight(path, origin)
            acao = r["headers"].get("Access-Control-Allow-Origin", "(missing)")
            print(f"  {path}: status={r['status']} ACAO={acao}")

    print("\n=== GET doctor (agent) ===")
    for label, origin in origins:
        r = get_probe("/op_agent/v1/doctor", origin)
        print(f"{label}: status={r['status']} ACAO={r['headers'].get('Access-Control-Allow-Origin', '-')}")

    # nginx config snippet
    deploy = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")
    if deploy.exists():
        import paramiko

        cfg = json.loads(deploy.read_text(encoding="utf-8"))
        ssh = cfg["targets"]["trading-operation"]["ssh"]
        c = paramiko.SSHClient()
        c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
        c.connect(ssh["host"], 22, ssh["user"], ssh.get("password"), timeout=20)
        _, o, _ = c.exec_command(
            "docker ps -q --filter publish=8088 | head -1 | xargs -I{} docker exec {} "
            "sh -c 'nginx -T 2>/dev/null | grep -E \"add_header Access-Control|map \\$http_origin\" | head -30'",
            timeout=30,
        )
        print("\n=== nginx CORS config (snippet) ===")
        print(o.read().decode("utf-8", "replace") or "(empty)")
        c.close()
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
