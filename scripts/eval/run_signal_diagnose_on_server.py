#!/usr/bin/env python3
"""Run signal_diagnose smoke on geegoo-agent server (WeKnora localhost)."""
from __future__ import annotations

import json
import sys
from pathlib import Path

import paramiko

DEPLOY = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")
OUT = Path(__file__).resolve().parents[2] / "_signal_diagnose_report.md"


def main() -> int:
    cfg = json.loads(DEPLOY.read_text(encoding="utf-8-sig"))
    s = cfg["targets"]["geegoo-agent"]["ssh"]
    sig = cfg["targets"]["geegoo-signal"]["ssh"]
    c = paramiko.SSHClient()
    c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    c.connect(s["host"], username=s["user"], password=s.get("password"), timeout=30)
    cs = paramiko.SSHClient()
    cs.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    cs.connect(sig["host"], username=sig["user"], password=sig.get("password"), timeout=30)
    _, o, _ = cs.exec_command(
        "grep -E '^GEEGOO_SIGNAL_SIGNAL_API_KEY=' /root/apps/GeeGooSignal/.env | head -1",
        timeout=30,
    )
    line = o.read().decode("utf-8", errors="replace").strip()
    cs.close()
    key = line.split("=", 1)[1].strip().strip('"').strip("'") if "=" in line else ""
    remote_cmd = f"""
set -e
cd /home/ubuntu/.geegoo/geegoo-agent
export GEEGOO_CONFIG=/home/ubuntu/.geegoo/config.json
export SIGNAL_API_URL=http://146.56.225.252:3200
export SIGNAL_CATALOG_URL=http://146.56.225.252:3210
export SIGNAL_API_KEY={key}
export SIGNAL_CATALOG_KEY={key}
export MCP_TOKEN=$(python3 -c "import json; print(json.load(open('/home/ubuntu/.geegoo/config.json')).get('mcp_token',''))")
export SIGNAL_DIAGNOSE_SKIP_KB=1
go run ./scripts/eval/signal_diagnose_smoke '诊断 SAR · 腾讯'
"""
    _, o, e = c.exec_command(remote_cmd, timeout=300)
    out = o.read().decode("utf-8", errors="replace")
    err = e.read().decode("utf-8", errors="replace")
    c.close()
    OUT.write_text(out, encoding="utf-8")
    if out:
        print(out, end="" if out.endswith("\n") else "\n")
    if err:
        print(err, file=sys.stderr, end="" if err.endswith("\n") else "\n")
    return 0 if out and "知识库写入失败" not in out and "Workflow 已暂停" not in out else (1 if err or not out else 0)


if __name__ == "__main__":
    raise SystemExit(main())
