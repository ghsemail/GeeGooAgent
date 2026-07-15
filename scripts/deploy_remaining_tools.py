#!/usr/bin/env python3
"""Deploy remaining tool fixes: agent, bot signal env, signal stack."""
from __future__ import annotations

import json
import subprocess
import sys
from pathlib import Path

import paramiko

DEPLOY = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")
AGENT_DEPLOY = Path(__file__).resolve().parent / "deploy_agent_server.py"
DEPLOY_PY = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\scripts\deploy.py")


def ssh_run(target: dict, cmd: str, timeout: int = 300) -> None:
    ssh = target["ssh"]
    remote = target["remote_dir"]
    c = paramiko.SSHClient()
    c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    c.connect(ssh["host"], username=ssh["user"], password=ssh.get("password"), timeout=30)
    _, out, err = c.exec_command(f"cd {remote} && {cmd}", timeout=timeout)
    o = out.read().decode("utf-8", "replace")
    e = err.read().decode("utf-8", "replace")
    code = out.channel.recv_exit_status()
    c.close()
    print(o.rstrip())
    if e.strip():
        print("STDERR:", e.rstrip())
    if code != 0:
        raise RuntimeError(f"remote exit {code}")


def main() -> int:
    cfg = json.loads(DEPLOY.read_text(encoding="utf-8"))

    print("=== GeeGooAgent deploy ===")
    subprocess.check_call([sys.executable, str(AGENT_DEPLOY)])

    print("\n=== GeeGooBot signal env + mcp-api ===")
    bot = cfg["targets"]["geegoo-bot"]
    ssh_run(
        bot,
        "touch .env && "
        "grep -q '^GEEGOO_SIGNAL_SIGNAL_API_URL=' .env || "
        "echo 'GEEGOO_SIGNAL_SIGNAL_API_URL=http://146.56.225.252:3200' >> .env && "
        "grep -q '^GEEGOO_SIGNAL_CATALOG_API_URL=' .env || "
        "echo 'GEEGOO_SIGNAL_CATALOG_API_URL=http://146.56.225.252:3210' >> .env && "
        "grep -q '^GEEGOO_SIGNAL_ANALYZE_API_URL=' .env || "
        "echo 'GEEGOO_SIGNAL_ANALYZE_API_URL=http://146.56.225.252:3230' >> .env && "
        "grep -E 'GEEGOO_SIGNAL' .env | head -6 && "
        "echo 4 | bash start.sh && tail -n 3 mcp-api.out",
        timeout=300,
    )

    print("\n=== GeeGooSignal sync + restart ===")
    sig = cfg["targets"]["geegoo-signal"]
    ssh_run(sig, "git fetch origin main && git reset --hard origin/main && git log -1 --oneline", timeout=120)
    for svc in ("signal-api", "analyze-api"):
        print(f"\n--- restart {svc} ---")
        subprocess.check_call([sys.executable, str(DEPLOY_PY), "-t", "geegoo-signal", "-s", svc])

    return 0


if __name__ == "__main__":
    raise SystemExit(main())
