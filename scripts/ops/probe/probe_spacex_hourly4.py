#!/usr/bin/env python3
from __future__ import annotations
import json
from pathlib import Path
import paramiko

DEPLOY = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")
DCA_BOT = "6a449bc14b77fe41d732b809"

REMOTE = r"""
import json, os, urllib.request
from collections import Counter
from bson import ObjectId
from pymongo import MongoClient

oid = ObjectId(""" + json.dumps(DCA_BOT) + r""")
os.chdir('/home/ubuntu/apps/GeeGooBot')
for line in open('.env'):
    line=line.strip()
    if line and not line.startswith('#') and '=' in line:
        k,v=line.split('=',1); os.environ.setdefault(k.strip(), v.strip().strip('"').strip("'"))
db=MongoClient(os.environ['GEEGOO_BOT_MONGO_URI'])[os.environ['GEEGOO_BOT_MONGO_DB']]

print('type counts', list(db['dca_log'].aggregate([
    {'$match': {'bot_id': oid}},
    {'$group': {'_id': '$type', 'n': {'$sum': 1}}},
])))

# last 50 of each type
for typ in ['DCA', 'DCASR', None]:
    q={'bot_id': oid}
    if typ: q['type']=typ
    else: q['type']={'$exists': False}
    rows=list(db['dca_log'].find(q).sort('time',-1).limit(25))
    print('\n=== type', typ, 'latest', len(rows), '===')
    for row in rows[:20]:
        lg=row.get('log') or {}
        pos=lg.get('position') or {}
        ta=lg.get('trade_agent') or {}
        buy=lg.get('buy_signal')
        # show raw buy briefly
        bv=None
        if isinstance(buy, dict):
            bv=buy.get('value', buy)
        print(row.get('time'), 'stored_type=', row.get('type'),
              'next=', lg.get('next_opt'), 'sig_date=', lg.get('signal_date'),
              'buy_val=', bv if not isinstance(bv,dict) else str(bv)[:80],
              'agent=', ta.get('result'), 'opt=', pos.get('opt'))

# only next_opt buy/sell among last 80 any type
rows=list(db['dca_log'].find({'bot_id': oid}).sort('time',-1).limit(200))
sig_rows=[r for r in rows if (r.get('log') or {}).get('next_opt') in ('buy','sell')]
print('\n=== among last 200 logs, next_opt buy/sell ===', len(sig_rows))
for row in sig_rows[:30]:
    lg=row.get('log') or {}
    ta=lg.get('trade_agent') or {}
    print(row.get('time'), row.get('type'), 'next=', lg.get('next_opt'),
          'sig_date=', lg.get('signal_date'), 'agent=', ta.get('result'), ta.get('confidence'))

# gaps between buy signals
buys=[r for r in reversed(sig_rows) if (r.get('log') or {}).get('next_opt')=='buy']
print('\n=== buy signal intervals (minutes) ===')
prev=None
for row in buys[-20:]:
    t=row.get('time')
    if prev is not None:
        try:
            delta=(t-prev).total_seconds()/60.0
            print(prev, '->', t, 'delta_min=', round(delta,1), 'sig_date=', (row.get('log') or {}).get('signal_date'))
        except Exception as e:
            print('delta err', e, t, prev)
    prev=t

# live signal via GeeGooSignal
bot=db['dca_bot'].find_one({'_id': oid})
sig=bot.get('signal') or {}
body={
  'code': bot.get('code'),
  'frequency': bot.get('frequency'),
  'buy_signal': sig.get('buy_signal'),
  'sell_signal': sig.get('sell_signal'),
}
base=os.environ.get('GEEGOO_SIGNAL_SIGNAL_API_URL','http://146.56.225.252:3200')
key=os.environ.get('GEEGOO_SIGNAL_SIGNAL_API_KEY','')
print('\n=== live GetBotSignal', base, '===')
for path in ['/getBotSignal','/v1/getBotSignal','/signal/getBotSignal']:
    url=base.rstrip('/')+path
    try:
        req=urllib.request.Request(url, data=json.dumps(body).encode(),
            headers={'Content-Type':'application/json','Authorization':'Bearer '+key,'X-API-Key':key}, method='POST')
        with urllib.request.urlopen(req, timeout=45) as r:
            data=json.loads(r.read().decode())
        print('OK', path, json.dumps(data, ensure_ascii=False)[:1200])
        break
    except Exception as e:
        print('FAIL', path, e)

# scheduler status for this bot
print('\n=== scheduler status ===')
try:
    with urllib.request.urlopen('http://127.0.0.1:6200/scheduler/status', timeout=15) as r:
        st=json.loads(r.read().decode())
    # find spcx
    text=json.dumps(st, ensure_ascii=False)
    if '6a449bc1' in text or 'SPCX' in text:
        print('found bot in status')
    # print dca pool summary
    print(json.dumps({k:st.get(k) for k in st if k in ('pools','dca','region','job_counts','counts')} or list(st.keys())[:20], ensure_ascii=False)[:1500])
    # dump jobs mentioning bot
    def walk(o, path=''):
        if isinstance(o, dict):
            if any(str(o.get(k,'')).find('6a449bc1')>=0 for k in o):
                print('JOB', path, json.dumps(o, ensure_ascii=False)[:400])
            for k,v in o.items():
                walk(v, path+'.'+k)
        elif isinstance(o, list):
            for i,v in enumerate(o[:500]):
                walk(v, path+'[%d]'%i)
    walk(st)
except Exception as e:
    print('sched err', e)
"""

def main():
    cfg=json.loads(Path(r'C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json').read_text(encoding='utf-8'))
    s=cfg['targets']['geegoo-bot']['ssh']
    c=paramiko.SSHClient(); c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    c.connect(s['host'], username=s['user'], password=s.get('password'), timeout=30)
    p='/tmp/probe_spacex_hourly4.py'
    with c.open_sftp().file(p,'w') as f: f.write(REMOTE)
    _,o,e=c.exec_command('python3 '+p, timeout=240)
    print((o.read()+e.read()).decode('utf-8','replace')); c.close()
if __name__=='__main__': main()
