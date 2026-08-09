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
    # insert dummy log on HK data mongo via API... use refresh if possible
    print("=== insert test log on data ===")
    print(
        run(
            "geegoo-data",
            '''python3 - <<'PY'
from pymongo import MongoClient
import datetime
c = MongoClient()
db = c['aidb']
doc = {
    'run_id': 'refresh-test-delete-001',
    'run_date': datetime.datetime.now().strftime('%Y-%m-%d'),
    'status': 'success',
    'started_at': '2026-08-01 15:00:00',
    'finished_at': '2026-08-01 15:01:00',
    'total_stocks': 1,
    'success_stocks': 1,
    'failed_stocks': 0,
    'total_news': 1,
    'details': [],
}
db.news_refresh_log.insert_one(doc)
print('inserted', doc['run_id'])
PY''',
        )
    )

    print("\n=== catalog get/delete ===")
    print(
        run(
            "geegoo-signal",
            f'''curl -s -X POST http://127.0.0.1:3210/getNewsRefreshLogs -H "Content-Type: application/json" -H "Authorization: Bearer {KEY}" -d '{{"limit":1}}' '''
        )
    )
    print(
        run(
            "geegoo-signal",
            f'''curl -s -X POST http://127.0.0.1:3210/deleteNewsRefreshLogs -H "Content-Type: application/json" -H "Authorization: Bearer {KEY}" -d '{{"run_id":"refresh-test-delete-001"}}' '''
        )
    )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
