#!/usr/bin/env python3
"""统计某日标的层：inline / batch_runs / jev-results 每个 code 多少条痕迹。"""
from __future__ import annotations

import json
import sys
from collections import Counter
from pathlib import Path

import paramiko

DATE = sys.argv[1] if len(sys.argv) > 1 else "2026-11-23"
POOL = sys.argv[2] if len(sys.argv) > 2 else "cn_a"
SYMS = ["000001.SH", "000300.SH", "399001.SZ", "800000.HK", "800100.HK", "800700.HK"]

SKILL = Path.home() / ".cursor" / "skills" / "remote-deploy" / "scripts"
sys.path.insert(0, str(SKILL))
from credentials import resolve_ssh_cfg  # noqa: E402

DEPLOY = Path.home() / ".cursor" / "skills" / "remote-deploy" / "deploy.json"


def ssh_curl(path: str) -> str:
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
    safe = path.replace('"', "")
    cmd = (
        "set -a && . /home/ubuntu/.geegoo/agent.env && set +a && "
        f'curl -sS -H "Authorization: Bearer $MACRO_API_KEY" '
        f'"http://146.56.225.252:3220{safe}"'
    )
    _, stdout, stderr = client.exec_command(cmd, timeout=120)
    out = stdout.read().decode("utf-8", errors="replace")
    err = stderr.read().decode("utf-8", errors="replace")
    client.close()
    if err.strip():
        print(err, file=sys.stderr)
    return out


def main() -> int:
    print(f"=== date={DATE} pool={POOL} ===\n")

    snap = ssh_curl(f"/macro/quotes/snapshot?date={DATE}")
    try:
        data = json.loads(snap)
        items = (data.get("data") or {}).get("items") or []
        by_code = {str(it.get("code") or "").strip(): it for it in items if isinstance(it, dict)}
        print("[inline 日快照 items] 每个 code 在日文档里至多 1 行（后写覆盖）")
        for s in SYMS:
            row = by_code.get(s)
            if row:
                print(f"  {s}: 有 jev={row.get('jev_direction')} detail_ref={row.get('detail_ref','')[:24]}")
            else:
                print(f"  {s}: (无)")
    except json.JSONDecodeError:
        print("snapshot parse fail", snap[:200])

    wf = POOL
    runs_raw = ssh_curl(
        f"/macro/batch/runs?workflow={wf}&date_from={DATE}&date_to={DATE}&limit=40"
    )
    print("\n[batch/runs] 一次 progress 一条 run；同一标的可出现在多个 run 的 items 里")
    try:
        runs = (json.loads(runs_raw).get("data") or {}).get("items") or []
        print(f"  当日 {wf} run 条数: {len(runs)}")
        sym_hit = Counter()
        for r in runs:
            rid = r.get("id")
            st, detail, _ = 0, {}, ""
            detail_raw = ssh_curl(f"/macro/batch/runs/{rid}")
            run = (json.loads(detail_raw).get("data") or {}).get("run") or {}
            for p in run.get("pools") or []:
                for it in p.get("items") or []:
                    sym = str(it.get("symbol") or "").strip()
                    if sym and it.get("success"):
                        sym_hit[sym] += 1
        for s in SYMS:
            if s in sym_hit or s.endswith(".SH") or s.endswith(".SZ"):
                if POOL == "hk" and not s.endswith(".HK"):
                    continue
                if POOL == "cn_a" and s.endswith(".HK"):
                    continue
                print(f"  {s}: 出现在 {sym_hit.get(s, 0)} 个 success run 中")
    except Exception as e:
        print("  batch/runs", e, runs_raw[:300])

    jev_raw = ssh_curl(
        f"/macro/jev/results/symbols?date={DATE}&pool_workflow={POOL}&source=canonical"
    )
    print("\n[jev-results canonical] 物理多行；读侧按 (date,pool,symbol) 取 latest")
    try:
        jitems = (json.loads(jev_raw).get("data") or {}).get("items") or []
        sym_docs = Counter(str(x.get("symbol") or "") for x in jitems if isinstance(x, dict))
        print(f"  canonical 文档总数: {len(jitems)}")
        for s in SYMS:
            if POOL == "hk" and not s.endswith(".HK"):
                continue
            if POOL == "cn_a" and s.endswith(".HK"):
                continue
            n = sym_docs.get(s, 0)
            print(f"  {s}: {n} 条 canonical 行")
    except Exception as e:
        print("  jev", e, jev_raw[:200])

    for s in SYMS[:3] if POOL == "cn_a" else SYMS[3:]:
        if POOL == "cn_a" and s.endswith(".HK"):
            continue
        if POOL == "hk" and not s.endswith(".HK"):
            continue
        one = ssh_curl(
            f"/macro/jev/results/symbols/{s}?date={DATE}&pool_workflow={POOL}"
        )
        try:
            res = (json.loads(one).get("data") or {}).get("result") or {}
            print(
                f"\n  GET current {s}: status={res.get('status')} "
                f"source_batch_id={res.get('source_batch_id')} "
                f"data_source={res.get('data_source')} rev={res.get('revision')}"
            )
        except Exception:
            pass
    return 0


if __name__ == "__main__":
    raise SystemExit(main())