#!/usr/bin/env python3
from __future__ import annotations
import json
from pathlib import Path
import paramiko

DEPLOY = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")
cfg = json.loads(DEPLOY.read_text(encoding="utf-8"))
ssh = cfg["targets"]["geegoo-signal"]["ssh"]
c = paramiko.SSHClient()
c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
c.connect(ssh["host"], username=ssh["user"], password=ssh["password"], timeout=60)

ana = "aac157767ebdc8889b83b268852cc8ac09f4f360b67b36d7"
cat = "850367bc6d5fe8a4a53f267f5c308ac6d2ec1474d1764fe9"
sig = "a76e2d4b4aa8b8eb154f3f2e8feff49a2c34c0f94d012402"

cmd = f'''
ANA="{ana}"; CAT="{cat}"; SIG="{sig}";
probe() {{ local port=$1 key=$2 p=$3; code=$(curl -s -o /tmp/o -w "%{{http_code}}" -X POST http://127.0.0.1:$port/$p -H "Authorization: Bearer $key" -H "Content-Type: application/json" -d "{{}}"); echo "$port/$p $code"; }}
for p in getSingleAnalysis getSingleAnalysisHistory deleteSingleAnalysis getTechnicalAnalysis getFundamentalAnalysis getTechnicalAnalysisHistory getFundamentalAnalysisHistory DeleteTechnicalCache DeleteFundamentalCache; do probe 3230 "$ANA" "$p"; done
for p in getIndexSignal getCustomSignal getSignalCombination getAllIndexSignalId getAISignal queryVersion getVersion getModel getNewsRefreshLogs getStrategyGenerationLogs getLLMResult loopBackStrategy searchCode checkBackendServices getSinglePromptTemplate getAttitudePromptList; do probe 3210 "$CAT" "$p"; done
for p in getDashboardSignal getDashboardKline getSupportingPrice searchCode; do probe 3200 "$SIG" "$p"; done
probe 3300 "" getStockNews 2>/dev/null || true
curl -s -o /dev/null -w "47.80.14.120:3300/health %{{http_code}}\\n" http://47.80.14.120:3300/health
curl -s -o /dev/null -w "82.157.97.76:3300/health %{{http_code}}\\n" http://82.157.97.76:3300/health
'''
_, o, e = c.exec_command(cmd, timeout=120)
raw = (o.read() + e.read()).decode("utf-8", errors="replace")
Path(r"D:\Geegoo\GeeGooAgent\scripts\probe_paths_signal.txt").write_text(raw, encoding="utf-8")
print(raw.encode("ascii", errors="replace").decode())
c.close()
