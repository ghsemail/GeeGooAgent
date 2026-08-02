#!/usr/bin/env python3
import json
from pathlib import Path
import paramiko

DEPLOY = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")
cfg = json.loads(DEPLOY.read_text(encoding="utf-8"))
KEY = "850367bc6d5fe8a4a53f267f5c308ac6d2ec1474d1764fe9"


def run(target: str, cmd: str) -> str:
    ssh = cfg["targets"][target]["ssh"]
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


def main() -> int:
    print("=== delete with bearer (catalog) ===")
    print(
        run(
            "geegoo-signal",
            f'curl -s -w "\\nHTTP:%{{http_code}}" -X POST http://127.0.0.1:3210/deleteNewsRefreshLogs '
            f'-H "Content-Type: application/json" -H "Authorization: Bearer {KEY}" '
            '-d \'{"run_id":"__nonexistent__"}\'',
        )
    )

    print("\n=== delete via nginx op_catalog ===")
    print(
        run(
            "geegoo-signal",
            f'curl -s -w "\\nHTTP:%{{http_code}}" -X POST http://127.0.0.1:8088/op_catalog/deleteNewsRefreshLogs '
            f'-H "Content-Type: application/json" -H "Authorization: Bearer {KEY}" '
            '-d \'{"run_id":"__nonexistent__"}\'',
        )
    )

    print("\n=== CN news sample ===")
    print(
        run(
            "geegoo-data-cn",
            'curl -s -X POST http://127.0.0.1:3300/getStockNews -H "Content-Type: application/json" '
            '-d \'{"stock_list":[{"code":"000858.SZ","name":{"init":"五粮液"}}],"language":"cn"}\' '
            "| python3 -c \"import sys,json; d=json.load(sys.stdin); n=d[0] if d else {}; t=n.get('title',{}); s=n.get('summary',{}); print('n',len(d)); print('title.cn',repr(t.get('cn','')[:60])); print('title.init',repr(t.get('init','')[:60])); print('title.en',repr(t.get('en',''))); print('summary.cn',repr(str(s.get('cn',''))[:80]))\"",
        )
    )

    print("\n=== bot proxy CN news ===")
    bot_key = run("geegoo-bot", "grep ^GEEGOO_BOT_APP_API_KEY= /home/ubuntu/apps/GeeGooBot/.env | cut -d= -f2-").strip()
    print(
        run(
            "geegoo-bot",
            f'curl -s -X POST http://127.0.0.1:3100/getStockNews -H "Content-Type: application/json" '
            f'-H "Authorization: Bearer {bot_key}" '
            '-d \'{"stock_list":[{"code":"000858.SZ","name":{"init":"五粮液"}}],"language":"cn"}\' '
            "| python3 -c \"import sys,json; d=json.load(sys.stdin); n=d[0] if d else {}; t=n.get('title',{}); print('title.cn',repr(t.get('cn','')[:80])); print('title.init',repr(t.get('init','')[:80]))\"",
        )
    )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
