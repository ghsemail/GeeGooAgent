#!/usr/bin/env python3
from pathlib import Path
import paramiko, json

REMOTE = r'''
import json
from bson import ObjectId
from pymongo import MongoClient
from datetime import datetime, timedelta

def env(k):
    for line in open("/home/ubuntu/apps/GeeGooBot/.env"):
        if line.startswith(k + "="):
            return line.strip().split("=", 1)[1]
    return ""

db = MongoClient(env("GEEGOO_BOT_MONGO_URI"))[env("GEEGOO_BOT_MONGO_DB") or "QT_DB"]
bid = ObjectId("6a449bc14b77fe41d732b809")
bot = db.dca_bot.find_one({"_id": bid})
info = db.dca_info.find_one({"bot_id": bid})

print("=== order_size / advanced ===")
print(json.dumps(bot.get("order_size"), ensure_ascii=False))
print(json.dumps(bot.get("advanced_setting"), ensure_ascii=False))
print("status", info.get("status"), "total_safety", info.get("total_safety_count"), "open_deals", info.get("open_deals"), "last_order", info.get("last_order_time"))

print("\n=== err_log signal_buy ===")
start = datetime(2026,8,7)
for row in db.err_log.find({"bot_id": str(bid)}).sort("time", -1).limit(20):
    print(row.get("time"), row.get("order_type"), row.get("message") or row.get("err") or row.get("error"))
for row in db.err_log.find({"bot_id": str(bid), "time": {"$gte": start}}).sort("time", 1):
    print("aug", row.get("time"), row.get("order_type"), str(row.get("message") or row.get("error") or row)[:400])

# also by user
uid = "6366170502d5c175fd586fe8"
for row in db.err_log.find({"user_id": uid, "time": {"$gte": start}}).sort("time", -1).limit(15):
    if "SPCX" in str(row).upper() or str(row.get("bot_id")) == str(bid):
        print("user_err", row.get("time"), row.get("order_type"), str(row.get("message") or row.get("error") or row)[:400])

print("\n=== grep bot service logs ===")
'''

cfg = json.loads(Path(r'C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json').read_text(encoding='utf-8'))
s = cfg['targets']['geegoo-bot']['ssh']
c = paramiko.SSHClient(); c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
c.connect(s['host'], username=s['user'], password=s.get('password'), timeout=30)
p='/tmp/e.py'
with c.open_sftp().file(p,'w') as f: f.write(REMOTE)
_,o,e=c.exec_command('python3 '+p + "; grep -i '6a449bc14b77fe41d732b809\\|SPCX\\|购买力\\|DCA买入' /home/ubuntu/apps/GeeGooBot/*.out 2>/dev/null | tail -30", timeout=120)
print((o.read()+e.read()).decode('utf-8','replace'))
c.close()
