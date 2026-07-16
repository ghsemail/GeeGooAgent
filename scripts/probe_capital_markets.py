#!/usr/bin/env python3
"""Probe get_capital_flow / get_capital_distribution for A/HK/US markets."""
from __future__ import annotations

import json
import re
import urllib.request
from pathlib import Path

import paramiko

DEPLOY = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")

CODES = [
    ("A股", "600519.SH"),
    ("港股", "00700.HK"),
    ("美股", "AAPL.US"),
]


def ssh_run(ssh_cfg: dict, cmd: str, timeout: int = 90) -> tuple[int, str, str]:
    client = paramiko.SSHClient()
    client.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    client.connect(
        hostname=ssh_cfg["host"],
        port=int(ssh_cfg.get("port", 22)),
        username=ssh_cfg["user"],
        password=ssh_cfg.get("password"),
        timeout=30,
    )
    _, stdout, stderr = client.exec_command(cmd, timeout=timeout)
    out = stdout.read().decode("utf-8", errors="replace")
    err = stderr.read().decode("utf-8", errors="replace")
    code = stdout.channel.recv_exit_status()
    client.close()
    return code, out, err


def curl_on_host(ssh_cfg: dict, url: str, body: dict, bearer: str = "") -> tuple[str, str]:
    payload = json.dumps(body, ensure_ascii=False).replace("'", "'\\''")
    auth = f"-H 'Authorization: Bearer {bearer}' " if bearer else ""
    cmd = (
        f"curl -sS -m 60 -w '\\nHTTP:%{{http_code}}' "
        f"{auth}-H 'Content-Type: application/json' "
        f"-d '{payload}' {url}"
    )
    _, out, err = ssh_run(ssh_cfg, cmd, timeout=75)
    text = out if out.strip() else err
    m = re.search(r"HTTP:(\d{3})", text)
    http = m.group(1) if m else "?"
    body_txt = re.sub(r"\nHTTP:\d{3}$", "", text.strip())
    return http, body_txt


def eastmoney_flow(code: str) -> tuple[bool, str]:
    secid = "1." + code.replace(".SH", "") if code.endswith(".SH") else "0." + code.replace(".SZ", "")
    url = (
        "https://push2.eastmoney.com/api/qt/stock/fflow/kline/get"
        f"?lmt=1&klt=101&secid={secid}&fields1=f1,f2,f3,f7&fields2=f51,f52,f53,f54,f55,f56,f57,f58,f59,f60,f61,f62,f63,f64,f65"
    )
    try:
        with urllib.request.urlopen(url, timeout=15) as resp:
            raw = json.loads(resp.read().decode())
        klines = raw.get("data", {}).get("klines") or []
        if not klines:
            return False, "empty klines"
        return True, klines[-1][:80]
    except Exception as e:
        return False, str(e)


def eastmoney_dist(code: str) -> tuple[bool, str]:
    secid = "1." + code.replace(".SH", "") if code.endswith(".SH") else "0." + code.replace(".SZ", "")
    url = (
        "https://push2.eastmoney.com/api/qt/stock/get"
        f"?secid={secid}&fields=f62,f184,f66,f69,f72,f75,f78,f81,f84,f87,f204,f205,f124"
    )
    try:
        with urllib.request.urlopen(url, timeout=15) as resp:
            raw = json.loads(resp.read().decode())
        data = raw.get("data") or {}
        if not data:
            return False, "empty data"
        return True, f"f66={data.get('f66')} f72={data.get('f72')} f78={data.get('f78')} f84={data.get('f84')}"
    except Exception as e:
        return False, str(e)


def summarize_items(body: str) -> str:
    try:
        j = json.loads(body)
    except json.JSONDecodeError:
        return body[:200]
    data = j.get("data", j)
    if isinstance(data, list):
        if not data:
            return "items=0 (empty)"
        last = data[-1]
        return f"items={len(data)} last_main_in_flow={last.get('main_in_flow', last.get('MainInFlow', '?'))} source_hint=GeeGooData/futu"
    if isinstance(data, dict):
        keys = ["capital_in_super", "capital_in_big", "CapitalInSuper", "update_time"]
        picked = {k: data.get(k) for k in keys if k in data}
        if picked:
            return str(picked)
        return str(data)[:200]
    return str(j)[:200]


def main() -> None:
    cfg = json.loads(DEPLOY.read_text(encoding="utf-8"))
    agent_ssh = cfg["targets"]["geegoo-agent"]["ssh"]
    bot_ssh = cfg["targets"]["geegoo-bot"]["ssh"]
    data_ssh = cfg["targets"].get("geegoo-data", {}).get("ssh") or cfg["targets"]["trading-data"]["ssh"]

    _, out, _ = ssh_run(
        agent_ssh,
        "python3 -c \"import json; c=json.load(open('/home/ubuntu/.geegoo/config.json')); "
        "print(c.get('mcp_token','')); print(c.get('geegoo_api_key',''))\"",
    )
    lines = [ln.strip() for ln in out.strip().splitlines() if ln.strip()]
    token = lines[0] if lines else ""
    bot_key = lines[1] if len(lines) > 1 else ""

    print(f"Agent host: {agent_ssh['host']}")
    print(f"GeeGooBot host: {bot_ssh['host']} :3120")
    print(f"GeeGooData host: {data_ssh['host']} :3300")
    print()

    for market, code in CODES:
        print("=" * 70)
        print(f"{market}  {code}")
        print("-" * 70)

        http, body = curl_on_host(
            bot_ssh,
            "http://127.0.0.1:3120/getCapitalFlow",
            {"mcp_token": token, "code": code, "period": "DAY"},
            bot_key,
        )
        ok = http == "200" and '"code":100' in body.replace(" ", "")
        items_summary = summarize_items(body)
        print(f"[1] GeeGooBot mcp-api POST /getCapitalFlow  HTTP {http}  {'OK' if ok else 'FAIL/SKIP'}")
        print(f"    {items_summary}")

        http2, body2 = curl_on_host(
            bot_ssh,
            "http://127.0.0.1:3120/getCapitalDistribution",
            {"mcp_token": token, "code": code},
            bot_key,
        )
        ok2 = http2 == "200" and '"code":100' in body2.replace(" ", "")
        print(f"[2] GeeGooBot mcp-api POST /getCapitalDistribution  HTTP {http2}  {'OK' if ok2 else 'FAIL/SKIP'}")
        print(f"    {summarize_items(body2)}")

        http3, body3 = curl_on_host(
            data_ssh,
            "http://127.0.0.1:3300/v1/market/capital/flow",
            {"code": code, "period": "DAY"},
        )
        print(f"[3] GeeGooData data-api POST /v1/market/capital/flow  HTTP {http3}")
        print(f"    {body3[:280]}")

        http4, body4 = curl_on_host(
            data_ssh,
            "http://127.0.0.1:3300/v1/market/capital/distribution",
            {"code": code},
        )
        print(f"[4] GeeGooData data-api POST /v1/market/capital/distribution  HTTP {http4}")
        print(f"    {body4[:280]}")

        if market == "A股":
            em_ok, em_msg = eastmoney_flow(code)
            print(f"[5] Agent东财回退 capital flow (push2.eastmoney.com)  {'OK' if em_ok else 'FAIL'}")
            print(f"    {em_msg}")
            em_ok2, em_msg2 = eastmoney_dist(code)
            print(f"[6] Agent东财回退 capital distribution  {'OK' if em_ok2 else 'FAIL'}")
            print(f"    {em_msg2}")
        print()

    # GeeGooData direct with service token (from bot .env on server)
    _, env_out, _ = ssh_run(
        bot_ssh,
        "grep GEEGOO_DATA_SERVICE_TOKEN /home/ubuntu/apps/GeeGooBot/.env 2>/dev/null | head -1",
    )
    data_token = env_out.strip().split("=", 1)[-1].strip() if "=" in env_out else ""
    if data_token:
        print("=" * 70)
        print("GeeGooData 直连（带 Bearer，验证 source 字段）")
        for market, code in CODES:
            http, body = curl_on_host(
                data_ssh,
                "http://127.0.0.1:3300/v1/market/capital/flow",
                {"code": code, "period": "DAY"},
                data_token,
            )
            print(f"{market} flow HTTP {http}: {body[:320]}")
            http2, body2 = curl_on_host(
                data_ssh,
                "http://127.0.0.1:3300/v1/market/capital/distribution",
                {"code": code},
                data_token,
            )
            print(f"{market} dist HTTP {http2}: {body2[:320]}")
            print()


if __name__ == "__main__":
    main()
