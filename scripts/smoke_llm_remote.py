#!/usr/bin/env python3
"""Smoke test agent LLM with correct runtime auth."""
from __future__ import annotations

import json
import re
import urllib.error
import urllib.request
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


REMOTE = r"""
import json, os, urllib.error, urllib.request

def load_env(path):
    env = {}
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
runtime_key = env.get("GEEGOO_AGENT_RUNTIME_API_KEY", "").strip()
if not runtime_key:
    raise SystemExit("missing GEEGOO_AGENT_RUNTIME_API_KEY in agent.env")

mcp = cfg.get("mcp_token", "")
body = json.dumps({
    "model": "geegoo-agent",
    "messages": [{"role": "user", "content": "只回复 ok"}],
    "stream": False,
}).encode()
req = urllib.request.Request(
    "http://127.0.0.1:3400/v1/chat/completions",
    data=body,
    headers={
        "Authorization": f"Bearer {runtime_key}",
        "Content-Type": "application/json",
        "X-MCP-Token": mcp,
        "X-Approve-Writes": "true",
    },
    method="POST",
)
try:
    with urllib.request.urlopen(req, timeout=120) as resp:
        doc = json.loads(resp.read().decode())
except urllib.error.HTTPError as he:
    print("HTTP", he.code, he.read()[:500].decode("utf-8", "replace"))
    raise SystemExit(1)

choices = doc.get("choices") or []
text = ""
if choices:
    text = (choices[0].get("message") or {}).get("content") or ""
finish = choices[0].get("finish_reason") if choices else ""
print("finish_reason:", finish)
print("assistant:", text[:500])
if finish == "error" or "Authentication Fails" in text or "invalid" in text.lower():
    raise SystemExit(1)
print("SMOKE_OK")
"""


def main() -> None:
    out = ssh(f"python3 <<'PY'\n{REMOTE}\nPY", timeout=180)
    print(out)
    if "SMOKE_OK" not in out:
        raise SystemExit(1)


if __name__ == "__main__":
    main()
