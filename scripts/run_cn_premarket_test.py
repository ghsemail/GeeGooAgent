#!/usr/bin/env python3
"""Trigger A-share (CN) premarket_market skill on agent server and fetch report."""
from __future__ import annotations

import json
import sys
from pathlib import Path

import paramiko

DEPLOY = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")


def main() -> int:
    cfg = json.loads(DEPLOY.read_text(encoding="utf-8"))
    ssh = cfg["targets"]["geegoo-agent"]["ssh"]
    client = paramiko.SSHClient()
    client.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    client.connect(
        hostname=ssh["host"],
        port=ssh.get("port", 22),
        username=ssh["user"],
        password=ssh.get("password"),
        timeout=30,
    )

    remote_py = r'''
import json, os, subprocess, urllib.request
from datetime import date

today = date.today().isoformat()
cfg_path = os.path.expanduser("~/.geegoo/config.json")
with open(cfg_path, encoding="utf-8") as f:
    cfg = json.load(f)
mcp_token = cfg.get("mcp_token", "")

# Weekend / CN data issues: use dry_run so trading-day gate passes and tools stub.
test_cfg_path = "/tmp/geegoo-cn-premarket-test.json"
test_cfg = dict(cfg)
test_cfg["dry_run"] = True
with open(test_cfg_path, "w", encoding="utf-8") as f:
    json.dump(test_cfg, f)

def post(url, body):
    req = urllib.request.Request(url, data=json.dumps(body).encode(), headers={"Content-Type": "application/json"})
    with urllib.request.urlopen(req, timeout=30) as r:
        return json.loads(r.read().decode())

out = {"today": today, "dry_run": True}
try:
    td = post("http://118.195.135.97:3120/checkTradingDay", {"mcp_token": mcp_token, "code": "000001.SZ"})
    out["trading_day"] = td
except Exception as e:
    out["trading_day_error"] = str(e)

run = subprocess.run(
    [
        os.path.expanduser("~/.geegoo/bin/geegoo"), "run",
        "--config", test_cfg_path, "--market", "CN", "stock_premarket",
    ],
    capture_output=True, text=True, timeout=600,
)
out["run_exit"] = run.returncode
out["run_stdout"] = run.stdout[-4000:]
out["run_stderr"] = run.stderr[-2000:]

report_dir = os.path.expanduser(f"~/.geegoo/data/reports/{today}")
out["report_dir_exists"] = os.path.isdir(report_dir)
paths = []
if os.path.isdir(report_dir):
    for name in os.listdir(report_dir):
        if "market-CN" in name or "market-cn" in name.lower():
            paths.append(os.path.join(report_dir, name))
out["report_paths"] = paths
for p in paths:
    try:
        with open(p, encoding="utf-8") as f:
            out["report_content"] = f.read()
    except OSError as e:
        out["report_read_error"] = str(e)

if not paths:
    try:
        mkt = post("http://118.195.135.97:3120/getMarketPremarketReport", {
            "mcp_token": mcp_token, "market": "CN", "report_date": today,
        })
        out["mcp_report"] = mkt
    except Exception as e:
        out["mcp_report_error"] = str(e)

print(json.dumps(out, ensure_ascii=False))
'''
    _, stdout, stderr = client.exec_command(f"python3 <<'PY'\n{remote_py}\nPY", timeout=620)
    raw = stdout.read().decode("utf-8", errors="replace").strip()
    err = stderr.read().decode("utf-8", errors="replace").strip()
    client.close()

    if err and not raw:
        print("REMOTE STDERR:", err, file=sys.stderr)
        return 1

    # remote may print extra lines; take last JSON object line
    data = None
    for line in reversed(raw.splitlines()):
        line = line.strip()
        if line.startswith("{"):
            data = json.loads(line)
            break
    if data is None:
        print(raw)
        return 1

    print("=== A股交易日检查 ===")
    td = data.get("trading_day") or {}
    print(json.dumps(td, ensure_ascii=False, indent=2))

    print("\n=== Workflow 执行 ===")
    print(f"exit={data.get('run_exit')}")
    if data.get("run_stdout"):
        print(data["run_stdout"])
    if data.get("run_stderr"):
        print("stderr:", data["run_stderr"])

    content = data.get("report_content")
    if content:
        print("\n=== 本地报告文件 ===")
        for p in data.get("report_paths", []):
            print(p)
        print("\n" + "=" * 60)
        print(content)
        print("=" * 60)
        return 0

    mcp = data.get("mcp_report")
    if mcp:
        print("\n=== MCP 市场盘前报告 ===")
        d = mcp.get("data") or mcp
        print(f"result={d.get('result')} confidence={d.get('confidence')}")
        print(f"summary={d.get('summary')}")
        report = d.get("report") or ""
        print("\n" + "=" * 60)
        print(report[:8000])
        if len(report) > 8000:
            print("\n... (truncated)")
        print("=" * 60)
        return 0

    print("\n未生成报告。")
    if data.get("mcp_report_error"):
        print("MCP:", data["mcp_report_error"])
    return 2


if __name__ == "__main__":
    raise SystemExit(main())
