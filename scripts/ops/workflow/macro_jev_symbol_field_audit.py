#!/usr/bin/env python3
"""Audit canonical symbol JEV rows: lab embed + agent binding completeness."""
from __future__ import annotations

import json
import re
import sys
import urllib.parse
import urllib.request
from pathlib import Path

import paramiko

SKILL = Path.home() / ".cursor" / "skills" / "remote-deploy" / "scripts"
sys.path.insert(0, str(SKILL))
from credentials import resolve_ssh_cfg  # noqa: E402

DEPLOY = Path.home() / ".cursor" / "skills" / "remote-deploy" / "deploy.json"
MACRO = "http://146.56.225.252:3220"
POOLS = ("cn_a", "hk", "us", "commodity_fx", "bond", "crypto", "volatility")
LANE_KEYS = (
    "price_analysis_jev",
    "kline_jev",
    "technical_analysis_jev",
    "index_trend_jev",
    "jev_unified",
)


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


def get_json(path: str, key: str) -> dict | list | None:
    url = f"{MACRO.rstrip('/')}{path}"
    req = urllib.request.Request(url, headers={"Authorization": f"Bearer {key}"})
    with urllib.request.urlopen(req, timeout=45) as resp:
        raw = json.loads(resp.read().decode("utf-8", errors="replace"))
    if isinstance(raw, dict) and raw.get("code") not in (None, 100, 0):
        return None
    return raw


def lane_hits(lab: dict) -> list[str]:
    out: list[str] = []
    for k in LANE_KEYS:
        v = lab.get(k)
        if v is None:
            continue
        if isinstance(v, dict) and v:
            out.append(k)
        elif v:
            out.append(k)
    return out


def audit_item(it: dict) -> dict:
    lab = it.get("diagnose_episode_summary") or {}
    if not isinstance(lab, dict):
        lab = {}
    lanes = lane_hits(lab)
    embed_ref = (lab.get("detail_ref") or "").strip()
    return {
        "symbol": it.get("symbol"),
        "date": it.get("date"),
        "status": it.get("status"),
        "dir": (it.get("jev_direction") or "")[:12],
        "detail_ref": (it.get("detail_ref") or "")[:24],
        "embed_ref": embed_ref[:24],
        "session": bool((it.get("agent_session_id") or "").strip()),
        "run": bool((it.get("agent_run_id") or "").strip()),
        "lab": bool(lab),
        "lanes": len(lanes),
        "lane_keys": ",".join(lanes) if lanes else "-",
        "unified": bool(lab.get("jev_unified") if isinstance(lab.get("jev_unified"), dict) else lab.get("jev_unified")),
    }


def main() -> int:
    dates = sys.argv[1:] if len(sys.argv) > 1 else ["2026-11-24", "2026-11-23"]
    key = macro_key()
    if not key:
        print("no MACRO_API_KEY")
        return 2
    for date in dates:
        print(f"\n======== {date} ========")
        for wf in POOLS:
            q = urllib.parse.urlencode(
                {"date": date, "pool_workflow": wf, "source": "canonical"}
            )
            raw = get_json(f"/macro/jev/results/symbols?{q}", key)
            items = []
            if isinstance(raw, dict):
                items = (raw.get("data") or {}).get("items") or []
            if not items:
                print(f"{wf:14} (0 rows)")
                continue
            ok = fail = partial = 0
            for it in items:
                if not isinstance(it, dict):
                    continue
                a = audit_item(it)
                if a["status"] != "success":
                    fail += 1
                    continue
                if a["lab"] and a["lanes"] >= 4 and a["session"] and a["run"]:
                    ok += 1
                else:
                    partial += 1
            print(
                f"{wf:14} n={len(items)} full4={ok} partial={partial} fail={fail}"
            )
            for it in items:
                a = audit_item(it)
                flag = "OK" if (
                    a["status"] == "success"
                    and a["lab"]
                    and a["lanes"] >= 4
                    and a["session"]
                    and a["run"]
                ) else "!!"
                print(
                    f"  {flag} {a['symbol']:18} {a['status']:10} "
                    f"lanes={a['lanes']} sess={int(a['session'])} run={int(a['run'])} "
                    f"ref={a['detail_ref'] or '-'} emb={a['embed_ref'] or '-'}"
                )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())