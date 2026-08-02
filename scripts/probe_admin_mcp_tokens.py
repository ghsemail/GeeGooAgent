#!/usr/bin/env python3
import json
from pathlib import Path
import paramiko

cfg = json.loads(Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json").read_text(encoding="utf-8"))

def run_ssh(target: str, cmd: str) -> str:
    s = cfg["targets"][target]["ssh"]
    c = paramiko.SSHClient()
    c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    c.connect(s["host"], username=s["user"], password=s["password"], timeout=60)
    _, o, e = c.exec_command(cmd, timeout=40)
    out = o.read().decode("utf-8", errors="replace")
    err = e.read().decode("utf-8", errors="replace")
    c.close()
    return out + err

bot_cmd = (
    "mongosh 'mongodb://ghsemail:Ghs%402022@127.0.0.1:27017/QT_DB?authSource=QT_DB' "
    "--quiet --eval 'JSON.stringify(db.admin.find({},{username:1,mcp_token:1}).limit(5).toArray())'"
)
signal_signal_cmd = (
    "mongosh 'mongodb://127.0.0.1:27017/Signal_DB' "
    "--quiet --eval 'JSON.stringify(db.admin.find({},{username:1,mcp_token:1}).limit(5).toArray())'"
)

print("=== bot QT_DB.admin ===")
print(run_ssh("geegoo-bot", bot_cmd))
print("=== signal Signal_DB.admin ===")
print(run_ssh("geegoo-signal", signal_signal_cmd))
