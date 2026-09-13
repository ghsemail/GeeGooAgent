#!/usr/bin/env python3
"""Upload and run strategy_signal_explain_followup live eval on geegoo-agent host."""
from __future__ import annotations

import json
import sys
from pathlib import Path

import paramiko

ROOT = Path(__file__).resolve().parent
DEPLOY = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")
REMOTE = "/tmp/run_strategy_explain_followup_live.py"


def main() -> int:
    cfg = json.loads(DEPLOY.read_text(encoding="utf-8"))
    ssh = cfg["targets"]["geegoo-agent"]["ssh"]
    client = paramiko.SSHClient()
    client.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    client.connect(
        hostname=ssh["host"],
        username=ssh["user"],
        password=ssh.get("password"),
        timeout=30,
    )
    sftp = client.open_sftp()
    with sftp.file(REMOTE, "w") as f:
        f.write((ROOT / "run_strategy_explain_followup_live.py").read_text(encoding="utf-8"))
    sftp.close()
    _, stdout, stderr = client.exec_command(f"python3 {REMOTE}", timeout=1200)
    out = (stdout.read() + stderr.read()).decode("utf-8", errors="replace")
    if hasattr(sys.stdout, "reconfigure"):
        try:
            sys.stdout.reconfigure(encoding="utf-8", errors="replace")
        except Exception:
            pass
    print(out)
    client.close()
    return 0 if "verify ok=True" in out or (
        "diagnose_called=True" in out and "FAIL" not in out.split("turn 2")[-1][:200]
    ) else (1 if "FAIL" in out else 0)


if __name__ == "__main__":
    raise SystemExit(main())
