#!/usr/bin/env python3
import glob, json, os
p = sorted(glob.glob(os.path.expanduser("~/.geegoo/data/checkpoints/**/latest.json"), recursive=True), key=os.path.getmtime)[-1]
d = json.load(open(p, encoding="utf-8"))
print("keys", d.keys())
w = d.get("working") or {}
print("working keys sample", list(w.keys())[:20])
print("report_date", w.get("report_date"), "market", w.get("market"))
