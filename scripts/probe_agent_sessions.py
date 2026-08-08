#!/usr/bin/env python3
import glob, json, os

base = os.path.expanduser("~/.geegoo/data")
sessions = sorted(glob.glob(base + "/sessions/*.json"), key=os.path.getmtime)[-3:]
for p in sessions:
    print("SESSION", os.path.basename(p))
    try:
        data = json.load(open(p, encoding="utf-8"))
    except Exception as e:
        print("  read err", e)
        continue
    print("  skill", data.get("skill"), "report_date", data.get("report_date"), "market", data.get("market"))
    print("  market_report_id", data.get("market_report_id"), "phase", data.get("phase"))
