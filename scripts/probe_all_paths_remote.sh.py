#!/usr/bin/env python3
"""Probe all bot-api endpoint paths exist (localhost on each server)."""
from __future__ import annotations

import json
from pathlib import Path

import paramiko

DEPLOY = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")

BOT_PATHS = "login register getUserStock getUserInfo getNewsStock getCurrentPrice getUserBot createBot editBot deleteBot getDCABotLog getGRIDBotLog getHDGBotLog getBindingBot getPosition getAttitudeLog getSmartTradeLog getSmartReminderLog checkTradeConnection queryTradingDate getUserList getKeyList".split()

AGENT_PATHS = "getUserAgents getAgentSessionMessages agentChat addUserAgent updateUserAgent deleteUserAgent setActiveUserAgent".split()

ANALYZE_PATHS = "getSingleAnalysis getSingleAnalysisHistory deleteSingleAnalysis getTechnicalAnalysis getFundamentalAnalysis getTechnicalAnalysisHistory getFundamentalAnalysisHistory DeleteTechnicalCache DeleteFundamentalCache".split()

CATALOG_EXTRA = "getIndexSignal getCustomSignal getSignalCombination getAllIndexSignalId getAISignal queryVersion getVersion getModel getNewsRefreshLogs getStrategyGenerationLogs getLLMResult loopBackStrategy searchCode checkBackendServices".split()


def curl_loop(target: str, port: int, key: str, paths: list[str]) -> None:
    cfg = json.loads(DEPLOY.read_text(encoding="utf-8"))
    ssh = cfg["targets"][target]["ssh"]
    c = paramiko.SSHClient()
    c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    c.connect(ssh["host"], username=ssh["user"], password=ssh["password"], timeout=60)
    joined = " ".join(paths)
    cmd = (
        f'KEY="{key}"; for p in {joined}; do '
        f'code=$(curl -s -o /tmp/out -w "%{{http_code}}" -X POST http://127.0.0.1:{port}/$p '
        f'-H "Content-Type: application/json" -H "Authorization: Bearer $KEY" -d "{{}}"); '
        f'echo "$p $code $(head -c 60 /tmp/out)"; done'
    )
    _, o, e = c.exec_command(cmd, timeout=300)
    print((o.read() + e.read()).decode("utf-8", errors="replace"))
    c.close()


def main() -> int:
    bot = "sk-7a2b3c4d5e6f7g8h9i0j1k2l3m4n5o6p7q8r9s0t1u2v3w4x5y6z7a8b9c0d1e2"
    cat = "850367bc6d5fe8a4a53f267f5c308ac6d2ec1474d1764fe9"
    sig = "a76e2d4b4aa8b8eb154f3f2e8feff49a2c34c0f94d012402"
    ana = "aac157767ebdc8889b83b268852cc8ac09f4f360b67b36d7"

    print("=== GeeGooBot app-api :3100 ===")
    curl_loop("geegoo-bot", 3100, bot, BOT_PATHS)

    print("=== GeeGooBot agent-api :3110 ===")
    curl_loop("geegoo-bot", 3110, bot, AGENT_PATHS)

    print("=== GeeGooSignal analyze-api :3230 ===")
    curl_loop("geegoo-signal", 3230, ana, ANALYZE_PATHS)

    print("=== GeeGooSignal catalog-api :3210 ===")
    curl_loop("geegoo-signal", 3210, cat, CATALOG_EXTRA)

    print("=== GeeGooSignal signal-api :3200 ===")
    curl_loop("geegoo-signal", 3200, sig, "getDashboardSignal getDashboardKline getSupportingPrice searchCode".split())
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
