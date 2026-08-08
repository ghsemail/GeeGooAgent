#!/usr/bin/env python3
from pathlib import Path
import paramiko, json

REMOTE = r'''
import json
from bson import ObjectId
from pymongo import MongoClient

def env(k):
    for line in open("/home/ubuntu/apps/GeeGooBot/.env"):
        if line.startswith(k + "="):
            return line.strip().split("=", 1)[1]
    return ""

db = MongoClient(env("GEEGOO_BOT_MONGO_URI"))[env("GEEGOO_BOT_MONGO_DB") or "QT_DB"]
uid = ObjectId("6366170502d5c175fd586fe8")
user = db.user.find_one({"_id": uid})
print("user keys sample", [k for k in user.keys() if any(x in k.lower() for x in ['trade','futu','bind','broker','agent','mcp','switch'])])
for k in ['trade_bind','futu','broker','trade_connection','trade_account','agent','agent_switch','intraday_agent','auto_trade']:
    if k in user: print(k, user.get(k))

# any trade connection collection
for coll in db.list_collection_names():
    if 'trade' in coll.lower() or 'futu' in coll.lower() or 'bind' in coll.lower():
        n = db[coll].count_documents({"user_id": str(uid)})
        if n: print(coll, 'docs for user', n)

bid = ObjectId("6a449bc14b77fe41d732b809")
info = db.dca_info.find_one({"bot_id": bid})
print('dca_info full-ish', json.dumps({k:info.get(k) for k in info.keys() if k!='_id'}, default=str, ensure_ascii=False)[:3000])

# logs with place order attempt
for lg in db.dca_log.find({"bot_id": bid}).sort("_id", -1).limit(200):
    log = lg.get("log") or {}
    s = json.dumps(log, ensure_ascii=False)
    if 'place' in s.lower() or 'order' in s.lower() and ('fail' in s.lower() or 'skip' in s.lower() or '未' in s):
        print('hit', lg.get('time'), s[:500])
'''

cfg = json.loads(Path(r'C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json').read_text(encoding='utf-8'))
s = cfg['targets']['geegoo-bot']['ssh']
c = paramiko.SSHClient(); c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
c.connect(s['host'], username=s['user'], password=s.get('password'), timeout=30)
p='/tmp/u.py'
with c.open_sftp().file(p,'w') as f: f.write(REMOTE)
_,o,e=c.exec_command('python3 '+p,timeout=120)
print((o.read()+e.read()).decode('utf-8','replace'))
c.close()
