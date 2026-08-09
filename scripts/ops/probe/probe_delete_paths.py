#!/usr/bin/env python3
import json
from pathlib import Path
import paramiko

DEPLOY = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")
cfg = json.loads(DEPLOY.read_text(encoding="utf-8"))
KEY = "850367bc6d5fe8a4a53f267f5c308ac6d2ec1474d1764fe9"


def run(cmd: str) -> str:
    ssh = cfg["targets"]["geegoo-signal"]["ssh"]
    c = paramiko.SSHClient()
    c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    c.connect(
        hostname=ssh["host"],
        port=int(ssh.get("port", 22)),
        username=ssh["user"],
        password=ssh.get("password"),
        timeout=60,
    )
    _, o, e = c.exec_command(cmd, timeout=120)
    out = (o.read() + e.read()).decode()
    c.close()
    return out


def curl(url: str, body: str, auth: bool) -> str:
    h = '-H "Content-Type: application/json"'
    if auth:
        h += f' -H "Authorization: Bearer {KEY}"'
    return run(
        f'curl -s -w "\\nHTTP:%{{http_code}}" -X POST {url} {h} -d \'{body}\''
    )


def main() -> int:
    rid = json.loads(
        run(
            'curl -s -X POST http://127.0.0.1:3300/getNewsRefreshLogs '
            '-H "Content-Type: application/json" -d \'{"limit":1}\''
        )
    )[0]["run_id"]

    tests = [
        ("catalog direct", f"http://127.0.0.1:3210/deleteNewsRefreshLogs", True),
        ("nginx op_catalog", f"http://127.0.0.1:8088/op_catalog/deleteNewsRefreshLogs", True),
        ("nginx no auth", f"http://127.0.0.1:8088/op_catalog/deleteNewsRefreshLogs", False),
        ("3210 no auth", f"http://127.0.0.1:3210/deleteNewsRefreshLogs", False),
    ]
    for name, url, auth in tests:
        print(f"\n=== {name} ===")
        print(curl(url, json.dumps({"run_id": rid}), auth))
        # re-fetch run_id if deleted
        rid = json.loads(
            run(
                'curl -s -X POST http://127.0.0.1:3300/getNewsRefreshLogs '
                '-H "Content-Type: application/json" -d \'{"limit":1}\''
            )
        )[0]["run_id"]

    print("\n=== nginx op_catalog config ===")
    print(
        run(
            "docker exec 0cb244428c30 sh -c 'grep -R op_catalog /etc/nginx 2>/dev/null | head -20'"
        )
    )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
