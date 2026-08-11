#!/usr/bin/env python3
"""Diagnose news LLM bilingual translation failures."""
from __future__ import annotations

import json
from pathlib import Path

import paramiko

DEPLOY = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")


def run(target: str, cmd: str, timeout: int = 120) -> str:
    cfg = json.loads(DEPLOY.read_text(encoding="utf-8"))
    ssh = cfg["targets"][target]["ssh"]
    client = paramiko.SSHClient()
    client.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    client.connect(
        hostname=ssh["host"],
        port=int(ssh.get("port", 22)),
        username=ssh["user"],
        password=ssh.get("password"),
        timeout=60,
    )
    _, stdout, stderr = client.exec_command(cmd, timeout=timeout)
    out = (stdout.read() + stderr.read()).decode("utf-8", "replace")
    client.close()
    return out


def main() -> int:
    print("=== GeeGooData HK .env (news/analyze) ===")
    print(
        run(
            "geegoo-data",
            "grep -E '^(GEEGOO_SIGNAL_ANALYZE|GEEGOO_DATA_NEWS_LLM|GEEGOO_DATA_MONGO)' "
            "/root/apps/GeeGooData/.env 2>/dev/null || echo 'no .env'",
        )
    )

    key = run(
        "geegoo-signal",
        "grep '^GEEGOO_SIGNAL_ANALYZE_API_KEY=' /root/apps/GeeGooSignal/.env | cut -d= -f2-",
    ).strip()
    print(f"analyze_api_key_len={len(key)}")

    payload = json.dumps(
        {
            "title": "Tesla shares fall after downgrade",
            "snippet": "Shares dropped in after-hours trading",
        }
    )

    print("\n=== GeeGooSignal local enrichStockNews ===")
    print(
        run(
            "geegoo-signal",
            "curl -s -w '\\nHTTP:%{http_code}' -X POST http://127.0.0.1:3230/enrichStockNews "
            "-H 'Content-Type: application/json' "
            f"-H 'Authorization: Bearer {key}' "
            f"-d '{payload}'",
        )[:1200]
    )

    print("\n=== GeeGooData HK -> Signal enrichStockNews ===")
    print(
        run(
            "geegoo-data",
            "curl -s -w '\\nHTTP:%{http_code}' -X POST http://146.56.225.252:3230/enrichStockNews "
            "-H 'Content-Type: application/json' "
            f"-H 'Authorization: Bearer {key}' "
            f"-d '{payload}'",
        )[:1200]
    )

    print("\n=== TSLA getStockNews (HK data) ===")
    print(
        run(
            "geegoo-data",
            "curl -s -X POST http://127.0.0.1:3300/getStockNews -H 'Content-Type: application/json' "
            "-d '{\"stock_list\":[{\"code\":\"TSLA.US\",\"name\":{\"init\":\"特斯拉\"}}]}' "
            "| python3 -c \"import sys,json; d=json.load(sys.stdin); print('count',len(d));\\n"
            "[print(i,n.get('date'),'cn',bool((n.get('title',{}).get('cn') or '').strip()),"
            "'en',bool((n.get('title',{}).get('en') or '').strip()),"
            "'cn_title',(n.get('title',{}).get('cn') or '')[:40]) for i,n in enumerate(d[:6])]\"",
        )
    )

    print("\n=== news-worker.out tail ===")
    print(run("geegoo-data", "tail -40 /root/apps/GeeGooData/news-worker.out 2>/dev/null || echo no log"))

    print("\n=== recent refresh logs (TSLA) ===")
    print(
        run(
            "geegoo-data",
            "curl -s -X POST http://127.0.0.1:3300/getNewsRefreshLogs -H 'Content-Type: application/json' "
            "-d '{\"limit\":2}' "
            "| python3 -c \"import sys,json; logs=json.load(sys.stdin);\\n"
            "for log in logs[:2]:\\n"
            " print('run',log.get('run_date'),log.get('status'),'total_news',log.get('total_news'));\\n"
            " [print(' ',d) for d in log.get('details',[]) if 'TSLA' in d.get('code','')]\"",
        )
    )

    print("\n=== GeeGooSignal analyze-api / ai_model configured ===")
    print(
        run(
            "geegoo-signal",
            "python3 -c \"import os,pymongo; "
            "uri=os.environ.get('GEEGOO_SIGNAL_MONGO_URI',''); "
            "c=pymongo.MongoClient(uri, serverSelectionTimeoutMS=5000); "
            "db=c.get_default_database(); "
            "m=db.ai_model_db.find_one({'type':'configured'}); "
            "print('model', (m or {}).get('name'), 'provider', (m or {}).get('provider'), "
            "'base_url', (m or {}).get('base_url','')[:60], 'has_token', bool((m or {}).get('token')))\" "
            "2>/dev/null || echo mongo probe failed",
        )
    )

    title = (
        "Elon Musk's Boring Company Is Raising Money at a $20 Billion Valuation. "
        "Here's How His Empire Outside Tesla Is Growing."
    )
    payload2 = json.dumps({"title": title, "snippet": ""})
    print("\n=== enrichStockNews for empty-cn TSLA headline ===")
    print(
        run(
            "geegoo-signal",
            "curl -s -X POST http://127.0.0.1:3230/enrichStockNews "
            "-H 'Content-Type: application/json' "
            f"-H 'Authorization: Bearer {key}' "
            f"-d '{payload2}'",
        )[:1500]
    )

    print("\n=== analyze-api.out tail ===")
    print(
        run(
            "geegoo-signal",
            "tail -60 /root/apps/GeeGooSignal/analyze-api.out 2>/dev/null || echo no analyze log",
        )
    )

    print("\n=== TSLA rows with empty cn in mongo ===")
    print(
        run(
            "geegoo-data",
            "python3 <<'PY'\n"
            "import pymongo\n"
            "c=pymongo.MongoClient('mongodb://127.0.0.1:27017', serverSelectionTimeoutMS=5000)\n"
            "col=c.aidb.news_cache\n"
            "for r in col.find({'code':'TSLA.US','ts':'2026-08-10'}, {'_id':0,'news':1}):\n"
            "    t=(r.get('news') or {}).get('title') or {}\n"
            "    if not (t.get('cn') or '').strip():\n"
            "        print('EMPTY_CN', (t.get('en') or t.get('init') or '')[:100])\n"
            "PY",
        )
    )

    return 0


if __name__ == "__main__":
    raise SystemExit(main())
