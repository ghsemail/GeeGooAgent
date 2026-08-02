#!/usr/bin/env python3
import json
from pathlib import Path
import paramiko

DEPLOY = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")
cfg = json.loads(DEPLOY.read_text(encoding="utf-8"))
KEY = "850367bc6d5fe8a4a53f267f5c308ac6d1764fe9"


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
    key = "850367bc6d5fe8a4a53f267f5c308ac6d2ec1474d1764fe9"
    print(run("geegoo-data", f'''python3 - <<'PY'
import json, urllib.request
req = urllib.request.Request(
    "http://127.0.0.1:3300/getNewsRefreshLogs",
    data=json.dumps({{"limit": 1}}).encode(),
    headers={{"Content-Type": "application/json"}},
    method="POST",
)
logs = json.load(urllib.request.urlopen(req))
print("keys", list(logs[0].keys()) if logs else None)
print("run_id", logs[0].get("run_id") if logs else None)
rid = logs[0].get("run_id") if logs else None
if rid:
    req2 = urllib.request.Request(
        "http://127.0.0.1:3300/deleteNewsRefreshLogs",
        data=json.dumps({{"run_id": rid}}).encode(),
        headers={{"Content-Type": "application/json"}},
        method="POST",
    )
    print("delete", urllib.request.urlopen(req2).read().decode())
    req3 = urllib.request.Request(
        f"http://127.0.0.1:3210/deleteNewsRefreshLogs",
        data=json.dumps({{"run_id": rid}}).encode(),
        headers={{"Content-Type": "application/json", "Authorization": "Bearer {key}"}},
        method="POST",
    )
    try:
        print("catalog delete re-insert test skipped - already deleted")
    except Exception as e:
        print(e)
PY'''))

    print("\n=== news doc fields ===")
    print(
        run(
            "geegoo-data-cn",
            '''python3 - <<'PY'
import json, urllib.request
req = urllib.request.Request(
    "http://127.0.0.1:3300/getStockNews",
    data=json.dumps({"stock_list":[{"code":"000858.SZ","name":{"init":"五粮液"}}],"language":"cn"}).encode(),
    headers={"Content-Type": "application/json"},
    method="POST",
)
items = json.load(urllib.request.urlopen(req))
n = items[0]
print(json.dumps({"title": n.get("title"), "summary": n.get("summary"), "publisher": n.get("publisher")}, ensure_ascii=False, indent=2)[:1200])
PY''',
        )
    )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
