#!/usr/bin/env python3
"""Audit all trading_operation indicator SIGNALS on 00700.HK via getBotSignal."""
from __future__ import annotations

import json
from pathlib import Path

import paramiko

DEPLOY = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")

# From trading_operation signal_mgt_controller indexList + defaults
INDICATORS = [
    {
        "index": "RSIThrehold",
        "param": {"period": "25", "threholdBuy": "30", "threholdSell": "70"},
    },
    {
        "index": "RSICross",
        "param": {"fastPeriod": "10", "slowPeriod": "100"},
    },
    {
        "index": "SAR",
        "param": {"acceleration": "0.02", "maximum": "0.2"},
    },
    {
        "index": "MACD",
        "param": {"fastPeriod": "12", "slowPeriod": "26", "signalPeriod": "9"},
    },
    {
        "index": "EMA",
        "param": {"fastPeriod": "25", "mediumPeriod": "50", "slowPeriod": "120"},
    },
    {
        "index": "ChandelierExit",
        "param": {"period": "22", "atrMultiplier": "3"},
    },
    {
        "index": "BBAND",
        "param": {"period": "20", "matype": "2"},
    },
    {
        "index": "QFL",
        "param": {
            "basePeriods": "36",
            "pumpPeriods": "8",
            "pump": "3",
            "baseCrack": "3",
        },
    },
    {"index": "VWAP", "param": {}},
    {"index": "HeikinAshi", "param": {}},
    {
        "index": "KDJ",
        "param": {
            "period": "9",
            "p1": "3",
            "p2": "3",
            "threholdBuy": "30",
            "threholdSell": "70",
        },
    },
]

REMOTE = r"""
import json, os, urllib.request, urllib.error

os.chdir('/root/apps/GeeGooSignal')
for ef in ['.env']:
    if not os.path.isfile(ef):
        continue
    for line in open(ef):
        line=line.strip()
        if line and not line.startswith('#') and '=' in line:
            k,v=line.split('=',1)
            os.environ.setdefault(k.strip(), v.strip().strip('"').strip("'"))

BASE = 'http://127.0.0.1:3200'
KEY = os.environ.get('GEEGOO_SIGNAL_SIGNAL_API_KEY','')
INDICATORS = """ + json.dumps(INDICATORS, ensure_ascii=False) + r"""
CODE = '00700.HK'
FREQ = '60m'

def post(path, body):
    req = urllib.request.Request(
        BASE + path,
        data=json.dumps(body).encode(),
        headers={
            'Content-Type': 'application/json',
            'Authorization': 'Bearer ' + KEY,
            'X-API-Key': KEY,
        },
        method='POST',
    )
    try:
        with urllib.request.urlopen(req, timeout=90) as r:
            return r.status, json.loads(r.read().decode())
    except urllib.error.HTTPError as e:
        raw = e.read().decode(errors='replace')
        try:
            return e.code, json.loads(raw)
        except Exception:
            return e.code, {'raw': raw[:500]}
    except Exception as e:
        return 0, {'error': str(e)}

def one(index, typ, param):
    body = {
        'code': CODE,
        'frequency': FREQ,
        'buy_signal': [{'index': index, 'type': typ, 'param': param}],
        'sell_signal': [{'index': 'nosignal', 'type': '', 'param': {}}],
    }
    status, data = post('/getBotSignal', body)
    buy = (data or {}).get('buy_signal') or {}
    return {
        'http': status,
        'signal': buy.get('signal'),
        'next_opt': buy.get('next_opt'),
        'reason': (buy.get('reason') or '')[:180],
        'trade_date': (data or {}).get('trade_date'),
        'err': (data or {}).get('error') or (data or {}).get('raw') or (data or {}).get('message'),
    }

rows = []
print('=== AUDIT getBotSignal SIGNAL type on', CODE, FREQ, '===')
for item in INDICATORS:
    idx = item['index']
    param = item['param']
    sig = one(idx, 'signal', param)
    flag = one(idx, 'flag', param)
    # classification
    ok_api = sig.get('http') == 200 and flag.get('http') == 200
    market_fail = 'market fetch failed' in str(sig.get('reason','')) or 'market fetch failed' in str(flag.get('reason',''))
    unsupported = 'unsupported' in str(sig.get('reason','')).lower() or 'unsupported' in str(flag.get('reason','')).lower()
    # signal vs flag should often differ on continuous markets; not always required every bar
    differs = sig.get('signal') != flag.get('signal')
    # For type=signal alone, reason should mention signal= not look like pure continuous close-above
    reason = str(sig.get('reason',''))
    looks_legacy_state = any(x in reason.lower() for x in ['close ', 'above', 'below']) and 'signal=' not in reason and 'flag=' not in reason
    status = 'PASS'
    notes = []
    if not ok_api:
        status = 'FAIL_HTTP'
        notes.append('http_error')
    elif market_fail:
        status = 'FAIL_MARKET'
        notes.append('market_fetch')
    elif unsupported:
        status = 'FAIL_UNSUPPORTED'
        notes.append('engine_unsupported')
    elif looks_legacy_state:
        status = 'WARN_LEGACY_REASON'
        notes.append('reason_looks_old_state_logic')
    elif sig.get('next_opt') not in ('buy','sell','hold'):
        status = 'FAIL_SHAPE'
        notes.append('bad_next_opt')
    else:
        notes.append('signal_ok')
        if differs:
            notes.append('differs_from_flag')
        else:
            notes.append('same_as_flag_this_bar')

    row = {
        'index': idx,
        'status': status,
        'signal': {'v': sig.get('signal'), 'opt': sig.get('next_opt'), 'reason': sig.get('reason')},
        'flag': {'v': flag.get('signal'), 'opt': flag.get('next_opt'), 'reason': flag.get('reason')},
        'differs': differs,
        'notes': notes,
        'trade_date': sig.get('trade_date'),
    }
    rows.append(row)
    print(json.dumps(row, ensure_ascii=False))

# summary
from collections import Counter
c = Counter(r['status'] for r in rows)
print('\n=== SUMMARY ===')
print(json.dumps({
    'code': CODE,
    'frequency': FREQ,
    'counts': dict(c),
    'pass': [r['index'] for r in rows if r['status']=='PASS'],
    'fail': [r['index'] for r in rows if r['status'].startswith('FAIL')],
    'warn': [r['index'] for r in rows if r['status'].startswith('WARN')],
}, ensure_ascii=False, indent=2))
"""


def main() -> int:
    cfg = json.loads(DEPLOY.read_text(encoding="utf-8"))
    s = cfg["targets"]["geegoo-signal"]["ssh"]
    c = paramiko.SSHClient()
    c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    c.connect(s["host"], username=s["user"], password=s.get("password"), timeout=30)
    p = "/tmp/audit_signal_00700.py"
    with c.open_sftp().file(p, "w") as f:
        f.write(REMOTE)
    _, o, e = c.exec_command(f"python3 {p}", timeout=300)
    print((o.read() + e.read()).decode("utf-8", errors="replace"))
    c.close()
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
