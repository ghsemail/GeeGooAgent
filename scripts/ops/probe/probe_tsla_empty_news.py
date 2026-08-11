#!/usr/bin/env python3
import json
import urllib.request
from pathlib import Path

import paramiko

DEPLOY = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")


def run(target: str, cmd: str, timeout: int = 120) -> str:
    cfg = json.loads(DEPLOY.read_text(encoding="utf-8"))
    ssh = cfg["targets"][target]["ssh"]
    c = paramiko.SSHClient()
    c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    c.connect(ssh["host"], username=ssh["user"], password=ssh.get("password"), timeout=60)
    _, o, e = c.exec_command(cmd, timeout=timeout)
    return (o.read() + e.read()).decode("utf-8", "replace")


def main() -> int:
    print("=== TSLA mongo by date ===")
    print(
        run(
            "geegoo-data",
            "python3 <<'PY'\n"
            "import pymongo\n"
            "c=pymongo.MongoClient('mongodb://127.0.0.1:27017', serverSelectionTimeoutMS=5000)\n"
            "col=c.aidb.news_cache\n"
            "for ts in ['2026-08-10','2026-08-09','2026-08-08']:\n"
            "    rows=list(col.find({'code':'TSLA.US','ts':ts}))\n"
            "    if not rows: continue\n"
            "    empty=0\n"
            "    for r in rows:\n"
            "        t=(r.get('news') or {}).get('title') or {}\n"
            "        s=(r.get('news') or {}).get('summary') or {}\n"
            "        cn_t=(t.get('cn') or '').strip(); en_t=(t.get('en') or t.get('init') or '').strip()\n"
            "        cn_s=(s.get('cn') or '').strip(); en_s=(s.get('en') or s.get('init') or '').strip()\n"
            "        if not cn_t and not en_t: empty+=1; print('NO_TITLE', ts)\n"
            "        elif not cn_t: print('EMPTY_CN', ts, en_t[:60])\n"
            "        if not cn_s and not en_s and not cn_t and not en_t: print('FULL_EMPTY', ts)\n"
            "    print(ts, 'rows', len(rows), 'no_title', empty)\n"
            "PY",
        )
    )

    print("\n=== getStockNews what app receives (server local) ===")
    print(
        run(
            "geegoo-data",
            "curl -s -X POST http://127.0.0.1:3300/getStockNews -H 'Content-Type: application/json' "
            "-d '{\"stock_list\":[{\"code\":\"TSLA.US\",\"name\":{\"init\":\"特斯拉\"}}]}' "
            "| python3 -c \"import sys,json;d=json.load(sys.stdin);"
            "print('count',len(d),'cn_empty',sum(1 for n in d if not (n.get('title',{}).get('cn') or '').strip()));"
            "[print('EMPTY',i,(n.get('title',{}).get('en') or '')[:60]) for i,n in enumerate(d) if not (n.get('title',{}).get('cn') or '').strip()]\"",
        )
    )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
