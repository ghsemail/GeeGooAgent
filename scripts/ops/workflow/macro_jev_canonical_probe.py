#!/usr/bin/env python3
"""查当日 canonical 标的 JEV 与指定 batch 的 PUT 拒绝原因。"""
from __future__ import annotations

import json
import re
import sys
import urllib.request
from pathlib import Path

import paramiko

SKILL = Path.home() / ".cursor" / "skills" / "remote-deploy" / "scripts"
sys.path.insert(0, str(SKILL))
from credentials import resolve_ssh_cfg  # noqa: E402

DEPLOY = Path.home() / ".cursor" / "skills" / "remote-deploy" / "deploy.json"
MACRO = "http://146.56.225.252:3220"


def macro_key() -> str:
    cfg = json.loads(DEPLOY.read_text(encoding="utf-8"))
    ssh_cfg = resolve_ssh_cfg(cfg["targets"]["geegoo-agent"]["ssh"], "geegoo-agent")
    client = paramiko.SSHClient()
    client.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    client.connect(
        hostname=ssh_cfg["host"],
        port=int(ssh_cfg.get("port", 22)),
        username=ssh_cfg["user"],
        password=ssh_cfg.get("password"),
        key_filename=ssh_cfg.get("key_file"),
        timeout=30,
    )
    _, stdout, _ = client.exec_command(
        'grep -E "^MACRO_API_KEY=" /home/ubuntu/.geegoo/agent.env | head -1',
        timeout=20,
    )
    line = stdout.read().decode("utf-8", errors="replace").strip()
    client.close()
    m = re.match(r"MACRO_API_KEY=(.*)", line)
    return m.group(1).strip().strip('"') if m else ""


def get_json(url: str, key: str) -> tuple[int, object]:
    req = urllib.request.Request(url, headers={"Authorization": f"Bearer {key}"})
    with urllib.request.urlopen(req, timeout=30) as resp:
        return resp.status, json.loads(resp.read().decode("utf-8", errors="replace"))


def main() -> int:
    date = sys.argv[1] if len(sys.argv) > 1 else "2026-11-23"
    key = macro_key()
    if not key:
        print("no MACRO_API_KEY")
        return 2
    for wf in ("cn_a", "hk"):
        print(f"\n### {wf} @ {date} canonical symbols")
        q = f"date={date}&pool_workflow={wf}&source=canonical"
        url = f"{MACRO.rstrip('/')}/macro/jev/results/symbols?{q}"
        try:
            st, data = get_json(url, key)
        except Exception as e:
            req = urllib.request.Request(url, headers={"Authorization": f"Bearer {key}"})
            try:
                with urllib.request.urlopen(req, timeout=30) as resp:
                    raw = resp.read(500).decode("utf-8", errors="replace")
                print("GET failed", e, "raw=", raw[:300])
            except Exception as e2:
                print("GET failed", e, e2)
            continue
        print("HTTP", st)
        items = ((data.get("data") or {}).get("items") or []) if isinstance(data, dict) else []
        if not items:
            print("(no canonical rows)")
        for it in items:
            if not isinstance(it, dict):
                continue
            print(
                f"  {it.get('symbol')} status={it.get('status')} "
                f"batch={it.get('source_batch_id')} rev={it.get('revision')} "
                f"rid={(it.get('result_id') or '')[:48]} "
                f"run={(str(it.get('agent_run_id') or ''))[:28]}"
            )
        q2 = f"workflow={wf}&date_from={date}&date_to={date}&limit=1"
        url2 = f"{MACRO.rstrip('/')}/macro/batch/runs?{q2}"
        try:
            _, br = get_json(url2, key)
            runs = br.get("runs") or br.get("items") or []
            if runs:
                r = runs[0]
                print(f"latest batch run: id={r.get('id')} success={r.get('success')} {r.get('summary')}")
        except Exception as e:
            print("batch/runs", e)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())