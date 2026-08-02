#!/usr/bin/env python3
"""Commit, push, and deploy ops-log date range query (GeeGooSignal + TradingData + trading_operation)."""
from __future__ import annotations

import json
import subprocess
import sys
import time
from pathlib import Path

import paramiko

DEPLOY_CFG = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")
GIT = r"C:\Program Files\Git\bin\git.exe"


def git(repo: str, *args: str) -> subprocess.CompletedProcess[str]:
    return subprocess.run([GIT, "-C", repo, *args], capture_output=True, text=True, check=False)


def commit_push(repo: str, files: list[str], message: str) -> None:
    print(f"\n=== commit {repo} ===")
    for f in files:
        r = git(repo, "add", f)
        if r.returncode != 0:
            raise RuntimeError(r.stderr or r.stdout)
    diff = git(repo, "diff", "--cached", "--quiet")
    if diff.returncode == 0:
        print("no staged changes, skip commit")
        return
    r = git(repo, "commit", "-m", message)
    print(r.stdout or r.stderr)
    if r.returncode != 0:
        raise RuntimeError(f"commit failed: {r.stderr or r.stdout}")
    r = git(repo, "push", "origin", "main")
    print(r.stdout or r.stderr)
    if r.returncode != 0:
        raise RuntimeError(f"push failed: {r.stderr or r.stdout}")


def ssh_run(host: str, user: str, password: str, cmd: str) -> str:
    client = paramiko.SSHClient()
    client.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    client.connect(host, 22, user, password, timeout=30)
    _, stdout, stderr = client.exec_command(cmd, timeout=300)
    out = stdout.read().decode("utf-8", "replace")
    err = stderr.read().decode("utf-8", "replace")
    client.close()
    if out.strip():
        print(out.rstrip())
    if err.strip():
        print("STDERR:", err.rstrip())
    return out


def main() -> int:
    commit_push(
        r"D:\Geegoo\GeeGooSignal",
        [
            "internal/catalog/admin/util.go",
            "internal/catalog/admin/strategy_logs.go",
            "internal/catalog/admin/news_logs.go",
        ],
        "feat(catalog-api): support run_date_from/run_date_to for ops log queries",
    )
    commit_push(
        r"D:\Geegoo\TradingData",
        ["News/NewsCenter.py", "NewsServer.py"],
        "feat(news): support run_date_from/run_date_to in refresh log queries",
    )
    commit_push(
        r"D:\Geegoo\trading_operation",
        [
            "lib/api/news_server.dart",
            "lib/api/strategy_log_server.dart",
            "lib/modules/news_refresh_log_mgt/controllers/news_refresh_log_controller.dart",
            "lib/modules/news_refresh_log_mgt/controllers/strategy_generation_log_controller.dart",
            "lib/modules/news_refresh_log_mgt/widgets/news_refresh_log_tab.dart",
            "lib/modules/news_refresh_log_mgt/widgets/strategy_generation_log_tab.dart",
            "lib/modules/news_refresh_log_mgt/widgets/ops_log_filter_widgets.dart",
            "pubspec.yaml",
            "pubspec.lock",
            "web/index.html",
        ],
        "feat(ops-log): date range filter UI and API params for maintenance logs",
    )

    cfg = json.loads(DEPLOY_CFG.read_text(encoding="utf-8"))

    sig = cfg["targets"]["geegoo-signal"]["ssh"]
    print("\n=== deploy GeeGooSignal ===")
    ssh_run(
        sig["host"],
        sig["user"],
        sig["password"],
        "cd /root/apps/GeeGooSignal && git fetch origin main && git reset --hard origin/main && bash start.sh restart",
    )
    time.sleep(8)
    ssh_run(
        sig["host"],
        sig["user"],
        sig["password"],
        "curl -sf http://127.0.0.1:3210/health && echo && "
        "curl -s -X POST http://127.0.0.1:3210/getStrategyGenerationLogs "
        "-H 'Content-Type: application/json' "
        "-H 'Authorization: Bearer 850367bc6d5fe8a4a53f267f5c308ac6d2ec1474d1764fe9' "
        "-d '{\"limit\":1,\"run_date_from\":\"2026-01-01\",\"run_date_to\":\"2026-12-31\"}' | head -c 200; echo",
    )

    data = cfg["targets"]["geegoo-tradingdata"]["ssh"]
    print("\n=== deploy TradingData NewsServer ===")
    ssh_run(
        data["host"],
        data["user"],
        data["password"],
        "cd /root/apps/TradingData && git fetch origin main && git reset --hard origin/main && printf '4\\n0\\n' | bash start.sh",
    )
    time.sleep(3)
    ssh_run(
        data["host"],
        data["user"],
        data["password"],
        "curl -s -X POST http://127.0.0.1:5800/getNewsRefreshLogs "
        "-H 'Content-Type: application/json' "
        "-d '{\"limit\":1,\"run_date_from\":\"2026-01-01\",\"run_date_to\":\"2026-12-31\"}' | head -c 200; echo",
    )

    print("\n=== build + deploy trading_operation web ===")
    r = subprocess.run(
        ["flutter", "build", "web"],
        cwd=r"D:\Geegoo\trading_operation",
        capture_output=True,
        text=True,
    )
    if r.returncode != 0:
        print(r.stdout[-2000:])
        print(r.stderr[-2000:])
        return 1
    print("\n".join(r.stdout.splitlines()[-3:]))
    r = subprocess.run(
        [sys.executable, str(DEPLOY_CFG.parent / "scripts" / "deploy_trading_operation_web.py")],
        capture_output=True,
        text=True,
    )
    print(r.stdout or r.stderr)
    if r.returncode != 0:
        return 1

    print("\n=== done ===")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
