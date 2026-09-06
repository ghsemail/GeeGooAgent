#!/usr/bin/env python3
"""SSH to geegoo-agent host and run stock TurnPlan live eval cases."""
from __future__ import annotations

import argparse
import json
import sys
import time
from pathlib import Path

import paramiko

if hasattr(sys.stdout, "reconfigure"):
    try:
        sys.stdout.reconfigure(encoding="utf-8", errors="replace")
    except Exception:
        pass

DEPLOY = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")
SCRIPT = Path(__file__).with_name("run_turnplan_stock_eval.py")
REMOTE_PATH = "/tmp/run_turnplan_stock_eval.py"
REMOTE_LOG = "/tmp/run_turnplan_stock_eval.log"


def connect(ssh_cfg: dict) -> paramiko.SSHClient:
    client = paramiko.SSHClient()
    client.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    client.connect(
        hostname=ssh_cfg["host"],
        port=int(ssh_cfg.get("port", 22)),
        username=ssh_cfg["user"],
        password=ssh_cfg.get("password"),
        timeout=30,
    )
    transport = client.get_transport()
    if transport:
        transport.set_keepalive(30)
    return client


def ssh_exec(client: paramiko.SSHClient, cmd: str, timeout: int = 60) -> tuple[int, str, str]:
    _, stdout, stderr = client.exec_command(cmd, timeout=timeout)
    out = stdout.read().decode("utf-8", errors="replace")
    err = stderr.read().decode("utf-8", errors="replace")
    code = stdout.channel.recv_exit_status()
    return code, out, err


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--case", help="single case id")
    args = parser.parse_args()

    cfg = json.loads(DEPLOY.read_text(encoding="utf-8"))
    agent = cfg["targets"]["geegoo-agent"]["ssh"]
    client = connect(agent)
    sftp = client.open_sftp()
    sftp.put(str(SCRIPT), REMOTE_PATH)
    sftp.close()

    env = ""
    argv = ""
    if args.case:
        env = f"TURNPLAN_ONLY_CASE={json.dumps(args.case)} "
        argv = f" --case {args.case}"

    start_cmd = (
        f"rm -f {REMOTE_LOG}; "
        f"nohup {env}python3 {REMOTE_PATH}{argv} > {REMOTE_LOG} 2>&1 & echo $!"
    )
    _, out, err = ssh_exec(client, start_cmd, timeout=30)
    pid = out.strip().splitlines()[-1] if out.strip() else ""
    print(f"Started on {agent['host']} pid={pid}", flush=True)

    deadline = time.time() + 3600
    last = ""
    while time.time() < deadline:
        time.sleep(15)
        _, log_out, _ = ssh_exec(client, f"tail -n 80 {REMOTE_LOG}", timeout=30)
        if log_out != last:
            # print only new content
            if log_out.startswith(last):
                sys.stdout.write(log_out[len(last) :])
                sys.stdout.flush()
            else:
                sys.stdout.write(log_out)
                sys.stdout.flush()
            last = log_out
        if "======== SUMMARY ========" in log_out:
            break
        _, alive, _ = ssh_exec(
            client,
            f"ps -p {pid} >/dev/null 2>&1 && echo running || echo done",
            timeout=15,
        )
        if pid and "done" in alive and "SUMMARY" in log_out:
            break
        if pid and "done" in alive and log_out and "Traceback" in log_out:
            break

    _, log_out, _ = ssh_exec(client, f"cat {REMOTE_LOG}", timeout=60)
    client.close()

    if "PASS  turn_plan_" in log_out and "FAIL  turn_plan_" not in log_out.split("SUMMARY")[-1]:
        return 0
    if args.case and f"PASS  {args.case}" in log_out:
        return 0
    if "FAIL" in log_out or "Traceback" in log_out:
        return 1
    return 1


if __name__ == "__main__":
    raise SystemExit(main())
