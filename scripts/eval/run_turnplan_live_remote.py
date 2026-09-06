#!/usr/bin/env python3
"""Upload TurnPlan live eval runner to geegoo-agent host and execute in background."""
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

ROOT = Path(__file__).resolve().parent
DEPLOY = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")
REMOTE_DIR = "/tmp/geegoo_turnplan_eval"
REMOTE_LOG = f"{REMOTE_DIR}/run.log"


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


def ssh_exec(client: paramiko.SSHClient, cmd: str, timeout: int = 120) -> tuple[int, str, str]:
    _, stdout, stderr = client.exec_command(cmd, timeout=timeout)
    out = stdout.read().decode("utf-8", errors="replace")
    err = stderr.read().decode("utf-8", errors="replace")
    return stdout.channel.recv_exit_status(), out, err


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--case")
    parser.add_argument("--category")
    parser.add_argument("--list", action="store_true")
    parser.add_argument("--timeout", type=int, default=7200, help="poll timeout seconds")
    args = parser.parse_args()

    cfg = json.loads(DEPLOY.read_text(encoding="utf-8"))
    agent = cfg["targets"]["geegoo-agent"]["ssh"]
    client = connect(agent)
    sftp = client.open_sftp()
    try:
        ssh_exec(client, f"mkdir -p {REMOTE_DIR}")
        sftp.put(str(ROOT / "run_turnplan_live.py"), f"{REMOTE_DIR}/run_turnplan_live.py")
        sftp.put(str(ROOT / "turnplan_cases.json"), f"{REMOTE_DIR}/turnplan_cases.json")
    finally:
        sftp.close()

    argv = []
    if args.list:
        argv.append("--list")
    if args.case:
        argv.append(f"--case {args.case}")
    if args.category:
        argv.append(f"--category {args.category}")
    env = ""
    if args.case:
        env = f"TURNPLAN_ONLY_CASE={json.dumps(args.case)} "

    start_cmd = (
        f"rm -f {REMOTE_LOG}; "
        f"cd {REMOTE_DIR} && {env}nohup python3 run_turnplan_live.py {' '.join(argv)} "
        f"> {REMOTE_LOG} 2>&1 & echo $!"
    )
    _, out, _ = ssh_exec(client, start_cmd, timeout=30)
    pid = out.strip().splitlines()[-1] if out.strip() else ""
    print(f"Started on {agent['host']} pid={pid}", flush=True)

    if args.list:
        deadline = time.time() + 60
    else:
        deadline = time.time() + args.timeout
    last = ""
    while time.time() < deadline:
        time.sleep(15)
        _, log_out, _ = ssh_exec(client, f"tail -n 100 {REMOTE_LOG}", timeout=30)
        if log_out != last:
            chunk = log_out[len(last) :] if log_out.startswith(last) else log_out
            sys.stdout.write(chunk)
            sys.stdout.flush()
            last = log_out
        if args.list and log_out.strip():
            break
        if "======== SUMMARY ========" in log_out:
            break
        if pid:
            _, alive, _ = ssh_exec(
                client, f"ps -p {pid} >/dev/null 2>&1 && echo running || echo done", timeout=15
            )
            if "done" in alive and ("SUMMARY" in log_out or "Traceback" in log_out):
                break

    _, log_out, _ = ssh_exec(client, f"cat {REMOTE_LOG}", timeout=60)
    client.close()

    if args.list:
        return 0
    if "Traceback" in log_out:
        return 1
    summary = log_out.split("======== SUMMARY ========")[-1] if "SUMMARY" in log_out else log_out
    if "FAIL" in summary:
        return 1
    return 0 if "PASS" in summary else 1


if __name__ == "__main__":
    raise SystemExit(main())
