#!/usr/bin/env python3
import json, re
from pathlib import Path
import paramiko

DEPLOY = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")
TOK = "mcp_HVTSYfumrCexAU66EutTM4v2A5aGYXiF"


def run(cmd):
    cfg = json.loads(DEPLOY.read_text(encoding="utf-8"))
    s = cfg["targets"]["geegoo-tradingbot"]["ssh"]
    c = paramiko.SSHClient()
    c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    c.connect(s["host"], username=s["user"], password=s.get("password"), timeout=30)
    _, o, _ = c.exec_command(cmd, timeout=60)
    return o.read().decode("utf-8", errors="replace")


def main():
    out = run(
        f"""KEY=$(grep GEEGOO_BOT_MCP_API_KEY /home/ubuntu/apps/GeeGooBot/.env|cut -d= -f2-); """
        f"""curl -s -X POST http://127.0.0.1:3120/getSinglePromptTemplate """
        f"""-H "Authorization: Bearer $KEY" -H 'Content-Type: application/json' """
        f"""-d '{{"mcp_token":"{TOK}","type":"tech","period":"daily"}}'"""
    )
    m = re.search(r'"user_id":"([^"]+)"', out)
    uid = m.group(1) if m else ""
    print("resolved user_id:", uid)
    if uid:
        body = json.dumps({"user_id": uid})
        print(run(f"curl -s -X POST http://127.0.0.1:3140/checkUser -H 'Content-Type: application/json' -d '{body}'"))


if __name__ == "__main__":
    main()
