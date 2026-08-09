#!/usr/bin/env python3
"""Sync GEEGOO_SIGNAL_SIGNAL_API_KEY on GeeGooBot from GeeGooSignal and restart app-api."""
from __future__ import annotations

import json
import re
import sys
from pathlib import Path

import paramiko

DEPLOY = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")
BOT_DIR = "/home/ubuntu/apps/GeeGooBot"
SIG_ENV = "/root/apps/GeeGooSignal/.env"
BOT_ENV = f"{BOT_DIR}/.env"


def ssh_client(ssh_cfg: dict) -> paramiko.SSHClient:
    c = paramiko.SSHClient()
    c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    c.connect(
        hostname=ssh_cfg["host"],
        port=int(ssh_cfg.get("port", 22)),
        username=ssh_cfg["user"],
        password=ssh_cfg.get("password"),
        timeout=60,
    )
    return c


def run(c: paramiko.SSHClient, cmd: str) -> str:
    _, o, e = c.exec_command(cmd)
    return (o.read() + e.read()).decode().strip()


def grep_key(c: paramiko.SSHClient, path: str, name: str) -> str:
    return run(c, f"grep '^{name}=' {path} 2>/dev/null | cut -d= -f2- || true")


def set_env_key(c: paramiko.SSHClient, path: str, name: str, value: str) -> None:
    remote = f"""
import re
from pathlib import Path
p = Path({path!r})
text = p.read_text(encoding='utf-8')
key = {name!r}
val = {value!r}
pat = rf'^' + re.escape(key) + r'=.*$'
if re.search(pat, text, re.M):
    text = re.sub(pat, key + '=' + val, text, flags=re.M)
else:
    text = text.rstrip() + '\\n' + key + '=' + val + '\\n'
p.write_text(text, encoding='utf-8')
print('set', key)
"""
    import base64

    payload = base64.b64encode(remote.encode()).decode()
    out = run(c, f"python3 -c \"import base64; exec(base64.b64decode('{payload}').decode())\"")
    print(out)


def main() -> int:
    cfg = json.loads(DEPLOY.read_text(encoding="utf-8"))
    sig_c = ssh_client(cfg["targets"]["geegoo-tradingsignal"]["ssh"])
    bot_c = ssh_client(cfg["targets"]["geegoo-bot"]["ssh"])

    signal_key = grep_key(sig_c, SIG_ENV, "GEEGOO_SIGNAL_SIGNAL_API_KEY")
    if not signal_key:
        print("GEEGOO_SIGNAL_SIGNAL_API_KEY missing on signal host", file=sys.stderr)
        return 1
    print(f"signal-api key: {signal_key[:16]}...")

    old = grep_key(bot_c, BOT_ENV, "GEEGOO_SIGNAL_SIGNAL_API_KEY")
    print(f"bot old key: {old[:16] + '...' if old else '(empty)'}")
    set_env_key(bot_c, BOT_ENV, "GEEGOO_SIGNAL_SIGNAL_API_KEY", signal_key)

    print(run(bot_c, f"cd {BOT_DIR} && printf '3\\n' | bash start.sh"))
    verify = run(
        bot_c,
        "curl -s -H 'Authorization: Bearer "
        + grep_key(bot_c, BOT_ENV, "GEEGOO_BOT_APP_API_KEY")
        + "' -H 'Content-Type: application/json' "
        "-d '{\"user_id\":\"67935cda6272feb48b49ba49\",\"type\":\"flag\",\"frequency\":\"5m\",\"signal_index_list\":[],\"language\":\"cn\"}' "
        "http://127.0.0.1:3100/getUserStockTrend | head -c 200",
    )
    print("getUserStockTrend:", verify)
    sig_c.close()
    bot_c.close()
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
