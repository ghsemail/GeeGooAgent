#!/usr/bin/env python3
"""Live smoke: signal_diagnose workflow (SAR + 腾讯) via Go runner."""
from __future__ import annotations

import json
import os
import subprocess
import sys
from pathlib import Path

import paramiko

REPO = Path(__file__).resolve().parents[2]
DEPLOY = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")


def load_agent_mcp_token() -> str:
    cfg = json.loads(DEPLOY.read_text(encoding="utf-8-sig"))
    s = cfg["targets"]["geegoo-agent"]["ssh"]
    c = paramiko.SSHClient()
    c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    c.connect(s["host"], username=s["user"], password=s.get("password"), timeout=30)
    _, o, _ = c.exec_command(
        "python3 -c \"import json; print(json.load(open('/home/ubuntu/.geegoo/config.json')).get('mcp_token',''))\"",
        timeout=30,
    )
    token = o.read().decode("utf-8", errors="replace").strip()
    c.close()
    return token


def load_signal_key() -> tuple[str, str]:
    cfg = json.loads(DEPLOY.read_text(encoding="utf-8-sig"))
    s = cfg["targets"]["geegoo-signal"]["ssh"]
    c = paramiko.SSHClient()
    c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    c.connect(s["host"], username=s["user"], password=s.get("password"), timeout=30)
    _, o, _ = c.exec_command(
        "grep -E '^GEEGOO_SIGNAL_SIGNAL_API_KEY=' /root/apps/GeeGooSignal/.env | head -1",
        timeout=30,
    )
    line = o.read().decode("utf-8", errors="replace").strip()
    c.close()
    if "=" not in line:
        raise RuntimeError("missing GEEGOO_SIGNAL_SIGNAL_API_KEY in GeeGooSignal .env")
    key = line.split("=", 1)[1].strip().strip('"').strip("'")
    cat_key = key
    return key, cat_key


def main() -> int:
    key, cat_key = load_signal_key()
    mcp_token = load_agent_mcp_token()
    env = os.environ.copy()
    env["MCP_TOKEN"] = mcp_token
    env["SIGNAL_API_URL"] = "http://146.56.225.252:3200"
    env["SIGNAL_API_KEY"] = key
    env["SIGNAL_CATALOG_URL"] = "http://146.56.225.252:3210"
    env["SIGNAL_CATALOG_KEY"] = cat_key
    cmd = ["go", "run", "./scripts/eval/signal_diagnose_smoke", "诊断 SAR · 腾讯"]
    proc = subprocess.run(
        cmd, cwd=REPO, env=env, capture_output=True, text=True, timeout=300, encoding="utf-8", errors="replace"
    )
    if proc.stdout:
        print(proc.stdout, end="" if proc.stdout.endswith("\n") else "\n")
    if proc.stderr:
        print(proc.stderr, file=sys.stderr, end="" if proc.stderr.endswith("\n") else "\n")
    return proc.returncode


if __name__ == "__main__":
    raise SystemExit(main())
