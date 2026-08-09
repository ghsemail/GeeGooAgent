#!/usr/bin/env python3
"""Probe remaining warning tools on production."""
from __future__ import annotations

import json
import re
from pathlib import Path

import paramiko

DEPLOY = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")


def ssh_run(ssh_cfg: dict, cmd: str, timeout: int = 120) -> str:
    c = paramiko.SSHClient()
    c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    c.connect(ssh_cfg["host"], username=ssh_cfg["user"], password=ssh_cfg.get("password"), timeout=30)
    _, out, err = c.exec_command(cmd, timeout=timeout)
    text = out.read().decode("utf-8", "replace")
    if not text.strip() and err.read():
        text = err.read().decode("utf-8", "replace")
    c.close()
    return text


def curl_json(host_ssh: dict, url: str, auth: str, body: dict, timeout: int = 45) -> tuple[str, str]:
    payload = json.dumps(body, ensure_ascii=False).replace("'", "'\\''")
    cmd = (
        f"curl -sS -m {timeout} -w '\\nHTTP:%{{http_code}}' "
        f"-H 'Authorization: Bearer {auth}' -H 'Content-Type: application/json' "
        f"-d '{payload}' {url}"
    )
    raw = ssh_run(host_ssh, cmd, timeout=timeout + 15)
    m = re.search(r"HTTP:(\d{3})", raw)
    http = m.group(1) if m else "?"
    body_text = re.sub(r"\nHTTP:\d{3}$", "", raw).strip()
    return http, body_text


def ok(http: str, body: str) -> bool:
    if http != "200":
        return False
    if '"code":100' in body or '"code": 100' in body:
        return True
    if body.startswith("[") and len(body) > 2:
        return True
    if '"price"' in body or '"analysis_result"' in body or '"finalValue"' in body:
        return True
    return False


def main() -> int:
    cfg = json.loads(DEPLOY.read_text(encoding="utf-8"))
    agent = cfg["targets"]["geegoo-agent"]["ssh"]
    bot = cfg["targets"]["geegoo-bot"]["ssh"]
    sig = cfg["targets"]["geegoo-signal"]["ssh"]

    creds = ssh_run(
        agent,
        "python3 -c \"import json; c=json.load(open('/home/ubuntu/.geegoo/config.json')); "
        "print(c.get('mcp_token','')); print(c.get('geegoo_api_key','')); "
        "print(c.get('signal_api_key') or c.get('signal_catalog_api_key') or c.get('geegoo_api_key','')); "
        "print(c.get('signal_analyze_api_key') or c.get('signal_catalog_api_key') or '')\"",
    ).splitlines()
    token = creds[0].strip() if creds else ""
    bot_key = creds[1].strip() if len(creds) > 1 else ""
    sig_key = creds[2].strip() if len(creds) > 2 else ""
    analyze_key = creds[3].strip() if len(creds) > 3 else sig_key

    results: list[tuple[str, bool, str]] = []

    cases = [
        ("get_ticker", bot, "http://127.0.0.1:3120/getTicker", bot_key, {"mcp_token": token, "code": "00700.HK", "num": 3}),
        ("get_broker", bot, "http://127.0.0.1:3120/getBroker", bot_key, {"mcp_token": token, "code": "00700.HK"}),
        ("get_position", bot, "http://127.0.0.1:3120/getPosition", bot_key, {"mcp_token": token, "code": "00700.HK"}),
        ("search_code@mcp", bot, "http://127.0.0.1:3120/searchCode", bot_key, {"regex": "腾讯"}),
        ("generate_grid@analyze", sig, "http://127.0.0.1:3230/generateGridStrategy", analyze_key, {
            "mcp_token": token, "code": "00700.HK", "name": "腾讯控股", "frequency": "daily", "fund": 100000,
        }),
        ("generate_dca@analyze", sig, "http://127.0.0.1:3230/generateDCAStrategy", analyze_key, {
            "mcp_token": token, "code": "00700.HK", "name": "腾讯控股", "frequency": "daily", "fund": 100000,
        }),
        ("loopback@signal", sig, "http://127.0.0.1:3200/loopBackStrategy", sig_key, {
            "type": "grid", "code": "00700.HK", "frequency": "daily", "fund": 100000, "months_back": 3,
            "grid_param": {"upper_limit_price": 520, "lower_limit_price": 420, "grid_num": 5},
        }),
        ("get_capital_flow", bot, "http://127.0.0.1:3120/getCapitalFlow", bot_key, {
            "mcp_token": token, "code": "00700.HK", "period": "DAY",
        }),
    ]

    print("Remaining tools probe")
    for name, host, url, key, body in cases:
        http, text = curl_json(host, url, key, body)
        passed = ok(http, text)
        preview = text.replace("\n", " ")[:100]
        results.append((name, passed, f"HTTP {http} {preview}"))
        mark = "PASS" if passed else "FAIL"
        print(f"  {name:24} {mark:4} HTTP {http}  {preview}")

    # Agent-local: news via geegoo dry-run is heavy; test Go fallback import path
    news_cmd = (
        "cd /home/ubuntu/.geegoo/geegoo-agent && "
        "go test ./internal/tools/newsrunner/... -count=1 2>&1 | tail -3"
    )
    news_out = ssh_run(agent, news_cmd, timeout=120)
    news_ok = "ok" in news_out
    results.append(("newsrunner tests", news_ok, news_out.strip()))
    print(f"  {'newsrunner tests':24} {'PASS' if news_ok else 'FAIL':4}  {news_out.strip()}")

    passed = sum(1 for _, p, _ in results if p)
    print(f"\n{passed}/{len(results)} passed")
    return 0 if passed == len(results) else 1


if __name__ == "__main__":
    raise SystemExit(main())
