#!/usr/bin/env python3
import glob, json, os
base = os.path.expanduser("~/.geegoo/data")
for p in sorted(glob.glob(base + "/checkpoints/**/*.json", recursive=True), key=os.path.getmtime)[-3:]:
    print("CP", p)
    try:
        d = json.load(open(p, encoding="utf-8"))
        w = d.get("working") or d
        print("  report_date", w.get("report_date"), "market", w.get("market"))
    except Exception as e:
        print("  err", e)
