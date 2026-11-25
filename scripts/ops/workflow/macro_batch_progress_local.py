#!/usr/bin/env python3
"""本机查 macro-api 当日 cn_a / hk 批跑进度。"""
from __future__ import annotations

import json
import re
import sys
import urllib.error
import urllib.request
from pathlib import Path

import paramiko

SKILL = Path.home() / ".cursor" / "skills" / "remote-deploy" / "scripts"
sys.path.insert(0, str(SKILL))
from credentials import resolve_ssh_cfg  # noqa: E402

DEPLOY = Path.home() / ".cursor" / "skills" / "remote-deploy" / "deploy.json"
MACRO = "http://146.56.225.252:3220"


def _macro_key() -> str:
    cfg = json.loads(DEPLOY.read_text(encoding="utf-8"))
    ssh_cfg = resolve_ssh_cfg(cfg["targets"]["geegoo-agent"]["ssh"], "geegoo-agent")
    client = paramiko.SSHClient()
    client.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    client.connect(
        hostname=ssh_cfg["host"],
        port=int(ssh_cfg.get("port", 22)),
        username=ssh_cfg["user"],
        password=ssh_cfg.get("password"),
        key_filename=ssh_cfg.get("key_file"),
        timeout=30,
    )
    _, stdout, _ = client.exec_command(
        'grep -E "^MACRO_API_KEY=" /home/ubuntu/.geegoo/agent.env | head -1',
        timeout=20,
    )
    line = stdout.read().decode("utf-8", errors="replace").strip()
    client.close()
    m = re.match(r"MACRO_API_KEY=(.*)", line)
    if not m:
        return ""
    return m.group(1).strip().strip('"')


def main() -> int:
    date = sys.argv[1] if len(sys.argv) > 1 else "2026-11-23"
    key = _macro_key()
    if not key:
        print("MACRO_API_KEY missing", file=sys.stderr)
        return 2
    for wf in ("cn_a", "hk"):
        q = (
            f"workflow={wf}&date_from={date}&date_to={date}&limit=2"
        )
        url = f"{MACRO.rstrip('/')}/macro/batch/runs?{q}"
        req = urllib.request.Request(url, headers={"Authorization": f"Bearer {key}"})
        try:
            with urllib.request.urlopen(req, timeout=20) as resp:
                data = json.loads(resp.read().decode("utf-8", errors="replace"))
        except urllib.error.HTTPError as e:
            print(wf, "HTTP", e.code, e.read()[:200])
            continue
        runs = data.get("runs") or data.get("items") or []
        if not runs:
            print(wf, date, "no batch runs yet")
            continue
        run = runs[0]
        print(
            wf,
            date,
            "summary:",
            run.get("summary"),
            "| success:",
            run.get("success"),
            "| pool_jev:",
            (run.get("pool_jev") or {}).get("jev_direction"),
        )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())