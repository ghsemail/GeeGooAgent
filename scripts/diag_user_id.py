#!/usr/bin/env python3
import json
from pathlib import Path
import paramiko

DEPLOY = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")
TOK = "mcp_HVTSYfumrCexAU66EutTM4v2A5aGYXiF"


def run(target, cmd, timeout=60):
    cfg = json.loads(DEPLOY.read_text(encoding="utf-8"))
    s = cfg["targets"][target]["ssh"]
    c = paramiko.SSHClient()
    c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    c.connect(s["host"], username=s["user"], password=s.get("password"), timeout=30)
    _, o, e = c.exec_command(cmd, timeout=timeout)
    print((o.read() or e.read()).decode("utf-8", errors="replace"))


def main():
    mcp = run.__doc__
    run(
        "geegoo-tradingbot",
        f"""python3 - <<'PY'
import json, urllib.request
key=open('/home/ubuntu/apps/GeeGooBot/.env').read().split('GEEGOO_BOT_MCP_API_KEY=')[1].split()[0]
req=urllib.request.Request('http://127.0.0.1:3120/checkTradingDay', data=json.dumps({{"mcp_token":"{TOK}","code":"SPCX.US"}}).encode(), headers={{"Authorization":"Bearer "+key,"Content-Type":"application/json"}}, method='POST')
print(urllib.request.urlopen(req, timeout=20).read().decode())
PY""",
    )
    run(
        "geegoo-tradingbot",
        f"""python3 - <<'PY'
from pymongo import MongoClient
c=MongoClient('mongodb://127.0.0.1:27017')
db=c['QT_DB']
doc=db.user.find_one({{'mcp.mcp_token':'{TOK}'}})
print('doc', doc and str(doc.get('_id')))
uid=str(doc['_id']) if doc else None
if uid:
 import urllib.request, json
 body=json.dumps({{'user_id':uid}}).encode()
 r=urllib.request.urlopen(urllib.request.Request('http://127.0.0.1:3140/checkUser', data=body, headers={{'Content-Type':'application/json'}}, method='POST'), timeout=10)
 print('checkUser', r.read().decode())
PY""",
    )


if __name__ == "__main__":
    main()
