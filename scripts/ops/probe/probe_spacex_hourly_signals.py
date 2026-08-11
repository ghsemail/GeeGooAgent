#!/usr/bin/env python3
"""Why SpaceX DCA bot shows a signal every hour."""
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

os.chdir('/home/ubuntu/apps/GeeGooBot')
for line in open('.env'):
    line = line.strip()
    if line and not line.startswith('#') and '=' in line:
        k, v = line.split('=', 1)
        os.environ.setdefault(k.strip(), v.strip().strip('"').strip("'"))
db = MongoClient(os.environ['GEEGOO_BOT_MONGO_URI'])[os.environ['GEEGOO_BOT_MONGO_DB']]

oid = ObjectId(DCA_BOT)
bot = db['dca_bot'].find_one({'_id': oid})
info = db['dca_info'].find_one({'bot_id': DCA_BOT}) or db['dca_info'].find_one({'bot_id': oid})

print('=== BOT ===')
print(json.dumps({
    'botname': bot.get('botname') if bot else None,
    'code': bot.get('code') if bot else None,
    'stock_name': bot.get('stock_name') if bot else None,
    'frequency': bot.get('frequency') if bot else None,
    'switch': bot.get('switch') if bot else None,
    'signal': bot.get('signal') if bot else None,
    'attitude': bot.get('attitude') if bot else None,
    'price': bot.get('price') if bot else None,
    'order_size': bot.get('order_size') if bot else None,
}, ensure_ascii=False, default=str))

print('\n=== INFO ===')
if info:
    pos = info.get('position') or {}
    print(json.dumps({
        'status': info.get('status'),
        'switch': info.get('switch'),
        'position_opt': pos.get('opt'),
        'order_status': pos.get('order_status'),
        'qty': pos.get('qty'),
        'can_sell_qty': pos.get('can_sell_qty'),
        'avg_cost': pos.get('avg_cost'),
    }, ensure_ascii=False, default=str))

# bot_id may be stored as string or ObjectId
since = datetime.now(timezone.utc) - timedelta(hours=36)
q_or = [{'$or': [{'bot_id': DCA_BOT}, {'bot_id': oid}], 'type': 'DCA', 'time': {'$gte': since}}]
# simplify: try both
rows = list(db['dca_log'].find({
    '$or': [{'bot_id': DCA_BOT}, {'bot_id': oid}],
    'type': 'DCA',
    'time': {'$gte': since},
}).sort('time', -1).limit(40))

print('\n=== LAST DCA LOGS (newest first, up to 40 / 36h) ===')
print('count', len(rows))
for row in rows:
    lg = row.get('log') or {}
    pos = lg.get('position') or {}
    buy = lg.get('buy_signal') or {}
    sell = lg.get('sell_signal') or {}
    ta = lg.get('trade_agent') or {}
    # summarize signal names that fired
    buy_names = []
    if isinstance(buy, dict):
        for k, v in buy.items():
            if isinstance(v, dict) and v.get('signal') in (1, '1', True):
                buy_names.append('%s=%s' % (k, v.get('signal')))
            elif k == 'signal' and v in (1, '1', -1, '-1'):
                buy_names.append('signal=%s' % v)
    sell_names = []
    if isinstance(sell, dict):
        for k, v in sell.items():
            if isinstance(v, dict) and v.get('signal') in (-1, '-1', 1, '1'):
                sell_names.append('%s=%s' % (k, v.get('signal')))
            elif k == 'signal' and v in (1, '1', -1, '-1'):
                sell_names.append('signal=%s' % v)
    print(json.dumps({
        'time': str(row.get('time')),
        'signal_date': lg.get('signal_date'),
        'next_opt': lg.get('next_opt'),
        'buy_hit': buy_names or str(buy)[:120],
        'sell_hit': sell_names or str(sell)[:120],
        'opt': pos.get('opt'),
        'order_status': pos.get('order_status'),
        'agent': {k: ta.get(k) for k in ('result', 'confidence')} if ta else None,
    }, ensure_ascii=False))

# next_opt distribution
from collections import Counter
c = Counter(str((r.get('log') or {}).get('next_opt')) for r in rows)
sd = Counter(str((r.get('log') or {}).get('signal_date')) for r in rows)
print('\n=== next_opt counts ===', dict(c))
print('=== signal_date counts ===', dict(sd))

# scheduler jobs for this bot
print('\n=== scheduler mentions (if any in recent worker/scheduler out — skip) ===')
# Check how DCA is scheduled
print('frequency field drives 60m cron ticks; each tick ALWAYS inserts a dca_log row.')
"""


def main() -> int:
    cfg = json.loads(DEPLOY.read_text(encoding="utf-8"))
    s = cfg["targets"]["geegoo-bot"]["ssh"]
    c = paramiko.SSHClient()
    c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    c.connect(s["host"], username=s["user"], password=s.get("password"), timeout=30)
    remote_path = "/tmp/probe_spacex_hourly.py"
    sftp = c.open_sftp()
    with sftp.file(remote_path, "w") as f:
        f.write(REMOTE)
    sftp.close()
    _, o, e = c.exec_command(f"python3 {remote_path}", timeout=180)
    print((o.read() + e.read()).decode("utf-8", errors="replace"))
    c.close()
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
