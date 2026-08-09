#!/usr/bin/env python3
"""Deep review: TradingData vs GeeGooData news migration."""
from __future__ import annotations

import json
from pathlib import Path

import paramiko

DEPLOY = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")


def run(target: str, cmd: str, timeout: int = 120) -> str:
    cfg = json.loads(DEPLOY.read_text(encoding="utf-8"))
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
    _, o, e = c.exec_command(cmd, timeout=timeout)
    out = (o.read() + e.read()).decode()
    c.close()
    return out


def main() -> int:
    print("## TradingData Python (47.80.14.120)")
    print(
        run(
            "geegoo-tradingdata",
            "ss -lntp | grep -E ':(5500|5600|5700|5800|5900)' || echo PORTS_DOWN; "
            "pgrep -af 'NewsServer|RefreshNews|AIServer|USData' || echo NO_PYTHON_NEWS",
        )
    )

    print("\n## GeeGooData HK (47.80.14.120)")
    print(run("geegoo-data", "cd /root/apps/GeeGooData && bash start.sh status"))
    print(
        run(
            "geegoo-data",
            "grep -E 'MONGO|BOT_SERVICE|NEWS_REFRESH|NEWS_SOURCES' /root/apps/GeeGooData/.env",
        )
    )
    print(
        run(
            "geegoo-data",
            'curl -s -X POST http://127.0.0.1:3300/getStockNews -H "Content-Type: application/json" '
            '-d \'{"stock_list":[{"code":"TSLA.US","name":{"init":"Tesla"}}]}\' '
            "| python3 -c \"import sys,json; d=json.load(sys.stdin); print('items',len(d), 'code', d[0].get('code') if d else None, 'publisher', d[0].get('publisher') if d else None)\"",
        )
    )
    print(
        run(
            "geegoo-data",
            'curl -s -X POST http://127.0.0.1:3300/getNewsRefreshLogs -H "Content-Type: application/json" '
            '-d \'{"limit":1}\' | python3 -c "import sys,json; d=json.load(sys.stdin); x=d[0]; print(x.get(\'run_date\'), x.get(\'total_news\'), x.get(\'success_stocks\'), x.get(\'status\'))"',
        )
    )
    print(run("geegoo-data", "tail -3 /root/apps/GeeGooData/news-worker.out 2>/dev/null || echo NO_WORKER_LOG"))

    print("\n## GeeGooData CN (82.157.97.76)")
    print(run("geegoo-data-cn", "cd /home/ubuntu/apps/GeeGooData && bash start.sh status 2>/dev/null || echo NO_START_SH"))
    print(
        run(
            "geegoo-data-cn",
            "grep -E 'MONGO|BOT_SERVICE|NEWS_REFRESH|NEWS_SOURCES' /home/ubuntu/apps/GeeGooData/.env 2>/dev/null || echo NO_ENV",
        )
    )
    print(
        run(
            "geegoo-data-cn",
            'curl -sf http://127.0.0.1:3300/health && echo; '
            'curl -s -X POST http://127.0.0.1:3300/getStockNews -H "Content-Type: application/json" '
            '-d \'{"stock_list":[{"code":"000858.SZ","name":{"init":"五粮液"}}]}\' | head -c 200; echo',
        )
    )

    print("\n## GeeGooBot getAllUserNewsStock")
    print(
        run(
            "geegoo-bot",
            'curl -s -X POST http://127.0.0.1:3140/getAllUserNewsStock '
            '-H "Content-Type: application/json" '
            '-H "Authorization: Bearer $(grep ^GEEGOO_BOT_SERVICE_API_KEY= /home/ubuntu/apps/GeeGooBot/.env|cut -d= -f2-)" '
            '-d "{}" | python3 -c "import sys,json; d=json.load(sys.stdin); print(\'stocks\', len(d), [x.get(\'code\') for x in d[:8]])"',
        )
    )

    print("\n## catalog news proxy (146.56.225.252)")
    print(
        run(
            "geegoo-signal",
            'curl -s -X POST http://127.0.0.1:3210/getNewsRefreshLogs '
            '-H "Content-Type: application/json" '
            '-H "Authorization: Bearer 850367bc6d5fe8a4a53f267f5c308ac6d2ec1474d1764fe9" '
            '-d \'{"limit":1}\' | python3 -c "import sys,json; d=json.load(sys.stdin); print(type(d).__name__, len(d) if isinstance(d,list) else d)"',
        )
    )

    print("\n## TradingServer only check")
    print(run("geegoo-signal", "curl -sf --connect-timeout 5 http://43.134.94.87:7000/health || echo FAIL"))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
