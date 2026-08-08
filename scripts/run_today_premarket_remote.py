#!/usr/bin/env python3
import json, os, subprocess, glob

market = "CN"
today = "2026-08-08"
cfg_path = os.path.expanduser("~/.geegoo/config.json")
install_dir = os.path.expanduser("~/.geegoo/geegoo-agent")
geegoo = os.path.join(install_dir, "geegoo")

for skill, rd in [("premarket_market", today), ("premarket_stock", today)]:
    run = subprocess.run(
        [geegoo, "run", "--config", cfg_path, "--market", market, "--report-date", rd, skill],
        capture_output=True, text=True, timeout=900,
    )
    print(skill, "exit", run.returncode)
    if run.stderr:
        print(run.stderr[-800:])

report_dir = os.path.expanduser("~/.geegoo/data/reports/20260808")
paths = sorted([p for p in glob.glob(report_dir + "/*-premarket.md") if "market-" not in os.path.basename(p)])
print("stocks", [os.path.basename(p) for p in paths])
if paths:
    c = open(paths[-1], encoding="utf-8").read()
    print("checks", {
        "market_bg": "## 市场背景" in c,
        "synthesis": "## 综合研判" in c,
        "footer": "个股盘前" in c,
    })
    print(c[:1200])
