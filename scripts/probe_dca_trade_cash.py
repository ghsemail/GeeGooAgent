#!/usr/bin/env python3
from pathlib import Path
import paramiko, json

REMOTE = r'''
import json, urllib.request
from bson import ObjectId
from pymongo import MongoClient

def env(k):
    for line in open("/home/ubuntu/apps/GeeGooBot/.env"):
        if line.startswith(k + "="):
            return line.strip().split("=", 1)[1]
    return ""

db = MongoClient(env("GEEGOO_BOT_MONGO_URI"))[env("GEEGOO_BOT_MONGO_DB") or "QT_DB"]
user = db.user.find_one({"_id": ObjectId("6366170502d5c175fd586fe8")})
trade = user.get("trade") or {}
print("user.trade", json.dumps(trade, ensure_ascii=False))

bot = db.dca_bot.find_one({"_id": ObjectId("6a449bc14b77fe41d732b809")})
print("order_size", bot.get("order_size"))
print("advanced_setting", bot.get("advanced_setting"))
print("maximum_price", (bot.get("advanced_setting") or {}).get("maximum_price_to_open_deal"))

# cash probe via internal API if possible
key = env("GEEGOO_BOT_APP_API_KEY")
body = {"user_id": "6366170502d5c175fd586fe8"}
req = urllib.request.Request("http://127.0.0.1:3100/getCash", data=json.dumps(body).encode(), headers={"Content-Type":"application/json","Authorization":"Bearer "+key}, method="POST")
try:
    with urllib.request.urlopen(req, timeout=30) as r:
        print("cash", r.read().decode()[:500])
except Exception as e:
    print("cash_err", e)

mcp_tok = user.get("mcp",{}).get("mcp_token","")
api_key = env("GEEGOO_BOT_APP_API_KEY")
for path, payload in [
    ("/getCash", {"mcp_token": mcp_tok}),
    ("/getBroker", {"mcp_token": mcp_tok, "code": "SPCX.US"}),
]:
    req = urllib.request.Request("http://118.195.135.97:3120"+path, data=json.dumps(payload).encode(), headers={"Content-Type":"application/json","Authorization":"Bearer "+api_key}, method="POST")
    try:
        with urllib.request.urlopen(req, timeout=30) as r:
            print(path, r.read().decode()[:600])
    except Exception as e:
        err = e.read().decode() if hasattr(e,"read") else str(e)
        print(path, "err", err[:400])
'''

cfg = json.loads(Path(r'C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json').read_text(encoding='utf-8'))
s = cfg['targets']['geegoo-bot']['ssh']
c = paramiko.SSHClient(); c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
c.connect(s['host'], username=s['user'], password=s.get('password'), timeout=30)
p='/tmp/tr.py'
with c.open_sftp().file(p,'w') as f: f.write(REMOTE)
_,o,e=c.exec_command('python3 '+p,timeout=90)
print((o.read()+e.read()).decode('utf-8','replace'))
c.close()
