#!/usr/bin/env python3
"""Check agent scheduler + postmarket_stock readiness for Monday."""
from __future__ import annotations

import json
from pathlib import Path

import paramiko

REMOTE = r'''
import json
import os
import subprocess
import urllib.request

def sh(cmd, timeout=120):
    return subprocess.check_output(cmd, shell=True, text=True, stderr=subprocess.STDOUT, timeout=timeout)

home = os.path.expanduser("~/.geegoo/geegoo-agent")
cfg = os.path.expanduser("~/.geegoo/config.json")
bin_path = os.path.join(home, "bin", "geegoo")

print("=== git ===")
try:
    print(sh(f"cd {home} && git rev-parse --short HEAD && git log -1 --oneline"))
except Exception as e:
    print("git_err", e)

print("=== scheduler process ===")
print(sh('pgrep -af "geegoo.*scheduler" || echo NONE'))

jobs_path = os.path.expanduser("~/.geegoo/scheduler/jobs.json")
print("=== jobs.json ===")
if os.path.exists(jobs_path):
    print(open(jobs_path, encoding="utf-8").read())
else:
    print("MISSING", jobs_path)

print("=== scheduler list ===")
if os.path.isfile(bin_path):
    try:
        print(sh(f"{bin_path} scheduler list --config {cfg} 2>&1"))
    except Exception as e:
        print("list_err", e)

print("=== dry-run postmarket_stock ===")
if os.path.isfile(bin_path):
    try:
        print(sh(f"cd {home} && timeout 180 {bin_path} run postmarket_stock --dry-run --config {cfg} 2>&1 | tail -50"))
    except Exception as e:
        print("dryrun_err", e)

print("=== scheduler status API ===")
try:
    with urllib.request.urlopen("http://127.0.0.1:3410/v1/scheduler/status", timeout=15) as r:
        print(r.read().decode()[:3000])
except Exception as e:
    print("status_err", e)

print("=== tail scheduler.out ===")
log = os.path.expanduser("~/.geegoo/geegoo-agent/scheduler.out")
if os.path.isfile(log):
    print(sh(f"tail -n 25 {log}"))
'''


def main() -> None:
    cfg = json.loads(
        Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json").read_text(
            encoding="utf-8"
        )
    )
    s = cfg["targets"]["geegoo-agent"]["ssh"]
    c = paramiko.SSHClient()
    c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    c.connect(s["host"], username=s["user"], password=s.get("password"), timeout=30)
    remote_path = "/tmp/probe_monday_postmarket.py"
    with c.open_sftp().file(remote_path, "w") as f:
        f.write(REMOTE)
    _, o, e = c.exec_command(f"python3 {remote_path}", timeout=240)
    print((o.read() + e.read()).decode("utf-8", "replace"))
    c.close()


if __name__ == "__main__":
    main()
