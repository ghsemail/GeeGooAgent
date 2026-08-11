#!/usr/bin/env python3
from __future__ import annotations
import json
from pathlib import Path
import paramiko

DEPLOY = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")
DCA_BOT = "6a449bc14b77fe41d732b809"

REMOTE = r"""
import json, os, urllib.request
from datetime import datetime, timedelta
from bson import ObjectId
from pymongo import MongoClient
from collections import Counter

DCA_BOT = """ + json.dumps(DCA_BOT) + r"""
oid = ObjectId(DCA_BOT)

os.chdir('/home/ubuntu/apps/GeeGooBot')
for line in open('.env'):
    line=line.strip()
    if line and not line.startswith('#') and '=' in line:
        k,v=line.split('=',1); os.environ.setdefault(k.strip(), v.strip().strip('"').strip("'"))
db=MongoClient(os.environ['GEEGOO_BOT_MONGO_URI'])[os.environ['GEEGOO_BOT_MONGO_DB']]

since = datetime.utcnow() - timedelta(days=2)
rows = list(db['dca_log'].find({'bot_id': oid, 'type': 'DCA', 'time': {'$gte': since}}).sort('time', -1))
print('=== type=DCA last 2d ===', len(rows))
for row in rows[:40]:
    lg = row.get('log') or {}
    pos = lg.get('position') or {}
    buy = lg.get('buy_signal') or {}
    sell = lg.get('sell_signal') or {}
    ta = lg.get('trade_agent') or {}
    def side(s):
        if not isinstance(s, dict):
            return str(s)[:80]
        parts=[]
        # common shapes: {SAR:{signal:1}, MACD:{signal:0}, value:1} or nested
        if 'value' in s: parts.append('value=%s'%s.get('value'))
        for k,v in s.items():
            if k in ('value','name'): continue
            if isinstance(v, dict) and 'signal' in v:
                parts.append('%s=%s'%(k,v.get('signal')))
        return ','.join(parts) or str(s)[:120]
    print(row.get('time'), 'next=', lg.get('next_opt'), 'sig_date=', lg.get('signal_date'),
          'buy=', side(buy), 'sell=', side(sell),
          'opt=', pos.get('opt'), 'status_related=', pos.get('order_status'),
          'agent=', ta.get('result'), ta.get('confidence'))

print('\nnext_opt', Counter(str((r.get('log') or {}).get('next_opt')) for r in rows))
print('sig_date unique sample', list(Counter(str((r.get('log') or {}).get('signal_date')) for r in rows).items())[:15])

# DCASR cadence
sr = list(db['dca_log'].find({'bot_id': oid, 'type': 'DCASR', 'time': {'$gte': since}}).sort('time', -1).limit(20))
print('\n=== DCASR recent ===', 'total_2d', db['dca_log'].count_documents({'bot_id': oid, 'type': 'DCASR', 'time': {'$gte': since}}))
for row in sr[:8]:
    lg=row.get('log') or {}
    print(row.get('time'), 'next=', lg.get('next_opt'), 'keys=', list(lg.keys()))

# scheduler jobs
print('\n=== scheduler DB / redis? ===')
for name in db.list_collection_names():
    if 'sched' in name.lower() or 'job' in name.lower() or 'task' in name.lower():
        print('coll', name, db[name].count_documents({}))

# live signal from TradingSignal
bot = db['dca_bot'].find_one({'_id': oid})
sig = bot.get('signal') or {}
buy_rules = sig.get('buy_signal')
sell_rules = sig.get('sell_signal')
print('\n=== live GetBotSignal ===')
# find signal URL from env
env_keys = [k for k in os.environ if 'SIGNAL' in k.upper() or 'GEEGOO' in k.upper()]
print('env related', {k: os.environ[k] for k in sorted(env_keys) if 'PASS' not in k and 'MONGO' not in k and 'SECRET' not in k and 'TOKEN' not in k})

# try call signal service like bot does
import urllib.request
body = {
  'code': 'SPCX.US',
  'frequency': '60m',
  'buy_signal': buy_rules,
  'sell_signal': sell_rules,
}
# discover endpoint from GeeGooBot config
print('trying signal endpoints...')
for base in [
    os.environ.get('GEEGOO_BOT_SIGNAL_URL',''),
    os.environ.get('SIGNAL_URL',''),
    'http://127.0.0.1:3300',
    'http://146.56.225.252:3300',
]:
    if not base: continue
    for path in ['/getBotSignal', '/signal/getBotSignal', '/api/getBotSignal']:
        url = base.rstrip('/') + path
        try:
            req = urllib.request.Request(url, data=json.dumps(body).encode(), headers={'Content-Type':'application/json'}, method='POST')
            with urllib.request.urlopen(req, timeout=20) as r:
                data = json.loads(r.read().decode())
            print('OK', url, json.dumps(data, ensure_ascii=False)[:800])
        except Exception as e:
            print('FAIL', url, type(e).__name__, e)
"""

def main():
    cfg=json.loads(Path(r'C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json').read_text(encoding='utf-8'))
    s=cfg['targets']['geegoo-bot']['ssh']
    c=paramiko.SSHClient(); c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    c.connect(s['host'], username=s['user'], password=s.get('password'), timeout=30)
    p='/tmp/probe_spacex_hourly3.py'
    with c.open_sftp().file(p,'w') as f: f.write(REMOTE)
    _,o,e=c.exec_command('python3 '+p, timeout=240)
    print((o.read()+e.read()).decode('utf-8','replace')); c.close()
if __name__=='__main__': main()
