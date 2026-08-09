#!/usr/bin/env python3
from __future__ import annotations
import json
from pathlib import Path
import paramiko

DCA_BOT = "6a449bc14b77fe41d732b809"
USER_ID = "6366170502d5c175fd586fe8"
API_KEY = "sk-7a2b3c4d5e6f7g8h9i0j1k2l3m4n5o6p7q8r9s0t1u2v3w4x5y6z7a8b9c0d1e2"
MCP = "http://118.195.135.97:3120"
TARGET_DATE = "2026-08-08"

REMOTE = """
import json, os, urllib.request, urllib.error
from datetime import datetime, timedelta
from bson import ObjectId
from pymongo import MongoClient

DCA_BOT = """ + json.dumps(DCA_BOT) + r"""
USER_ID = """ + json.dumps(USER_ID) + r"""
API_KEY = """ + json.dumps(API_KEY) + r"""
MCP = """ + json.dumps(MCP) + r"""
TARGET = """ + json.dumps(TARGET_DATE) + r"""

os.chdir('/home/ubuntu/apps/GeeGooBot')
for line in open('.env'):
    line=line.strip()
    if line and not line.startswith('#') and '=' in line:
        k,v=line.split('=',1); os.environ.setdefault(k.strip(), v.strip().strip('"').strip("'"))
db=MongoClient(os.environ['GEEGOO_BOT_MONGO_URI'])[os.environ['GEEGOO_BOT_MONGO_DB']]
user=db['user'].find_one({'_id':ObjectId(USER_ID)})
tok=user.get('mcp',{}).get('mcp_token','')

def mcp(path, body):
    req=urllib.request.Request(MCP+path,data=json.dumps(body).encode(),headers={'Content-Type':'application/json','Authorization':'Bearer '+API_KEY},method='POST')
    with urllib.request.urlopen(req,timeout=90) as r: return json.loads(r.read().decode())

print('collections', [c for c in db.list_collection_names() if 'dca' in c.lower()])

# find bot doc in any collection
for coll in ['dca_bot','trade_bot','bot']:
    doc=db[coll].find_one({'_id':ObjectId(DCA_BOT)}) if coll in db.list_collection_names() else None
    if doc:
        print('found in', coll)
        print(json.dumps({k:doc.get(k) for k in ['botname','code','switch','frequency','signal','order_size','user_id','attitude']},ensure_ascii=False,default=str)[:2500])

for coll in ['dca_info','trade_info']:
    if coll in db.list_collection_names():
        info=db[coll].find_one({'bot_id':DCA_BOT})
        if info:
            print(coll, json.dumps({k:info.get(k) for k in ['status','switch','position','last_run','error','notice']},ensure_ascii=False,default=str))

print('\n=== getDCABotLog ===')
r=mcp('/getDCABotLog',{'mcp_token':tok,'bot_id':DCA_BOT,'hold':False,'filter':'all'})
print('code', r.get('code'), 'keys', list(r.keys()))
data=r.get('data') if isinstance(r.get('data'),dict) else r
if isinstance(data,dict):
    print('info', json.dumps(data.get('info'), ensure_ascii=False)[:800])
    logs=data.get('log') or data.get('log_sr') or []
    print('log_count', len(logs))
    for row in logs[:30]:
        pos=row.get('position') or {}
        sig=row.get('signal') or {}
        print(row.get('time'), 'next_opt=',row.get('next_opt'),'opt=',pos.get('opt'),'order_status=',pos.get('order_status'),'qty=',pos.get('qty'))
        if sig: print('  signal', json.dumps(sig, ensure_ascii=False)[:400])
        if row.get('reason') or row.get('message'): print('  reason', row.get('reason') or row.get('message'))

print('\n=== dca_log mongo ===')
start=datetime.strptime(TARGET,'%Y-%m-%d'); end=start+timedelta(days=2)
for coll in ['dca_log','trade_log']:
    if coll not in db.list_collection_names(): continue
    rows=list(db[coll].find({'bot_id':DCA_BOT,'time':{'$gte':start,'$lt':end}}).sort('time',1))
    print(coll,'rows',len(rows))
    for row in rows[:40]:
        lg=row.get('log') or {}
        pos=lg.get('position') or {}
        print(row.get('time'),'next_opt=',lg.get('next_opt'),'opt=',pos.get('opt'),'order=',pos.get('order_status'))
        if lg.get('signal'): print(' ',json.dumps(lg.get('signal'),ensure_ascii=False)[:500])

print('\n=== getDCABotProfit ===')
r=mcp('/getDCABotProfit',{'mcp_token':tok,'bot_id':DCA_BOT})
print(json.dumps(r,ensure_ascii=False)[:1500])

print('\n=== attitude / signal check ===')
r=mcp('/getBotYesterdayAttitude',{'mcp_token':tok,'bot_id':DCA_BOT,'code':'SPCX.US'})
print('yesterday_attitude', json.dumps(r, ensure_ascii=False)[:1000])
"""

def main():
    cfg=json.loads(Path(r'C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json').read_text())
    s=cfg['targets']['geegoo-bot']['ssh']
    c=paramiko.SSHClient(); c.set_missing_host_key_policy(paramiko.AutoAddPolicy()); c.connect(s['host'],username=s['user'],password=s.get('password'),timeout=30)
    p='/tmp/probe_dca2.py'
    with c.open_sftp().file(p,'w') as f: f.write(REMOTE)
    _,o,e=c.exec_command('python3 '+p,timeout=240)
    print((o.read()+e.read()).decode('utf-8','replace')); c.close()

if __name__=='__main__': main()
