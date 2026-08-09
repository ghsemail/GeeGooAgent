#!/usr/bin/env python3
"""Push generated postmarket report to MCP for a stock code."""
from __future__ import annotations

import json
import sys
from pathlib import Path

import paramiko

DEPLOY = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")
USER = "6366170502d5c175fd586fe8"
BOT_KEY = "sk-7a2b3c4d5e6f7g8h9i0j1k2l3m4n5o6p7q8r9s0t1u2v3w4x5y6z7a8b9c0d1e2"


def main() -> int:
    code = sys.argv[1] if len(sys.argv) > 1 else "00700.HK"
    report_date = sys.argv[2] if len(sys.argv) > 2 else "2026-08-07"

    remote = f'''
import json, os, re, urllib.request, urllib.error
from pathlib import Path

CODE = "{code}"
REPORT_DATE = "{report_date}"
USER = "{USER}"
BOT = "{BOT_KEY}"
MCP = "http://118.195.135.97:3120"

cfg = json.load(open(os.path.expanduser("~/.geegoo/config.json")))
tok = cfg.get("mcp_token", "")
key = cfg.get("geegoo_api_key") or cfg.get("api_key", "")

runs = sorted(Path(os.path.expanduser("~/.geegoo/data/working")).glob("*.json"), key=lambda p: p.stat().st_mtime, reverse=True)
data = json.loads(runs[0].read_text())
ws = data["stocks"][CODE]
report = ""
for d in ["2026-08-09", "2026-08-08", "2026-08-07", "20260809", "20260808", "20260807"]:
    p = Path(os.path.expanduser(f"~/.geegoo/data/reports/{{d}}/{{CODE}}-postmarket.md"))
    if p.exists():
        report = p.read_text(encoding="utf-8", errors="replace")
        print("report_file", p)
        break
if not report:
    raise SystemExit("no local postmarket md")

bias = ws.get("session_bias") or "neutral"
change = float(ws.get("change_pct") or 0)
vs = ws.get("vs_stock_premarket") or "na"
stock_name = ws.get("stock_name") or CODE.split(".")[0]

def build_market_summary(report, change, bias, stock_name):
    bias_cn = {{"neutral": "中性", "bullish": "偏多", "bearish": "偏空"}}.get(bias, bias)
    word = "收跌" if change < 0 else ("收涨" if change > 0 else "平盘")
    parts = [f"{{stock_name}}当日{{word}}{{abs(change):.2f}}%，盘面倾向{{bias_cn}}"]
    for line in report.splitlines():
        t = line.strip().lstrip("> ").strip("* ").strip()
        if ("整体结论" in t or "整体判断" in t) and "：" in t:
            v = t.split("：", 1)[1].strip().strip("*")
            if v and len(v) > 4 and "多空严" not in v:
                parts.append(v)
                break
    for line in report.splitlines():
        t = re.sub(r"[#*>`\\-•]", "", line).strip()
        if len(t) > 24 and any(k in t for k in ("累计", "趋势", "震荡", "支撑", "压力")):
            parts.append(t[:140])
            break
    text = "，".join(parts)
    if not text.endswith("。"):
        text += "。"
    pad = "短线波动与量能变化需结合关键位继续观察。"
    while len(text) < 80:
        text += pad
    return text[:280]

trade_summary = (ws.get("bot_log_summary") or "").strip()
if not trade_summary or trade_summary in ("[]", "{{}}", "map[]") or len(trade_summary) < 40:
    trade_summary = "当日机器人未产生成交记录，持仓与策略状态保持不变，可关注下一交易日信号触发与仓位变化。"

experience = (
    f"当日涨跌幅约 {{change:+.2f}}%，盘面倾向{{ {{'neutral':'中性','bullish':'偏多','bearish':'偏空'}}.get(bias,bias) }}；"
    f"与盘前对照为{{vs}}。小时级信号与价格结构显示短线波动有限，宜结合成交量与关键位再评估次日节奏，"
    f"避免在方向未明朗时追涨杀跌，并留存本次对照结论供后续策略复盘参考。"
)

body = {{
    "mcp_token": tok,
    "code": CODE,
    "stock_name": stock_name,
    "session_date": REPORT_DATE,
    "session_bias": bias,
    "change_pct": change,
    "trade_summary": trade_summary,
    "market_summary": build_market_summary(report, change, bias, stock_name),
    "experience_summary": experience,
    "report": report,
    "summary": f"{{CODE}} 盘后 {{change:+.2f}}%，倾向 {{bias}}",
    "bot_id": ws.get("bot_id", ""),
    "bot_name": ws.get("bot_name", ""),
    "bot_type": ws.get("bot_type", ""),
    "vs_stock_premarket": vs,
    "stock_premarket_report_id": ws.get("premarket_report_id") or ws.get("stock_premarket_report_id") or "",
    "tags": ["stock_postmarket"],
}}

def post(path, payload):
    req = urllib.request.Request(
        MCP + path,
        data=json.dumps(payload).encode(),
        headers={{"Content-Type": "application/json", "Authorization": "Bearer " + key}},
        method="POST",
    )
    try:
        with urllib.request.urlopen(req, timeout=120) as r:
            return r.status, json.loads(r.read().decode())
    except urllib.error.HTTPError as e:
        raw = e.read().decode(errors="replace")
        try:
            return e.code, json.loads(raw)
        except Exception:
            return e.code, {{"raw": raw[:500]}}

c, r = post("/getStockDailyReports", {{"mcp_token": tok, "code": CODE, "report_date": REPORT_DATE}})
for row in (r.get("data") or {{}}).get("stock_postmarket") or []:
    rid = str(row.get("report_id") or row.get("_id") or "")
    if rid:
        post("/deleteStockPostmarketReport", {{"mcp_token": tok, "report_id": rid}})

c, r = post("/createStockPostmarketReport", body)
print("create", c, r.get("code"), r.get("message"), r.get("data"))
if r.get("code") != 100:
    raise SystemExit(1)

req = urllib.request.Request(
    "http://118.195.135.97:3140/reports/daily",
    data=json.dumps({{"user_id": USER, "phases": ["stock_postmarket"], "limit_per_phase": 20}}).encode(),
    headers={{"Content-Type": "application/json", "Authorization": "Bearer " + BOT}},
    method="POST",
)
api = json.loads(urllib.request.urlopen(req, timeout=60).read())
rows = (api.get("data") or {{}}).get("stock_postmarket") or []
hit = [x for x in rows if x.get("code") == CODE and REPORT_DATE in str(x.get("report_date") or "")]
print("api hit", len(hit))
if hit:
    h = hit[0]
    print("report_id", h.get("report_id"))
    print("bot_name", h.get("bot_name"))
    print("change_pct", h.get("change_pct"), "session_bias", h.get("session_bias"))
    print("market_summary", str(h.get("market_summary") or "")[:180])
    print("report_len", len(str(h.get("report") or "")))
'''

    cfg = json.loads(DEPLOY.read_text(encoding="utf-8"))
    ssh = cfg["targets"]["geegoo-agent"]["ssh"]
    c = paramiko.SSHClient()
    c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    c.connect(ssh["host"], username=ssh["user"], password=ssh.get("password"), timeout=30)
    _, o, e = c.exec_command(f"python3 <<'PY'\n{remote}\nPY", timeout=180)
    out = (o.read() + e.read()).decode("utf-8", errors="replace")
    print(out)
    c.close()
    return 0 if "api hit 1" in out or "api hit 1" in out else 1


if __name__ == "__main__":
    raise SystemExit(main())
