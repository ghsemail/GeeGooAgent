#!/usr/bin/env python3
import json
from datetime import datetime
from pathlib import Path
import paramiko

USER = "6366170502d5c175fd586fe8"
BOT = "sk-7a2b3c4d5e6f7g8h9i0j1k2l3m4n5o6p7q8r9s0t1u2v3w4x5y6z7a8b9c0d1e2"
DEPLOY = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")

py = f'''
import json, urllib.request
from datetime import datetime
USER = "{USER}"
BOT = "{BOT}"
body=json.dumps({{"user_id":USER,"limit_per_phase":5}}).encode()
req=urllib.request.Request("http://127.0.0.1:3140/reports/daily",data=body,headers={{"Content-Type":"application/json","Authorization":"Bearer "+BOT}},method="POST")
data=json.loads(urllib.request.urlopen(req,timeout=60).read())
today=datetime.now().strftime("%Y-%m-%d")
print("today", today, "api_code", data.get("code"))
for phase in ["pre_market","post_market","intraday"]:
    rows=data.get("data",{{}}).get(phase) or []
    today_n=0
    missing_date=0
    missing_body=0
    for r in rows:
        rd=str(r.get("report_date") or r.get("session_date") or "")
        if today in rd: today_n+=1
        if not rd: missing_date+=1
        rep=str(r.get("report") or "")
        summ=str(r.get("summary") or "")
        if phase=="post_market":
            body_ok=bool(summ or r.get("trade_summary") or r.get("market_summary") or rep)
        else:
            body_ok=bool(rep or summ or r.get("reason"))
        if not body_ok: missing_body+=1
    print(f"{{phase}}: n={{len(rows)}} today={{today_n}} missing_date={{missing_date}} empty_body={{missing_body}}")
    if rows:
        r=rows[0]
        keys=["report_id","report_date","session_date","code","bot_name","result","session_bias","confidence","summary","report","updated_at"]
        slim={{k:r.get(k) for k in keys}}
        for k in ["summary","report"]:
            if isinstance(slim.get(k),str) and len(slim[k])>100:
                slim[k]=slim[k][:100]+"..."
        print(" sample", json.dumps(slim, ensure_ascii=False))
'''

cfg = json.loads(DEPLOY.read_text(encoding="utf-8"))
ssh = cfg["targets"]["geegoo-bot"]["ssh"]
c = paramiko.SSHClient()
c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
c.connect(ssh["host"], username=ssh["user"], password=ssh["password"], timeout=30)
_, o, e = c.exec_command(f"python3 - <<'PY'\n{py}\nPY", timeout=90)
print((o.read() + e.read()).decode("utf-8", errors="replace"))
c.close()
