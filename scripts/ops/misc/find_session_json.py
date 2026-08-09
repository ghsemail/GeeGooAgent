#!/usr/bin/env python3
import glob, json, os
for root, dirs, files in os.walk(os.path.expanduser("~/.geegoo/data")):
    for f in files:
        if f.endswith(".json") and "run-20260808T095224" in f:
            print(os.path.join(root, f))
for p in sorted(glob.glob(os.path.expanduser("~/.geegoo/data/**/*.json"), recursive=True), key=os.path.getmtime)[-5:]:
    print("recent", p)
