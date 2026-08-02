#!/usr/bin/env python3
"""Delete English-only rows from GeeGooData aidb.news_cache (HK + CN nodes).

DEPRECATED: Do not use for language UX. News should stay bilingual in Mongo;
clients pick cn/en/hk by app language (TradingData parity). Use only for manual
data cleanup experiments.
"""
from __future__ import annotations

import argparse
import json
import re
import sys
from pathlib import Path

import paramiko

DEPLOY = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")

REMOTE_PY = r'''
import json
import re
from pymongo import MongoClient

def is_chinese_text(text: str) -> bool:
    text = (text or "").strip()
    if not text:
        return False
    cjk = len(re.findall(r"[\u4e00-\u9fff]", text))
    latin = len(re.findall(r"[A-Za-z]", text))
    return cjk > 0 and cjk >= latin

def title_maps(news: dict):
    title = (news or {}).get("title")
    if isinstance(title, dict):
        return title
    if isinstance(title, str):
        return {"init": title}
    return {}

def has_chinese_title(news: dict) -> bool:
    m = title_maps(news)
    for key in ("cn", "hk", "zh_hk", "init"):
        val = (m.get(key) or "").strip()
        if val and is_chinese_text(val):
            return True
    return False

def english_only_title(news: dict) -> bool:
    if has_chinese_title(news):
        return False
    m = title_maps(news)
    text = (m.get("init") or m.get("en") or "").strip()
    if not text:
        return False
    return not is_chinese_text(text)

dry_run = __DRY_RUN__
client = MongoClient(__MONGO_URI__)
coll = client[__DB_NAME__]["news_cache"]

total = coll.count_documents({})
english_ids = []
samples = []
for doc in coll.find({}, {"news": 1, "code": 1, "ts": 1}):
    news = doc.get("news") or {}
    if not english_only_title(news):
        continue
    english_ids.append(doc["_id"])
    if len(samples) < 5:
        title = title_maps(news)
        samples.append({
            "code": doc.get("code"),
            "ts": doc.get("ts"),
            "title": (title.get("init") or title.get("en") or "")[:120],
        })

deleted = 0
if not dry_run and english_ids:
    res = coll.delete_many({"_id": {"$in": english_ids}})
    deleted = res.deleted_count

remaining = coll.count_documents({})
print(json.dumps({
    "total_before": total,
    "english_only": len(english_ids),
    "deleted": deleted if not dry_run else 0,
    "remaining": remaining,
    "dry_run": dry_run,
    "samples": samples,
}, ensure_ascii=False))
'''

TARGETS = [
    ("geegoo-data", "HK/US"),
    ("geegoo-data-cn", "CN"),
]


def load_mongo_env(target: str, cfg: dict) -> tuple[str, str]:
    ssh = cfg["targets"][target]["ssh"]
    remote_dir = cfg["targets"][target]["remote_dir"]
    client = paramiko.SSHClient()
    client.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    client.connect(
        ssh["host"],
        username=ssh["user"],
        password=ssh["password"],
        timeout=60,
    )
    cmd = (
        f"grep -E '^GEEGOO_DATA_MONGO_URI=|^GEEGOO_DATA_AIDB=' {remote_dir}/.env "
        r"| sed 's/^GEEGOO_DATA_MONGO_URI=/URI=/' | sed 's/^GEEGOO_DATA_AIDB=/DB=/'"
    )
    _, stdout, stderr = client.exec_command(cmd, timeout=30)
    out = (stdout.read() + stderr.read()).decode("utf-8", errors="replace")
    client.close()
    mongo_uri = "mongodb://127.0.0.1:27017"
    db_name = "aidb"
    for line in out.splitlines():
        line = line.strip()
        if line.startswith("URI="):
            mongo_uri = line.split("=", 1)[1]
        elif line.startswith("DB="):
            db_name = line.split("=", 1)[1]
    return mongo_uri, db_name


def run_node(target: str, label: str, cfg: dict, dry_run: bool) -> dict:
    ssh = cfg["targets"][target]["ssh"]
    mongo_uri, db_name = load_mongo_env(target, cfg)
    py = (
        REMOTE_PY.replace("__DRY_RUN__", "True" if dry_run else "False")
        .replace("__MONGO_URI__", repr(mongo_uri))
        .replace("__DB_NAME__", repr(db_name))
    )
    client = paramiko.SSHClient()
    client.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    client.connect(
        ssh["host"],
        username=ssh["user"],
        password=ssh["password"],
        timeout=60,
    )
    prep = (
        "python3 -m pip install -q pymongo 2>/dev/null || "
        "python3 -m pip install --break-system-packages -q pymongo 2>/dev/null || "
        "pip3 install -q pymongo 2>/dev/null || true"
    )
    _, prep_out, prep_err = client.exec_command(prep, timeout=180)
    prep_out.channel.recv_exit_status()
    _, stdout, stderr = client.exec_command(f"python3 - <<'PY'\n{py}\nPY", timeout=600)
    out = (stdout.read() + stderr.read()).decode("utf-8", errors="replace").strip()
    client.close()
    if not out:
        raise RuntimeError(f"{label}: empty output")
    # last line should be JSON
    line = out.splitlines()[-1]
    try:
        data = json.loads(line)
    except json.JSONDecodeError as exc:
        raise RuntimeError(f"{label}: invalid JSON\n{out}") from exc
    data["node"] = label
    data["host"] = ssh["host"]
    return data


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--apply", action="store_true", help="Actually delete (default: dry-run)")
    args = parser.parse_args()
    dry_run = not args.apply
    cfg = json.loads(DEPLOY.read_text(encoding="utf-8"))

    print("mode:", "DRY_RUN" if dry_run else "DELETE")
    results = []
    for target, label in TARGETS:
        print(f"\n=== {label} ({cfg['targets'][target]['ssh']['host']}) ===")
        try:
            data = run_node(target, label, cfg, dry_run=dry_run)
            results.append(data)
            print(json.dumps(data, ensure_ascii=False, indent=2))
        except Exception as exc:
            print(f"ERROR: {exc}", file=sys.stderr)
            return 1

    total_en = sum(r.get("english_only", 0) for r in results)
    total_del = sum(r.get("deleted", 0) for r in results)
    print(f"\nSummary: english_only={total_en}, deleted={total_del}, dry_run={dry_run}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
