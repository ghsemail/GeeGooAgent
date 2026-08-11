#!/usr/bin/env python3
from __future__ import annotations
import json
from pathlib import Path
import paramiko

DEPLOY = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")
REMOTE = r"""
import os
from bson import ObjectId
from pymongo import MongoClient
os.chdir('/home/ubuntu/apps/GeeGooBot')
for line in open('.env'):
    line=line.strip()
    if line and not line.startswith('#') and '=' in line:
        k,v=line.split('=',1); os.environ.setdefault(k.strip(), v.strip().strip('"').strip("'"))
db=MongoClient(os.environ['GEEGOO_BOT_MONGO_URI'])[os.environ['GEEGOO_BOT_MONGO_DB']]
bid=ObjectId('6908395ac968cf04c9115041')
b=db.grid_bot.find_one({'_id':bid})
info=db.grid_info.find_one({'bot_id':bid})
print('grid param', b.get('grid'))
print('order_size', b.get('order_size'))
print('info buy_grid', info.get('buy_grid'))
print('info sell_grid', info.get('sell_grid'))
print('info buy_position', info.get('buy_position'))
print('info sell_position', info.get('sell_position'))
print('info current_grid', info.get('current_grid'))
bad=[]
for doc in db.grid_info.find({}, {'bot_id':1,'buy_grid':1,'sell_grid':1,'current_grid':1}):
    bg=doc.get('buy_grid') or []
    sg=doc.get('sell_grid') or []
    if isinstance(bg, list) and len(bg)>0 and all((x==0 or x==0.0) for x in bg):
        bad.append((str(doc.get('bot_id')), bg, sg, doc.get('current_grid')))
print('all_zero_buy_grid bots', len(bad))
for x in bad:
    bot=db.grid_bot.find_one({'_id': ObjectId(x[0])}) if len(x[0])==24 else None
    name=(bot or {}).get('botname'); code=(bot or {}).get('code')
    print(' ', name, code, x)
"""

def main():
    cfg=json.loads(DEPLOY.read_text(encoding='utf-8'))
    s=cfg['targets']['geegoo-bot']['ssh']
    c=paramiko.SSHClient(); c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    c.connect(s['host'], username=s['user'], password=s.get('password'), timeout=30)
    p='/tmp/probe_zhaojin2.py'
    with c.open_sftp().file(p,'w') as f: f.write(REMOTE)
    _,o,e=c.exec_command('python3 '+p, timeout=120)
    print((o.read()+e.read()).decode('utf-8','replace')); c.close()
if __name__=='__main__': main()
