#!/usr/bin/env python3
"""Verify LLM after key fix."""
from __future__ import annotations

import json
from pathlib import Path

import paramiko

DEPLOY = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")


def ssh(cmd: str, timeout: int = 180) -> str:
    cfg = json.loads(DEPLOY.read_text(encoding="utf-8"))
    s = cfg["targets"]["geegoo-agent"]["ssh"]
    c = paramiko.SSHClient()
    c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    c.connect(s["host"], username=s["user"], password=s.get("password"), timeout=30)
    _, o, e = c.exec_command(cmd, timeout=timeout)
    out = o.read().decode("utf-8", errors="replace")
    err = e.read().decode("utf-8", errors="replace")
    c.close()
    return out + err


def main() -> None:
    print(ssh("grep -E 'LLM|运营|queryModel' /home/ubuntu/.geegoo/geegoo-agent/agent-runtime.out | tail -5"))
    print(ssh("grep GEEGOO_AGENT_RUNTIME_API_KEY /home/ubuntu/.geegoo/agent.env || true"))
    py = r'''
import json, os, urllib.request
cfg=json.load(open("/home/ubuntu/.geegoo/config.json"))
env={}
for line in open("/home/ubuntu/.geegoo/agent.env"):
    line=line.strip()
    if line.startswith("export "):
        k,v=line[7:].split("=",1)
        env[k]=v.strip('"')
key=env.get("GEEGOO_AGENT_RUNTIME_API_KEY") or cfg.get("runtime_api_key","")
mcp=cfg.get("mcp_token","")
body=json.dumps({"message":"ping","session_id":"chat-llm-key-smoke"}).encode()
req=urllib.request.Request(
    "http://127.0.0.1:3400/v1/chat/stream",
    data=body,
    headers={"Authorization":"Bearer "+key,"Content-Type":"application/json","X-MCP-Token":mcp,"X-Approve-Writes":"true"},
    method="POST",
)
with urllib.request.urlopen(req, timeout=120) as resp:
    text=resp.read().decode("utf-8", errors="replace")
print(text[-1500:])
'''
    out = ssh(f"python3 <<'PY'\n{py}\nPY")
    print(out)
    if "Authentication Fails" in out or '"failed":true' in out:
        raise SystemExit(1)
    print("OK")


if __name__ == "__main__":
    main()
