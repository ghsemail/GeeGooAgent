#!/usr/bin/env python3
from __future__ import annotations
import json
from pathlib import Path
import paramiko

DEPLOY = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")
DCA_BOT = "6a449bc14b77fe41d732b809"

REMOTE = r"""
import json, os
from datetime import datetime, timedelta, timezone
from bson import ObjectId
from pymongo import MongoClient

DCA_BOT = """ + json.dumps(DCA_BOT) + r"""
oid = ObjectId(DCA_BOT)

os.chdir('/home/ubuntu/apps/GeeGooBot')
for line in open('.env'):
    line=line.strip()
    if line and not line.startswith('#') and '=' in line:
        k,v=line.split('=',1); os.environ.setdefault(k.strip(), v.strip().strip('"').strip("'"))
db=MongoClient(os.environ['GEEGOO_BOT_MONGO_URI'])[os.environ['GEEGOO_BOT_MONGO_DB']]

# sample bot_id types in dca_log
print('sample bot_ids:')
for row in db['dca_log'].find().sort('time',-1).limit(5):
    bid=row.get('bot_id')
    print(type(bid).__name__, bid, 'type=', row.get('type'), 'time=', row.get('time'))

print('\ncounts by bot_id match:')
for qname,q in [
    ('str', {'bot_id': DCA_BOT}),
    ('oid', {'bot_id': oid}),
    ('code in log', {}),
]:
    if qname=='code in log':
        n=db['dca_log'].count_documents({'log.code':'SPCX.US'})
        print(qname, n)
    else:
        print(qname, db['dca_log'].count_documents(q))

# find any SPCX logs
print('\n=== search SPCX in recent logs ===')
since=datetime.utcnow()-timedelta(days=3)
rows=list(db['dca_log'].find({'time':{'$gte':since}}).sort('time',-1).limit(2000))
hit=[]
for row in rows:
    lg=row.get('log') or {}
    bid=str(row.get('bot_id'))
    # match bot or code fields
    if bid==DCA_BOT or lg.get('code')=='SPCX.US' or 'SPCX' in str(lg)[:200]:
        hit.append(row)
print('scanned', len(rows), 'hits', len(hit))

# also find bot by name
bots=list(db['dca_bot'].find({'$or':[
    {'code':'SPCX.US'},
    {'stock_name':{'$regex':'SpaceX','$options':'i'}},
    {'botname':{'$regex':'SpaceX','$options':'i'}},
]}))
print('\nSPCX bots', len(bots))
for b in bots:
    print(str(b['_id']), b.get('botname'), b.get('code'), 'freq', b.get('frequency'))
    bid=b['_id']
    n1=db['dca_log'].count_documents({'bot_id': bid})
    n2=db['dca_log'].count_documents({'bot_id': str(bid)})
    print('  logs oid', n1, 'str', n2)
    latest=list(db['dca_log'].find({'bot_id': bid}).sort('time',-1).limit(30))
    if not latest:
        latest=list(db['dca_log'].find({'bot_id': str(bid)}).sort('time',-1).limit(30))
    for row in latest:
        lg=row.get('log') or {}
        pos=lg.get('position') or {}
        buy=lg.get('buy_signal')
        sell=lg.get('sell_signal')
        # compact buy/sell
        def side_sum(side):
            if not isinstance(side, dict):
                return str(side)[:80]
            out=[]
            for k,v in side.items():
                if k in ('name','index'): continue
                if isinstance(v, dict) and 'signal' in v:
                    out.append('%s:%s'%(k,v.get('signal')))
                elif k=='signal':
                    out.append('signal:%s'%v)
            return ','.join(out) if out else str(side)[:100]
        print(row.get('time'), 'next=', lg.get('next_opt'), 'sig_date=', lg.get('signal_date'),
              'buy=', side_sum(buy), 'sell=', side_sum(sell),
              'opt=', pos.get('opt'), 'agent=', (lg.get('trade_agent') or {}).get('result'))

# reminder?
rems=list(db['dca_reminder'].find({'$or':[{'code':'SPCX.US'},{'stock_name':{'$regex':'SpaceX','$options':'i'}}]}))
print('\nDCA reminders', len(rems))
for r in rems:
    print(str(r['_id']), r.get('botname') or r.get('reminder_name'), r.get('code'), r.get('frequency'), r.get('switch'))
    rid=r['_id']
    logs=list(db['dca_reminder_log'].find({'$or':[{'bot_id':rid},{'bot_id':str(rid)}]}).sort('time',-1).limit(25))
    print('  rem logs', len(logs))
    for row in logs:
        lg=row.get('log') or {}
        print(' ', row.get('time'), 'next=', lg.get('next_opt'), 'sig_date=', lg.get('signal_date'),
              'agent=', (lg.get('trade_agent') or {}).get('result'))
"""

def main():
    cfg=json.loads(DEPLOY.read_text(encoding='utf-8'))
    s=cfg['targets']['geegoo-bot']['ssh']
    c=paramiko.SSHClient(); c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    c.connect(s['host'], username=s['user'], password=s.get('password'), timeout=30)
    p='/tmp/probe_spacex_hourly2.py'
    with c.open_sftp().file(p,'w') as f: f.write(REMOTE)
    _,o,e=c.exec_command('python3 '+p, timeout=240)
    print((o.read()+e.read()).decode('utf-8','replace')); c.close()
if __name__=='__main__': main()
