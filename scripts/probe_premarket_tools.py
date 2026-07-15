#!/usr/bin/env python3
"""Probe pre_market MVP tools and their upstream endpoints."""
from __future__ import annotations

import json
import re
from pathlib import Path

import paramiko

DEPLOY = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")


def ssh_run(ssh_cfg: dict, cmd: str, timeout: int = 120) -> tuple[int, str, str]:
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


def http_code(text: str) -> str:
    m = re.search(r"HTTP:(\d{3})", text)
    return m.group(1) if m else "?"


def classify(http: str, body: str) -> str:
    if http == "200":
        if '"code":100' in body or '"code": 100' in body:
            return "OK"
        if body.strip().startswith("[") or '"price"' in body:
            return "OK"
        if '"status":"ok"' in body:
            return "OK"
        return "OK?"
    if http == "404":
        return "NOT_IMPL"
    if http in ("401", "403"):
        return "AUTH"
    if http == "502" or http == "500":
        return "UPSTREAM_ERR"
    if http == "400":
        return "BAD_REQ"
    if http == "000" or http == "?":
        return "TIMEOUT"
    return f"HTTP_{http}"


def mcp_curl(auth: str, body: dict) -> tuple[str, str]:
    path = body.pop("_path")
    payload = json.dumps(body)
    cmd = (
        f"curl -sS -m 45 -w '\\nHTTP:%{{http_code}}' "
        f"-H 'Authorization: Bearer {auth}' -H 'Content-Type: application/json' "
        f"-d '{payload}' http://127.0.0.1:3120/{path}"
    )
    code, out, err = ssh_run(bot_ssh, cmd, timeout=60)
    if code != 0 and not out.strip():
        out = err
    return http_code(out), out


def signal_curl(auth: str, path: str, body: dict) -> tuple[str, str]:
    payload = json.dumps(body)
    cmd = (
        f"curl -sS -m 30 -w '\\nHTTP:%{{http_code}}' "
        f"-H 'Authorization: Bearer {auth}' -H 'Content-Type: application/json' "
        f"-d '{payload}' http://127.0.0.1:3200/{path}"
    )
    code, out, err = ssh_run(sig_ssh, cmd, timeout=45)
    if code != 0 and not out.strip():
        out = err
    return http_code(out), out


def main() -> int:
    global bot_ssh, sig_ssh
    cfg = json.loads(DEPLOY.read_text(encoding="utf-8"))
    agent_ssh = cfg["targets"]["geegoo-agent"]["ssh"]
    bot_ssh = cfg["targets"]["geegoo-bot"]["ssh"]
    sig_ssh = cfg["targets"]["geegoo-signal"]["ssh"]

    code, out, _ = ssh_run(
        agent_ssh,
        "python3 -c \"import json; c=json.load(open('/home/ubuntu/.geegoo/config.json')); "
        "print(c.get('mcp_token','')); print(c.get('geegoo_api_key','')); "
        "print(c.get('signal_api_key') or c.get('signal_catalog_api_key') or ''); "
        "print(c.get('signal_api_url') or 'http://146.56.225.252:3200')\"",
    )
    lines = [ln.strip() for ln in out.strip().splitlines() if ln.strip()]
    token = lines[0] if lines else ""
    geegoo_key = lines[1] if len(lines) > 1 else ""
    sig_key = lines[2] if len(lines) > 2 else ""

    print("=" * 72)
    print("geegoo doctor")
    print("=" * 72)
    _, doc, _ = ssh_run(agent_ssh, "export PATH=$HOME/.geegoo/bin:$PATH; geegoo doctor 2>&1", timeout=90)
    print(doc.strip())
    doctor_ok = "全部检查通过" in doc

    tests: list[tuple[str, str, str, str]] = []

    mcp_cases = [
        ("check_trading_day", "/checkTradingDay", {"mcp_token": token, "code": "00700.HK"}),
        ("get_report_bot_codes", "/getReportBotCodes", {"mcp_token": token}),
        ("get_single_prompt_template", "/getSinglePromptTemplate", {"mcp_token": token, "type": "tech", "period": "daily"}),
        ("get_current_price", "/getCurrentPrice", {"mcp_token": token, "code": "00700.HK"}),
        ("get_capital_flow", "/getCapitalFlow", {"mcp_token": token, "code": "00700.HK", "period": "DAY"}),
        ("get_capital_distribution", "/getCapitalDistribution", {"mcp_token": token, "code": "00700.HK"}),
        ("get_mcp_analysis", "/getMCPAnalysis", {
            "mcp_token": token, "name": "腾讯控股", "code": "00700.HK",
            "prompt_id": "69ec7035b9ccd3d9befc6c23", "period": "hourly", "language": "cn",
        }),
        ("get_bot_yesterday_attitude", "/getBotYesterdayAttitude", {"mcp_token": token, "code": "00700.HK"}),
        ("list_today_reports", "/getPreMarketReports", {"mcp_token": token, "code": "00700.HK", "period": "daily"}),
        ("get_stock_daily_reports", "/getStockDailyReports", {"mcp_token": token, "code": "00700.HK"}),
        ("create_pre_market_report", "/createPreMarketReport", {"mcp_token": token, "code": "00700.HK", "period": "daily", "content": "probe"}),
        ("search_code (mcp)", "/searchCode", {"regex": "腾讯"}),
    ]

    print("\n" + "=" * 72)
    print("MCP endpoints (GeeGooBot :3120)")
    print("=" * 72)
    for tool, path, body in mcp_cases:
        b = dict(body)
        b["_path"] = path.lstrip("/")
        http, raw = mcp_curl(geegoo_key, b)
        status = classify(http, raw)
        tests.append((tool, "mcp-api", status, http))
        preview = raw.replace("\n", " ")[:120]
        print(f"  {tool:28} {status:12} HTTP {http}  {preview}")

    print("\n" + "=" * 72)
    print("Signal API (GeeGooSignal :3200)")
    print("=" * 72)
    http, raw = signal_curl(sig_key, "searchCode", {"regex": "腾讯", "market": ["HK"]})
    status = classify(http, raw)
    tests.append(("search_code", "signal-api", status, http))
    print(f"  search_code                  {status:12} HTTP {http}  {raw.replace(chr(10), ' ')[:120]}")

    local_tools = [
        ("fetch_market_news", "OK", "newsrunner Go fallback (see go test on agent)"),
        ("fetch_stock_news", "OK", "newsrunner Go fallback"),
        ("recall_yesterday_summary", "OK", "reads workspace reports or MCP fallback; skip if no yesterday file"),
        ("read_working_state", "LOCAL", "in-process memory"),
        ("save_local_report", "LOCAL", "writes workspace file"),
        ("send_feishu_summary", "SKIP", "needs feishu_webhook_url in config"),
        ("write_execution_log", "LOCAL", "appends local log"),
    ]
    print("\n" + "=" * 72)
    print("Local / stub tools (no remote HTTP)")
    print("=" * 72)
    for name, status, note in local_tools:
        tests.append((name, "local", status, note))
        print(f"  {name:28} {status:12}  {note}")

    ok = sum(1 for _, _, s, _ in tests if s == "OK")
    not_impl = [t for t, _, s, _ in tests if s == "NOT_IMPL"]
    err = [t for t, _, s, _ in tests if s in ("UPSTREAM_ERR", "TIMEOUT", "AUTH", "HTTP_") or s.startswith("HTTP_") and s not in ("HTTP_200",)]

    print("\n" + "=" * 72)
    print("SUMMARY")
    print("=" * 72)
    print(f"  doctor: {'PASS' if doctor_ok else 'FAIL'}")
    print(f"  remote OK: {ok}/{len([t for t in tests if t[1] != 'local'])}")
    if not_impl:
        print(f"  NOT_IMPL ({len(not_impl)}): {', '.join(not_impl)}")
    bad = [t for t, _, s, _ in tests if s in ("UPSTREAM_ERR", "TIMEOUT", "AUTH") or (s.startswith("HTTP_") and s not in ("HTTP_200",))]
    if bad:
        print(f"  ERRORS ({len(bad)}): {', '.join(bad)}")

    return 0 if doctor_ok and not bad else 1


if __name__ == "__main__":
    raise SystemExit(main())
