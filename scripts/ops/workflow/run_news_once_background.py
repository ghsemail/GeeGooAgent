#!/usr/bin/env python3
import json
import time
from pathlib import Path
import paramiko

cfg = json.loads(Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json").read_text(encoding="utf-8"))
data = cfg["targets"]["geegoo-tradingdata"]["ssh"]
c = paramiko.SSHClient()
c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
c.connect(data["host"], username=data["user"], password=data["password"], timeout=30)

print("=== start news-worker -once in background ===")
_, stdout, _ = c.exec_command(
    "cd /root/apps/GeeGooData && set -a && source .env && set +a && "
    "nohup ./bin/news-worker -once > /tmp/news-once.out 2>&1 & echo started",
    timeout=30,
)
print(stdout.read().decode())

for i in range(60):
    time.sleep(30)
    _, stdout, _ = c.exec_command(
        "tail -n 3 /tmp/news-once.out 2>/dev/null; pgrep -f 'news-worker -once' >/dev/null && echo RUNNING || echo DONE",
        timeout=20,
    )
    block = stdout.read().decode("utf-8", errors="replace")
    print(f"--- poll {i+1} ---")
    print(block)
    if "DONE" in block and "refresh complete" in block:
        break
    if "DONE" in block and i > 2:
        _, stdout2, _ = c.exec_command("cat /tmp/news-once.out | tail -20", timeout=30)
        print(stdout2.read().decode("utf-8", errors="replace"))
        break

py = (
    "import sys,json;d=json.load(sys.stdin);n=d[0] if d else {};"
    "t=n.get('title',{});s=n.get('summary',{});cn=t.get('cn') or '';sc=s.get('cn') or '';"
    "print('title.cn',cn[:100]);print('title_zh',any('\\u4e00'<=c<='\\u9fff' for c in cn));"
    "print('summary_zh',any('\\u4e00'<=c<='\\u9fff' for c in sc))"
)
for code in ("TSLA.US", "00700.HK"):
    payload = json.dumps({"stock_list": [{"code": code, "name": {"init": code}}], "language": "cn"})
    cmd = (
        f"curl -sf -X POST http://127.0.0.1:3300/getStockNews -H 'Content-Type: application/json' "
        f"-d '{payload}' | python3 -c \"{py}\""
    )
    print(f"=== verify {code} ===")
    _, stdout, _ = c.exec_command(cmd, timeout=30)
    print(stdout.read().decode("utf-8", errors="replace"))

c.close()
