#!/usr/bin/env python3
"""Deploy getMCPAnalysis + run SpaceX analysis E2E test."""
from __future__ import annotations

import json
import sys
import time
from pathlib import Path

import paramiko

DEPLOY = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")
MESSAGE = "分析 SpaceX 日信号趋势"
SESSION_ID = "chat-4f7f68b32257"


def ssh_run(target: str, cmd: str, timeout: int = 600) -> str:
    cfg = json.loads(DEPLOY.read_text(encoding="utf-8"))
    s = cfg["targets"][target]["ssh"]
    c = paramiko.SSHClient()
    c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    c.connect(s["host"], username=s["user"], password=s.get("password"), timeout=30)
    _, o, e = c.exec_command(cmd, timeout=timeout)
    out = o.read().decode("utf-8", errors="replace")
    err = e.read().decode("utf-8", errors="replace")
    code = o.channel.recv_exit_status()
    c.close()
    if code != 0:
        raise RuntimeError(f"{target} exit {code}: {err.strip() or out.strip()}")
    return out


def deploy_signal() -> None:
    cmds = [
        "cd /root/apps/GeeGooSignal && git fetch origin main && git reset --hard origin/main",
        "cd /root/apps/GeeGooSignal && git log -1 --oneline",
        "cd /root/apps/GeeGooSignal && chmod +x start.sh && bash start.sh restart",
        "sleep 3 && curl -sf http://127.0.0.1:3230/health",
    ]
    for cmd in cmds:
        print(f"\n>>> [signal] {cmd}")
        print(ssh_run("geegoo-tradingsignal", cmd))


def deploy_bot() -> str:
    ana_key = ssh_run(
        "geegoo-tradingsignal",
        "grep GEEGOO_SIGNAL_ANALYZE_API_KEY /root/apps/GeeGooSignal/.env | cut -d= -f2-",
    ).strip()
    cat_key = ssh_run(
        "geegoo-tradingsignal",
        "grep GEEGOO_SIGNAL_CATALOG_API_KEY /root/apps/GeeGooSignal/.env | cut -d= -f2-",
    ).strip()
    cmds = [
        "cd /home/ubuntu/apps/GeeGooBot && git fetch origin main && git reset --hard origin/main",
        "cd /home/ubuntu/apps/GeeGooBot && git log -1 --oneline",
        "cd /home/ubuntu/apps/GeeGooBot && bash scripts/bootstrap_env.sh",
        f"grep -q '^GEEGOO_SIGNAL_ANALYZE_API_URL=' /home/ubuntu/apps/GeeGooBot/.env || echo 'GEEGOO_SIGNAL_ANALYZE_API_URL=http://146.56.225.252:3230' >> /home/ubuntu/apps/GeeGooBot/.env",
        f"sed -i 's|^GEEGOO_SIGNAL_ANALYZE_API_KEY=.*|GEEGOO_SIGNAL_ANALYZE_API_KEY={ana_key}|' /home/ubuntu/apps/GeeGooBot/.env",
        f"sed -i 's|^GEEGOO_SIGNAL_CATALOG_API_KEY=.*|GEEGOO_SIGNAL_CATALOG_API_KEY={cat_key}|' /home/ubuntu/apps/GeeGooBot/.env",
        "cd /home/ubuntu/apps/GeeGooBot && echo 1 | bash start.sh",
        "sleep 4 && curl -sf http://127.0.0.1:3120/health",
    ]
    for cmd in cmds:
        print(f"\n>>> [bot] {cmd}")
        print(ssh_run("geegoo-tradingbot", cmd))
    return ssh_run("geegoo-tradingbot", "grep GEEGOO_BOT_MCP_API_KEY /home/ubuntu/apps/GeeGooBot/.env | cut -d= -f2-").strip()


def probe_mcp(mcp_key: str) -> None:
    tok = "mcp_HVTSYfumrCexAU66EutTM4v2A5aGYXiF"
    tpl = (
        f"curl -s -w '\\nHTTP:%{{http_code}}' -X POST http://127.0.0.1:3120/getSinglePromptTemplate "
        f"-H 'Authorization: Bearer {mcp_key}' -H 'Content-Type: application/json' "
        f"-d '{{\"mcp_token\":\"{tok}\",\"type\":\"tech\",\"period\":\"daily\"}}' | tail -c 400"
    )
    print("\n>>> [bot] probe getSinglePromptTemplate")
    print(ssh_run("geegoo-tradingbot", tpl))


def run_chat_stream() -> str:
    py = f"""
import json, urllib.request

def load_env(path):
    env = {{}}
    for line in open(path):
        line = line.strip()
        if not line or line.startswith("#"):
            continue
        if line.startswith("export "):
            line = line[7:]
        if "=" not in line:
            continue
        k, v = line.split("=", 1)
        env[k] = v.strip().strip('"').strip("'")
    return env

cfg = json.load(open("/home/ubuntu/.geegoo/config.json"))
env = load_env("/home/ubuntu/.geegoo/agent.env")
key = env.get("GEEGOO_AGENT_RUNTIME_API_KEY", "").strip()
mcp = cfg.get("mcp_token", "")
body = json.dumps({{"message": {MESSAGE!r}, "session_id": {SESSION_ID!r}}}).encode()
headers = {{
    "Content-Type": "application/json",
    "X-MCP-Token": mcp,
    "X-Approve-Writes": "true",
}}
if key:
    headers["Authorization"] = f"Bearer {{key}}"
req = urllib.request.Request(
    "http://127.0.0.1:3400/v1/chat/stream",
    data=body,
    headers=headers,
    method="POST",
)
with urllib.request.urlopen(req, timeout=600) as resp:
    print(resp.read().decode("utf-8", errors="replace"))
"""
    print(f"\n>>> [agent] POST /v1/chat/stream: {MESSAGE}")
    t0 = time.time()
    out = ssh_run("geegoo-agent", f"python3 <<'PY'\n{py}\nPY", timeout=620)
    print(out)
    print(f"\n--- elapsed {time.time()-t0:.1f}s ---")
    return out


def main() -> int:
    deploy_signal()
    mcp_key = deploy_bot()
    probe_mcp(mcp_key)
    out = run_chat_stream()
    if "event: turn_end" not in out or "event: done" not in out:
        print("FAIL: stream incomplete", file=sys.stderr)
        return 1
    if '"failed":true' in out or "HTTP 404" in out:
        print("FAIL: turn failed or 404", file=sys.stderr)
        return 1
    if "analysis_result" in out or "信号" in out or "SpaceX" in out or "SPCX" in out:
        print("OK: analysis content present")
    else:
        print("WARN: stream completed but analysis keywords not found in SSE body")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
