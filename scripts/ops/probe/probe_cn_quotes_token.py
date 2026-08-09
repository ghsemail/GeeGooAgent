#!/usr/bin/env python3
"""Probe CN quotes with service tokens from servers."""
from __future__ import annotations

import json
from pathlib import Path

import paramiko

DEPLOY = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")


def run(target: str, cmd: str) -> str:
    cfg = json.loads(DEPLOY.read_text(encoding="utf-8"))
    ssh = cfg["targets"][target]["ssh"]
    c = paramiko.SSHClient()
    c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    c.connect(ssh["host"], username=ssh["user"], password=ssh["password"], timeout=60)
    _, o, e = c.exec_command(cmd, timeout=90)
    out = (o.read() + e.read()).decode("utf-8", errors="replace")
    c.close()
    return out


def main() -> None:
    py = r'''
import json, os, re, urllib.request
code = "000858.SZ"

def load_env(path):
    env = {}
    try:
        for line in open(path):
            line=line.strip()
            if not line or line.startswith('#') or '=' not in line: continue
            k,v=line.split('=',1); env[k]=v
    except FileNotFoundError:
        pass
    return env

def post(url, body, token=None):
    data=json.dumps(body).encode()
    h={"Content-Type":"application/json"}
    if token: h["Authorization"]=f"Bearer {token}"
    req=urllib.request.Request(url,data=data,headers=h,method="POST")
    try:
        with urllib.request.urlopen(req,timeout=30) as r:
            return r.status, r.read().decode()[:300]
    except urllib.error.HTTPError as e:
        return e.code, e.read().decode()[:300]

cn_env = load_env("/home/ubuntu/apps/GeeGooData/.env")
hk_env = {}
try:
    hk_env = load_env("/root/apps/GeeGooData/.env")
except Exception:
    pass
bot_env = load_env("/home/ubuntu/apps/GeeGooBot/.env")

cn_t = cn_env.get("GEEGOO_DATA_SERVICE_TOKEN","")
hk_t = hk_env.get("GEEGOO_DATA_SERVICE_TOKEN","")
bot_cn_t = bot_env.get("GEEGOO_DATA_CN_SERVICE_TOKEN") or bot_env.get("GEEGOO_DATA_SERVICE_TOKEN","")

print("tokens", "cn", bool(cn_t), "hk", bool(hk_t), "bot_cn", bool(bot_cn_t))

for label, url, token in [
    ("cn+token", "http://127.0.0.1:3300", cn_t),
    ("cn-node quote", "http://82.157.97.76:3300", cn_t),
    ("hk-node quote CN", "http://47.80.14.120:3300", hk_t),
    ("bot token to cn", "http://82.157.97.76:3300", bot_cn_t),
]:
    s,b = post(url+"/v1/market/quote", {"code": code}, token)
    print(label, s, b)

s,b = post("http://127.0.0.1:3300/v1/market/klines", {"code":code,"frequency":"daily","limit":3}, cn_t)
print("cn klines", s, b[:250])

# futu opend
import socket
s=socket.socket(); s.settimeout(2)
try:
    s.connect(("127.0.0.1",11111)); print("opend port 11111 open")
except Exception as e:
    print("opend port", e)
finally:
    s.close()
'''
    print("## CN node")
    print(run("geegoo-data-cn", f"python3 -c {json.dumps(py)}"))
    print("## Bot env data cn")
    print(run("geegoo-bot", "grep -E 'GEEGOO_DATA|CN_' /home/ubuntu/apps/GeeGooBot/.env"))
    print("## Signal env data")
    print(run("geegoo-tradingsignal", "grep -E 'GEEGOO_DATA' /root/apps/GeeGooSignal/.env"))


if __name__ == "__main__":
    main()
