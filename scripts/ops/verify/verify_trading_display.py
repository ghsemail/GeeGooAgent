#!/usr/bin/env python3
"""Verify dashboard / stock detail / bot log API data on production."""
from __future__ import annotations

import json
import textwrap
from pathlib import Path

import paramiko

DEPLOY = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")


def run(ssh_cfg: dict, cmd: str, timeout: int = 120) -> str:
    client = paramiko.SSHClient()
    client.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    client.connect(
        ssh_cfg["host"],
        username=ssh_cfg["user"],
        password=ssh_cfg.get("password"),
        timeout=30,
    )
    try:
        print(f"\n{'='*60}\n>>> {cmd[:200]}{'...' if len(cmd)>200 else ''}\n")
        _, stdout, stderr = client.exec_command(cmd, timeout=timeout)
        out = (stdout.read() + stderr.read()).decode("utf-8", "replace")
        print(out[-8000:] if len(out) > 8000 else out)
        return out
    finally:
        client.close()


def main() -> None:
    cfg = json.loads(DEPLOY.read_text(encoding="utf-8"))
    bot = cfg["targets"]["geegoo-bot"]["ssh"]
    sig = cfg["targets"]["geegoo-signal"]["ssh"]

    probe = r"""bash -lc 'set -e
cd /home/ubuntu/apps/GeeGooBot
set -a; source .env; set +a

python3 <<'"'"'PY'"'"'
import json, os, urllib.request
from pymongo import MongoClient

mongo = MongoClient(os.environ["GEEGOO_BOT_MONGO_URI"])
db = mongo[os.environ["GEEGOO_BOT_MONGO_DB"]]

# sample user with stocks
doc = db.user_security.find_one({"code": {"$exists": True, "$ne": ""}})
if not doc:
    print("NO_USER_STOCK")
    raise SystemExit(1)
uid = str(doc["user_id"])
codes = [x["code"] for x in db.user_security.find({"user_id": doc["user_id"]}).limit(5)]
print("USER_ID", uid)
print("SAMPLE_CODES", codes)

# sample index ids from signal DB on signal host - use env if present
idx_ids = []
for coll_name in ("signal_index_db",):
    if coll_name in db.list_collection_names():
        for d in db[coll_name].find({"index.type": "flag"}).limit(3):
            idx_ids.append(str(d["_id"]))
print("LOCAL_INDEX_IDS", idx_ids)

app_key = os.environ.get("GEEGOO_BOT_APP_API_KEY", "")
signal_url = os.environ.get("GEEGOO_SIGNAL_SIGNAL_API_URL", "http://146.56.225.252:3200")

def post(url, body, key=None):
    data = json.dumps(body).encode()
    req = urllib.request.Request(url, data=data, method="POST")
    req.add_header("Content-Type", "application/json")
    if key:
        req.add_header("Authorization", f"Bearer {key}")
    try:
        with urllib.request.urlopen(req, timeout=60) as resp:
            raw = resp.read().decode()
            return resp.status, raw
    except Exception as e:
        if hasattr(e, "read"):
            return getattr(e, "code", 0), e.read().decode()
        return 0, str(e)

# 1) getUserStockTrend via app-api
st, raw = post(
    "http://127.0.0.1:3100/getUserStockTrend",
    {
        "user_id": uid,
        "type": "flag",
        "frequency": "5m",
        "signal_index_list": idx_ids,
        "language": "cn",
    },
    app_key,
)
print("\n=== getUserStockTrend status", st)
try:
    data = json.loads(raw)
    if isinstance(data, list):
        print("trend_rows", len(data))
        for row in data[:3]:
            print(" ", row.get("code"), "trend=", row.get("trend"))
    else:
        print("envelope", data)
except Exception:
    print(raw[:500])

# 2) direct signal getCodeListFlag
signal_key = os.environ.get("GEEGOO_SIGNAL_SIGNAL_API_KEY", "")
st2, raw2 = post(
    signal_url.rstrip("/") + "/getCodeListFlag",
    {
        "code_list": codes[:3],
        "type": "flag",
        "frequency": "5m",
        "signal_index_list": idx_ids,
        "language": "cn",
    },
    signal_key or None,
)
print("\n=== getCodeListFlag status", st2)
try:
    d2 = json.loads(raw2)
    for row in d2[:3]:
        print(" ", row)
except Exception:
    print(raw2[:500])

# 3) getDashboardSignal
if codes:
    st3, raw3 = post(
        signal_url.rstrip("/") + "/getDashboardSignal",
        {
            "code": codes[0],
            "type": "flag",
            "frequency": "daily",
            "signal_index_list": idx_ids,
            "language": "cn",
        },
        signal_key or None,
    )
    print("\n=== getDashboardSignal status", st3, "code=", codes[0])
    try:
        d3 = json.loads(raw3)
        print(" frequency=", d3.get("frequency"))
        print(" total=", d3.get("total"))
        sigs = d3.get("signal") or []
        print(" signal_count=", len(sigs))
        if sigs:
            print(" first_signal=", sigs[0])
    except Exception:
        print(raw3[:500])

# 4) GRID bot log sample
grid = db.grid_bot.find_one({"user_id": doc["user_id"]})
if grid:
    bot_id = str(grid["_id"])
    st4, raw4 = post(
        "http://127.0.0.1:3100/getGRIDBotLog",
        {"bot_id": bot_id, "hold": False},
        app_key,
    )
    print("\n=== getGRIDBotLog bot_id", bot_id, "status", st4)
    try:
        d4 = json.loads(raw4)
        if "code" in d4 and "log" not in d4:
            print(" error_envelope", d4)
        else:
            logs = d4.get("log") or []
            print(" log_count=", len(logs))
            if logs:
                print(" first_log_keys=", list(logs[0].keys()))
    except Exception:
        print(raw4[:500])
else:
    print("\n=== no GRID bot for user")

# 5) check GEEGOO_SIGNAL env
print("\n=== env")
print("GEEGOO_SIGNAL_SIGNAL_API_URL=", os.environ.get("GEEGOO_SIGNAL_SIGNAL_API_URL"))
print("GEEGOO_SIGNAL_SIGNAL_API_KEY set=", bool(os.environ.get("GEEGOO_SIGNAL_SIGNAL_API_KEY")))
PY
'"""

    run(bot, probe)

    sig_probe = r"""bash -lc 'set -e
cd /root/apps/GeeGooSignal
set -a; [ -f .env ] && source .env; set +a
echo SIGNAL_API_KEY_SET=$([ -n "$GEEGOO_SIGNAL_SIGNAL_API_KEY" ] && echo yes || echo no)
echo DATA_URL=$GEEGOO_DATA_HTTP_URL
curl -sf http://127.0.0.1:3200/health && echo signal-health-ok
python3 <<'"'"'PY'"'"'
import json, os, urllib.request
from pymongo import MongoClient

uri = os.environ.get("GEEGOO_SIGNAL_MONGO_URI", "")
db_name = os.environ.get("GEEGOO_SIGNAL_MONGO_DB", "Signal_DB")
key = os.environ.get("GEEGOO_SIGNAL_SIGNAL_API_KEY", "")
if not uri:
    print("NO_MONGO_URI")
    raise SystemExit(0)
c = MongoClient(uri)
db = c[db_name]
ids = [str(d["_id"]) for d in db.signal_index_db.find({"index.type": "flag"}).limit(5)]
print("FLAG_INDEX_COUNT", db.signal_index_db.count_documents({"index.type": "flag"}))
print("SAMPLE_INDEX_IDS", ids[:3])
body = {
    "code": "0700.HK",
    "type": "flag",
    "frequency": "daily",
    "signal_index_list": ids[:3],
    "language": "cn",
}
req = urllib.request.Request(
    "http://127.0.0.1:3200/getDashboardSignal",
    data=json.dumps(body).encode(),
    method="POST",
    headers={"Content-Type": "application/json", "Authorization": f"Bearer {key}"},
)
with urllib.request.urlopen(req, timeout=60) as resp:
    out = json.loads(resp.read().decode())
print("dashboard_total", out.get("total"))
print("dashboard_signal_len", len(out.get("signal") or []))
if out.get("signal"):
    print("first", out["signal"][0])
PY
'"""
    run(sig, sig_probe)


if __name__ == "__main__":
    main()
