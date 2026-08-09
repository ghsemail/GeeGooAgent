#!/usr/bin/env python3
"""Probe all trading_app bot-api endpoints from bot host localhost."""
from __future__ import annotations

import json
from pathlib import Path

import paramiko

DEPLOY = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")

BOT_ENDPOINTS = [
    "login", "register", "getUserStock", "getUserStockTrend", "addUserStock",
    "deleteUserStock", "getUserReminder", "getUserBot", "getUserHDGBot",
    "switchBot", "deleteBot", "createBot", "getDCABotLog", "getDCABotProfit",
    "getGRIDBotLog", "getGRIDBotProfit", "editBot", "getDCAReminderLog",
    "getGRIDReminderLog", "getCurrentPrice", "getUserInfo", "changePwd",
    "setNotice", "usedCheck", "getNewsStock", "addNewsStock", "deleteNewsStock",
    "switchTradeEnv", "setHost", "checkTradeConnection", "checkUnlockPwd",
    "getHDGBotLog", "getHDGBotProfit", "hdgCount", "getBindingBot",
    "getBotTPSLStatus", "getAttitudeLog", "setFcmToken", "setMcpToken",
    "getMcpToken", "getSmartTradeLog", "getSmartTradeProfit", "getPosition",
    "getSmartReminderLog",
]

AGENT_ENDPOINTS = [
    "getUserAgents", "addUserAgent", "updateUserAgent", "deleteUserAgent",
    "setActiveUserAgent", "agentChat", "getAgentSessionMessages",
]

ANALYZE_ENDPOINTS = [
    "getTechnicalAnalysisHistory", "getFundamentalAnalysisHistory",
    "DeleteTechnicalCache", "DeleteFundamentalCache",
    "getTechnicalAnalysis", "getFundamentalAnalysis",
    "getSingleAnalysis", "getSingleAnalysisHistory", "deleteSingleAnalysis",
]

PROMPT_ENDPOINTS = [
    "getSinglePromptTemplate", "getAttitudePromptList",
    "createCompetitorPromptTemplate", "editCompetitorPromptTemplate",
    "deleteCompetitorPromptTemplate", "createEtfPromptTemplate",
    "editEtfPromptTemplate", "deleteEtfPromptTemplate",
]

PY = r'''
import json, urllib.request, sys
KEY = sys.argv[1]
BASE = sys.argv[2]
paths = json.loads(sys.argv[3])
body = json.dumps({}).encode()
for p in paths:
    req = urllib.request.Request(f"{BASE}/{p}", data=body, method="POST",
        headers={"Content-Type":"application/json","Authorization":f"Bearer {KEY}"})
    try:
        with urllib.request.urlopen(req, timeout=20) as r:
            b = r.read()[:80]
            print(f"OK {p} http={r.status} {b!r}")
    except urllib.error.HTTPError as e:
        print(f"HTTP {p} http={e.code} {e.read()[:80]!r}")
    except Exception as e:
        print(f"ERR {p} {e}")
'''

def run_on(host_target: str, base: str, key: str, paths: list[str]) -> str:
    cfg = json.loads(DEPLOY.read_text(encoding="utf-8"))
    ssh = cfg["targets"][host_target]["ssh"]
    c = paramiko.SSHClient()
    c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    c.connect(ssh["host"], username=ssh["user"], password=ssh["password"], timeout=60)
    paths_json = json.dumps(paths)
    cmd = f"python3 -c {json.dumps(PY)} {json.dumps(key)} {json.dumps(base)} {json.dumps(paths_json)}"
    _, o, e = c.exec_command(cmd, timeout=300)
    out = (o.read() + e.read()).decode()
    c.close()
    return out


def main() -> int:
    bot_key = "sk-7a2b3c4d5e6f7g8h9i0j1k2l3m4n5o6p7q8r9s0t1u2v3w4x5y6z7a8b9c0d1e2"
    catalog_key = "850367bc6d5fe8a4a53f267f5c308ac6d2ec1474d1764fe9"
    analyze_key = "aac157767ebdc8889b83b268852cc8ac09f4f360b67b36d7"

    print("=== bot-api localhost paths ===")
    print(run_on("geegoo-bot", "http://127.0.0.1:3100", bot_key, BOT_ENDPOINTS))

    print("=== agent-api localhost paths ===")
    print(run_on("geegoo-bot", "http://127.0.0.1:3110", bot_key, AGENT_ENDPOINTS))

    print("=== analyze-api localhost paths ===")
    print(run_on("geegoo-signal", "http://127.0.0.1:3230", analyze_key, ANALYZE_ENDPOINTS))

    print("=== catalog prompt paths ===")
    print(run_on("geegoo-signal", "http://127.0.0.1:3210", catalog_key, PROMPT_ENDPOINTS))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
