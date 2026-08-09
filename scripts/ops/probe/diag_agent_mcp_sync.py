#!/usr/bin/env python3
from __future__ import annotations

import json
import urllib.error
import urllib.request
from pathlib import Path

import paramiko

DEPLOY = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")

SIGNAL = r"""
import json, subprocess
names = subprocess.check_output("docker ps --format '{{.Names}}'", shell=True, text=True).splitlines()
mongo = next((n for n in names if 'mongo' in n.lower()), '')
raw = subprocess.check_output(['docker','exec',mongo,'mongosh','Signal_DB','--quiet','--eval',
 "JSON.stringify(db.admin.find({username:'ghsemail'},{mcp_token:1,username:1,_id:1}).toArray())"], text=True).strip()
print(raw)
"""

BOT = r"""
import json, subprocess
names = subprocess.check_output("docker ps --format '{{.Names}}'", shell=True, text=True).splitlines()
mongo = next((n for n in names if 'mongo' in lower()), '')
"""

BOT = r"""
import subprocess
names = subprocess.check_output("docker ps --format '{{.Names}}'", shell=True, text=True).splitlines()
mongo = next((n for n in names if 'mongo' in n.lower()), '')
raw = subprocess.check_output(['docker','exec',mongo,'mongosh','QT_DB','--quiet','--eval',
 "JSON.stringify(db.admin.find({username:'ghsemail'},{mcp_token:1,username:1,_id:1}).toArray())"], text=True).strip()
print(raw)
"""


def ssh_run(target: str, py: str) -> str:
    cfg = json.loads(DEPLOY.read_text(encoding="utf-8"))
    s = cfg["targets"][target]["ssh"]
    c = paramiko.SSHClient()
    c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    c.connect(s["host"], username=s["user"], password=s.get("password"), timeout=60)
    _, o, e = c.exec_command(f"python3 <<'PY'\n{py}\nPY", timeout=60)
    out = o.read().decode().strip()
    err = e.read().decode().strip()
    c.close()
    if err:
        print("stderr", err)
    return out


def probe(url: str, token: str, api_key: str = "") -> None:
    headers = {"X-MCP-Token": token}
    if api_key:
        headers["Authorization"] = f"Bearer {api_key}"
    req = urllib.request.Request(url, headers=headers)
    try:
        with urllib.request.urlopen(req, timeout=20) as resp:
            print(url, "->", resp.status, resp.read(120))
    except urllib.error.HTTPError as ex:
        print(url, "->", ex.code, ex.read()[:160])


def main() -> None:
    sig = json.loads(ssh_run("geegoo-signal", SIGNAL) or "[]")
    bot = json.loads(ssh_run("geegoo-bot", BOT) or "[]")
    print("signal", sig)
    print("bot", bot)
    token = (sig[0].get("mcp_token") if sig else "") or ""
    print("token prefix", token[:12])
    cfg = json.loads(DEPLOY.read_text(encoding="utf-8"))
    s = cfg["targets"]["geegoo-bot"]["ssh"]
    c = paramiko.SSHClient()
    c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    c.connect(s["host"], username=s["user"], password=s.get("password"), timeout=60)
    _, o, _ = c.exec_command("grep '^GEEGOO_BOT_AGENT_API_KEY=' /home/ubuntu/apps/GeeGooBot/.env", timeout=20)
    api_key = o.read().decode().strip().split("=", 1)[1]
    c.close()
    probe("http://118.195.135.97:3110/op_agent/v1/dashboard/data", token, api_key)
    probe("http://146.56.225.252:8088/op_agent/v1/dashboard/data", token)


if __name__ == "__main__":
    main()
