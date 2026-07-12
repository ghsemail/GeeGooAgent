#!/usr/bin/env python3
"""Probe native analyze-api chain after Go migration."""
from __future__ import annotations

import json
from pathlib import Path

import paramiko

DEPLOY = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")
UID = "64afddf8c2a269ac1846fe70"
BODY = json.dumps({
    "user_id": UID,
    "name": "SpaceX",
    "code": "SPCX.US",
    "prompt_id": "6a006854b9ccd3d9befc6c24",
    "period": "daily",
    "language": "cn",
})


def ssh(target: str, cmd: str, timeout: int = 120) -> str:
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
    print(f"\n>>> [{target}] {cmd}\n{out}{err}")
    if code != 0:
        print(f"exit {code}")
    return out


def main() -> None:
    ssh("geegoo-tradingbot", f"curl -s -w '\\nHTTP:%{{http_code}}' -X POST http://127.0.0.1:3140/checkUser -H 'Content-Type: application/json' -d '{json.dumps({'user_id': UID})}'")
    ssh(
        "geegoo-tradingsignal",
        "ANA=$(grep GEEGOO_SIGNAL_ANALYZE_API_KEY /root/apps/GeeGooSignal/.env|cut -d= -f2-); "
        f"curl -s -w '\\nHTTP:%{{http_code}}' -m 60 -X POST http://127.0.0.1:3230/getMCPAnalysis "
        f"-H \"Authorization: Bearer $ANA\" -H 'Content-Type: application/json' -d '{BODY}' | tail -c 1200",
    )
    ssh("geegoo-tradingsignal", "tail -40 /root/apps/GeeGooSignal/analyze-api.out")
    ssh(
        "geegoo-tradingbot",
        "MCP=$(grep GEEGOO_BOT_MCP_API_KEY /home/ubuntu/apps/GeeGooBot/.env|cut -d= -f2-); "
        "curl -s -w '\\nHTTP:%{{http_code}}' -m 60 -X POST http://127.0.0.1:3120/getMCPAnalysis "
        f"-H \"Authorization: Bearer $MCP\" -H 'Content-Type: application/json' "
        "-d '{\"mcp_token\":\"mcp_HVTSYfumrCexAU66EutTM4v2A5aGYXiF\",\"name\":\"SpaceX\",\"code\":\"SPCX.US\","
        "\"prompt_id\":\"6a006854b9ccd3d9befc6c24\",\"period\":\"daily\",\"language\":\"cn\"}' | tail -c 1200",
    )
    ssh("geegoo-tradingdata", "DATA=$(grep GEEGOO_DATA_SERVICE_TOKEN /root/apps/GeeGooData/.env 2>/dev/null | cut -d= -f2-); curl -s -w '\\nHTTP:%{{http_code}}' -m 30 -X POST http://127.0.0.1:3300/v1/market/klines -H \"Authorization: Bearer $DATA\" -H 'Content-Type: application/json' -d '{\"code\":\"SPCX.US\",\"frequency\":\"1d\",\"limit\":30}' | tail -c 800")


if __name__ == "__main__":
    main()
