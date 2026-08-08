#!/usr/bin/env python3
"""Fix CN mcp-api token load, deploy agent, run live CN premarket_market for yesterday."""
from __future__ import annotations

import json
import subprocess
import sys
from datetime import date, timedelta
from pathlib import Path

import paramiko

DEPLOY = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")


def ssh_run(target: str, cmd: str, timeout: int = 600) -> str:
    cfg = json.loads(DEPLOY.read_text(encoding="utf-8"))
    ssh = cfg["targets"][target]["ssh"]
    c = paramiko.SSHClient()
    c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    c.connect(ssh["host"], username=ssh["user"], password=ssh.get("password"), timeout=30)
    _, o, e = c.exec_command(cmd, timeout=timeout)
    out = (o.read() + e.read()).decode("utf-8", errors="replace")
    c.close()
    return out


def main() -> int:
    yesterday = (date.today() - timedelta(days=1)).isoformat()
    print(f"=== 1) Restart GeeGooBot mcp-api (load CN token) ===")
    print(ssh_run("geegoo-bot", "cd /home/ubuntu/apps/GeeGooBot && printf '4\\n' | bash start.sh", timeout=180))

    print(f"\n=== 2) Verify CN checkTradingDay ===")
    remote_verify = r'''
import json, os, urllib.request, urllib.error
from pathlib import Path
env = {}
for line in Path("/home/ubuntu/apps/GeeGooBot/.env").read_text().splitlines():
    if "=" in line and not line.strip().startswith("#"):
        k,v=line.split("=",1); env[k]=v
mcp_key = env.get("GEEGOO_BOT_MCP_API_KEY","")
mcp_token = json.load(open(os.path.expanduser("~/.geegoo/config.json"))).get("mcp_token","")
body = json.dumps({"mcp_token": mcp_token, "code": "000001.SZ"}).encode()
req = urllib.request.Request("http://127.0.0.1:3120/checkTradingDay", data=body,
    headers={"Content-Type":"application/json","Authorization": f"Bearer {mcp_key}"}, method="POST")
try:
    with urllib.request.urlopen(req, timeout=20) as r:
        print("checkTradingDay", r.status, r.read().decode()[:400])
except urllib.error.HTTPError as e:
    print("checkTradingDay FAIL", e.code, e.read()[:400])
body2 = json.dumps({"mcp_token": mcp_token, "market": "CN", "limit": 2}).encode()
req2 = urllib.request.Request("http://127.0.0.1:3120/getMarketNews", data=body2,
    headers={"Content-Type":"application/json","Authorization": f"Bearer {mcp_key}"}, method="POST")
try:
    with urllib.request.urlopen(req2, timeout=30) as r:
        print("getMarketNews", r.status, r.read().decode()[:300])
except urllib.error.HTTPError as e:
    print("getMarketNews FAIL", e.code, e.read()[:300])
'''
    print(ssh_run("geegoo-bot", f"python3 <<'PY'\n{remote_verify}\nPY", timeout=60))

    print(f"\n=== 3) Install/update GeeGooAgent on server ===")
    install = json.loads(DEPLOY.read_text(encoding="utf-8"))["targets"]["geegoo-agent"]["install_cmd"]
    print(ssh_run("geegoo-agent", install, timeout=900))

    print(f"\n=== 4) Run live CN premarket_market for {yesterday} ===")
    run_cmd = (
        f"export PATH=$HOME/.geegoo/bin:$PATH; "
        f"timeout 900 $HOME/.geegoo/bin/geegoo run "
        f"--config $HOME/.geegoo/config.json --market CN --report-date {yesterday} premarket_market 2>&1"
    )
    print(ssh_run("geegoo-agent", run_cmd, timeout=920))

    print(f"\n=== 5) Fetch report ===")
    report_path = f"/home/ubuntu/.geegoo/data/reports/{yesterday}/market-CN-market_premarket.md"
    report = ssh_run("geegoo-agent", f"cat {report_path} 2>/dev/null || echo MISSING", timeout=30)
    print(report_path)
    print("=" * 60)
    print(report)
    print("=" * 60)

    mcp_fetch = f'''
import json, os, urllib.request
cfg=json.load(open(os.path.expanduser("~/.geegoo/config.json")))
tok=cfg.get("mcp_token",""); key=cfg.get("geegoo_api_key") or cfg.get("api_key","")
body=json.dumps({{"mcp_token":tok,"market":"CN","report_date":"{yesterday}"}}).encode()
req=urllib.request.Request("http://118.195.135.97:3120/getMarketPremarketReport", data=body,
    headers={{"Content-Type":"application/json","Authorization":"Bearer "+key}}, method="POST")
with urllib.request.urlopen(req, timeout=30) as r:
    d=json.loads(r.read().decode())
data=d.get("data") or d
print("MCP result=", data.get("result"), "confidence=", data.get("confidence"))
print("summary=", data.get("summary"))
print((data.get("report") or "")[:6000])
'''
    print("\n=== MCP stored report ===")
    print(ssh_run("geegoo-agent", f"python3 <<'PY'\n{mcp_fetch}\nPY", timeout=60))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
