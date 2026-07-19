#!/usr/bin/env python3
"""Probe analyze-api batch translation on production with timing."""
from __future__ import annotations

import json
import re
import time
from pathlib import Path

import paramiko

DEPLOY = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")


def ssh_run(ssh_cfg: dict, cmd: str, timeout: int = 300) -> str:
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
    client.close()
    return out if out.strip() else err


def curl_timed(host_ssh: dict, url: str, auth: str, body: dict, timeout: int = 240) -> tuple[float, str, str]:
    payload = json.dumps(body, ensure_ascii=False).replace("'", "'\\''")
    cmd = (
        f"date +%s%3N; "
        f"curl -sS -m {timeout} -w '\\nHTTP:%{{http_code}}' "
        f"-H 'Authorization: Bearer {auth}' -H 'Content-Type: application/json' "
        f"-d '{payload}' {url}; "
        f"echo; date +%s%3N"
    )
    raw = ssh_run(host_ssh, cmd, timeout=timeout + 30)
    lines = [ln for ln in raw.splitlines() if ln.strip()]
    if len(lines) < 3:
        return 0.0, "?", raw
    start_ms = int(lines[0])
    end_ms = int(lines[-1])
    body_text = "\n".join(lines[1:-1])
    m = re.search(r"HTTP:(\d{3})", body_text)
    http = m.group(1) if m else "?"
    body_text = re.sub(r"\nHTTP:\d{3}$", "", body_text).strip()
    return (end_ms - start_ms) / 1000.0, http, body_text


def main() -> int:
    cfg = json.loads(DEPLOY.read_text(encoding="utf-8"))
    sig = cfg["targets"]["geegoo-signal"]["ssh"]

    analyze_key = ssh_run(sig, "grep '^GEEGOO_SIGNAL_ANALYZE_API_KEY=' /root/apps/GeeGooSignal/.env | cut -d= -f2-").strip()
    cat_key = ssh_run(sig, "grep '^GEEGOO_SIGNAL_CATALOG_API_KEY=' /root/apps/GeeGooSignal/.env | cut -d= -f2-").strip()
    head = ssh_run(sig, "cd /root/apps/GeeGooSignal && git log -1 --oneline").strip()
    print(f"GeeGooSignal HEAD: {head}")
    _, _, sig_body = curl_timed(sig, "http://127.0.0.1:3210/getIndexSignalForSkill", cat_key, {}, timeout=30)
    _, _, combo_body = curl_timed(sig, "http://127.0.0.1:3210/getSignalCombinationForSkill", cat_key, {}, timeout=30)
    signal_id = ""
    for raw in (combo_body, sig_body):
        try:
            data = json.loads(raw)
            items = data if isinstance(data, list) else data.get("data", [])
            for item in items or []:
                sid = str(item.get("signal_id") or item.get("id") or item.get("_id") or "")
                if sid:
                    signal_id = sid
                    break
            if signal_id:
                break
        except json.JSONDecodeError:
            continue
    if not signal_id:
        try:
            data = json.loads(sig_body)
            items = data if isinstance(data, list) else data.get("data", [])
            if items:
                first = items[0]
                signal_id = str(first.get("signal_id") or first.get("id") or first.get("_id") or "")
        except json.JSONDecodeError:
            pass
    print(f"signal_id: {signal_id or '(none)'}")

    cases = [
        ("grid_cn", "http://127.0.0.1:3230/generateGridStrategy", {
            "code": "00700.HK", "name": "腾讯控股", "months_back": 3, "language": "cn",
        }),
        ("grid_en", "http://127.0.0.1:3230/generateGridStrategy", {
            "code": "00700.HK", "name": "腾讯控股", "months_back": 3, "language": "en",
        }),
    ]
    if signal_id:
        cases.append(("dca_cn", "http://127.0.0.1:3230/generateDCAStrategy", {
            "code": "00700.HK", "name": "腾讯控股", "months_back": 3, "signal_id": signal_id, "language": "cn",
        }))

    print("\n=== analyze-api live probe ===")
    for name, url, body in cases:
        elapsed, http, text = curl_timed(sig, url, analyze_key, body, timeout=240)
        ok = http == "200" and ('"code":100' in text or '"code": 100' in text)
        preview = text.replace("\n", " ")[:120]
        print(f"{name:10} {'PASS' if ok else 'FAIL'}  {elapsed:6.1f}s  HTTP {http}  {preview}")
        if ok:
            try:
                payload = json.loads(text)
                data = payload.get("data", {})
                if name.startswith("grid"):
                    reason = data.get("reason")
                    print(f"           reason type={type(reason).__name__} sample={str(reason)[:80]}")
                else:
                    tc = data.get("trend_conclusion", {})
                    print(f"           trend reason type={type(tc.get('reason')).__name__}")
            except json.JSONDecodeError:
                pass
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
