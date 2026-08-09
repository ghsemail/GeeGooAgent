#!/usr/bin/env python3
"""Test A-share market data via local Futu OpenD (same helper as GeeGooData)."""
from __future__ import annotations

import json
import subprocess
import sys
import urllib.request
from pathlib import Path

HELPER = Path(__file__).resolve().parents[2] / "GeeGooData" / "scripts" / "futu_market_helper.py"
CODES = ["600519.SH", "000001.SZ"]


def call_futu(operation: str, code: str, **extra) -> dict:
    req = {"operation": operation, "code": code, "futu_host": "127.0.0.1", "futu_port": 11111, **extra}
    proc = subprocess.run(
        [sys.executable, str(HELPER)],
        input=json.dumps(req),
        capture_output=True,
        text=True,
        timeout=120,
        encoding="utf-8",
    )
    if proc.returncode != 0 and not proc.stdout.strip():
        return {"ok": False, "error": proc.stderr.strip() or f"exit {proc.returncode}"}
    try:
        return json.loads(proc.stdout.strip().splitlines()[-1])
    except json.JSONDecodeError:
        return {"ok": False, "error": proc.stdout[:500], "stderr": proc.stderr[:500]}


def eastmoney_flow(code: str) -> str:
    secid = "1." + code.replace(".SH", "") if code.endswith(".SH") else "0." + code.replace(".SZ", "")
    url = (
        "https://push2.eastmoney.com/api/qt/stock/fflow/kline/get"
        f"?lmt=1&klt=101&secid={secid}&fields1=f1,f2,f3,f7&fields2=f51,f52,f53,f54,f55,f56,f57,f58,f59,f60,f61,f62,f63,f64,f65"
    )
    req = urllib.request.Request(url, headers={"User-Agent": "Mozilla/5.0"})
    with urllib.request.urlopen(req, timeout=20) as resp:
        d = json.loads(resp.read().decode())
    kl = (d.get("data") or {}).get("klines") or []
    return kl[-1] if kl else "EMPTY"


def main() -> int:
    print(f"Helper: {HELPER}")
    print(f"Exists: {HELPER.is_file()}\n")

    for code in CODES:
        print("=" * 60)
        print(code)
        for op, extra in [
            ("quote", {}),
            ("klines", {"frequency": "daily", "months_back": 1}),
            ("capital_flow", {"period": "DAY"}),
            ("capital_distribution", {}),
        ]:
            out = call_futu(op, code, **extra)
            ok = out.get("ok")
            print(f"  [{op}] {'OK' if ok else 'FAIL'}")
            if not ok:
                print(f"    error: {out.get('error', out)[:200]}")
                continue
            if op == "quote":
                print(f"    price={out.get('price')}")
            elif op == "klines":
                bars = out.get("bars") or []
                print(f"    bars={len(bars)} last={bars[-1] if bars else None}")
            elif op == "capital_flow":
                items = out.get("capital_flow") or []
                print(f"    items={len(items)} last={items[-1] if items else None}")
            elif op == "capital_distribution":
                dist = out.get("capital_distribution") or {}
                print(f"    {dist}")
        try:
            print(f"  [eastmoney flow fallback] {eastmoney_flow(code)}")
        except Exception as e:
            print(f"  [eastmoney flow fallback] ERR {e}")
        print()
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
