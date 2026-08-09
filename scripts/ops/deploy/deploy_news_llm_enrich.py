#!/usr/bin/env python3
"""Deploy news LLM bilingual enrichment (GeeGooSignal analyze-api + GeeGooData news-worker)."""
from __future__ import annotations

import json
import subprocess
import sys
import time
from pathlib import Path

import paramiko

DEPLOY_CFG = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")
GIT = r"C:\Program Files\Git\bin\git.exe"


def git(repo: str, *args: str) -> None:
    r = subprocess.run([GIT, "-C", repo, *args], capture_output=True, text=True)
    if r.returncode != 0:
        raise RuntimeError(r.stderr or r.stdout)


def ssh_run(host: str, user: str, password: str, cmd: str) -> str:
    client = paramiko.SSHClient()
    client.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    client.connect(host, username=user, password=password, timeout=30)
    _, stdout, stderr = client.exec_command(cmd, timeout=600)
    out = stdout.read().decode("utf-8", "replace").strip()
    err = stderr.read().decode("utf-8", "replace").strip()
    client.close()
    if out:
        print(out)
    if err:
        print("STDERR:", err)
    return out


def upsert_env(remote_dir: str, key: str, value: str) -> str:
    esc = value.replace("'", "'\"'\"'")
    return (
        f"grep -q '^{key}=' {remote_dir}/.env 2>/dev/null && "
        f"sed -i 's|^{key}=.*|{key}={esc}|' {remote_dir}/.env || "
        f"echo '{key}={esc}' >> {remote_dir}/.env"
    )


def main() -> int:
    cfg = json.loads(DEPLOY_CFG.read_text(encoding="utf-8"))
    sig = cfg["targets"]["geegoo-signal"]["ssh"]
    data = cfg["targets"]["geegoo-tradingdata"]["ssh"]

    analyze_key = ssh_run(
        sig["host"], sig["user"], sig["password"],
        "grep '^GEEGOO_SIGNAL_ANALYZE_API_KEY=' /root/apps/GeeGooSignal/.env | cut -d= -f2-",
    ).strip()
    if not analyze_key:
        print("ERROR: missing GEEGOO_SIGNAL_ANALYZE_API_KEY on GeeGooSignal")
        return 1

    print("=== deploy GeeGooSignal ===")
    ssh_run(
        sig["host"], sig["user"], sig["password"],
        "cd /root/apps/GeeGooSignal && git fetch origin main && git reset --hard origin/main && bash start.sh restart",
    )
    time.sleep(8)
    ssh_run(
        sig["host"], sig["user"], sig["password"],
        "curl -sf -X POST http://127.0.0.1:3230/enrichStockNews "
        "-H 'Content-Type: application/json' "
        f"-H 'Authorization: Bearer {analyze_key}' "
        "-d '{\"title\":\"Apple shares surge\",\"snippet\":\"Revenue beat\"}' | head -c 300; echo",
    )

    print("=== deploy GeeGooData + news-worker env ===")
    data_dir = "/root/apps/GeeGooData"
    env_cmds = [
        upsert_env(data_dir, "GEEGOO_SIGNAL_ANALYZE_API_URL", "http://146.56.225.252:3230"),
        upsert_env(data_dir, "GEEGOO_SIGNAL_ANALYZE_API_KEY", analyze_key),
        upsert_env(data_dir, "GEEGOO_DATA_NEWS_LLM_ENABLED", "true"),
    ]
    ssh_run(
        data["host"], data["user"], data["password"],
        " && ".join(env_cmds) + f" && cd {data_dir} && git fetch origin main && git reset --hard origin/main && bash start.sh restart",
    )
    time.sleep(5)
    print("=== run news-worker once (sample) ===")
    ssh_run(
        data["host"], data["user"], data["password"],
        f"cd {data_dir} && set -a && source .env && set +a && ./bin/news-worker -once 2>&1 | tail -12",
    )
    print("=== verify AAPL cn title ===")
    ssh_run(
        data["host"], data["user"], data["password"],
        "curl -sf -X POST http://127.0.0.1:3300/getStockNews -H 'Content-Type: application/json' "
        "-d '{\"stock_list\":[{\"code\":\"AAPL.US\",\"name\":{\"init\":\"Apple\"}}],\"language\":\"cn\"}' "
        "| python3 -c \"import sys,json; d=json.load(sys.stdin); n=d[0] if d else {}; t=n.get('title',{}); print('title.cn=', (t.get('cn') or '')[:80]); print('has_chinese=', any('\\u4e00' <= c <= '\\u9fff' for c in (t.get('cn') or '')))\"",
    )
    return 0


if __name__ == "__main__":
    sys.exit(main())
