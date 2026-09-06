#!/usr/bin/env python3
"""Run TurnPlan stock_analysis live eval cases via agent-runtime API (remote localhost)."""
from __future__ import annotations

import argparse
import json
import sys
import textwrap
import urllib.error
import urllib.request
import os

CASES = [
    {
        "id": "turn_plan_stock_price",
        "setup": [],
        "message": "帮我查一下腾讯控股现在的股价",
    },
    {
        "id": "turn_plan_stock_technical_chain",
        "setup": ["帮我查一下腾讯控股现在的股价"],
        "message": "再帮我看看腾讯的技术面和K线图",
    },
    {
        "id": "turn_plan_stock_symbol_switch",
        "setup": ["帮我分析一下中际旭创"],
        "message": "不聊中际旭创了，帮我分析一下贵州茅台",
    },
    {
        "id": "turn_plan_stock_colloquial_ref",
        "setup": ["帮我分析一下中际旭创"],
        "message": "它最近走势怎么样",
    },
]


def load_env() -> tuple[str, str]:
    import os

    home = os.path.expanduser("~/.geegoo")
    env: dict[str, str] = {}
    with open(os.path.join(home, "agent.env"), encoding="utf-8") as f:
        for line in f:
            line = line.strip()
            if not line or line.startswith("#") or "=" not in line:
                continue
            k, v = line.split("=", 1)
            k = k.strip()
            if k.startswith("export "):
                k = k[7:].strip()
            env[k] = v.strip().strip('"').strip("'")
    cfg = json.load(open(os.path.join(home, "config.json"), encoding="utf-8"))
    runtime_key = env.get("GEEGOO_AGENT_RUNTIME_API_KEY", "") or cfg.get("api_key", "")
    mcp_token = cfg.get("mcp_token", "")
    if not mcp_token:
        raise RuntimeError("missing mcp_token in config.json")
    return runtime_key, mcp_token


def post_json(url: str, body: dict, headers: dict, timeout: int = 900) -> tuple[int, str]:
    data = json.dumps(body, ensure_ascii=False).encode("utf-8")
    req = urllib.request.Request(url, data=data, headers=headers, method="POST")
    try:
        with urllib.request.urlopen(req, timeout=timeout) as resp:
            return resp.status, resp.read().decode("utf-8", errors="replace")
    except urllib.error.HTTPError as e:
        return e.code, e.read().decode("utf-8", errors="replace")


def chat_turn(runtime_key: str, mcp_token: str, message: str, session_id: str = "") -> dict:
    headers = {
        "Content-Type": "application/json",
        "Authorization": f"Bearer {runtime_key}",
        "X-Approve-Writes": "true",
    }
    body = {"message": message, "mcp_token": mcp_token}
    if session_id:
        body["session_id"] = session_id
    status, raw = post_json("http://127.0.0.1:3400/v1/chat/stream", body, headers, timeout=900)
    if status != 200:
        return {"ok": False, "error": f"chat HTTP {status}: {raw[:500]}"}

    sid = session_id
    turn_end: dict = {}
    event = ""
    for line in raw.splitlines():
        if line.startswith("event:"):
            event = line[6:].strip()
        elif line.startswith("data:"):
            payload = line[5:].strip()
            if event == "connected":
                try:
                    sid = json.loads(payload).get("session_id", sid) or sid
                except json.JSONDecodeError:
                    pass
            elif event == "turn_end":
                try:
                    turn_end = json.loads(payload)
                except json.JSONDecodeError:
                    turn_end = {"raw": payload}
    failed = bool(turn_end.get("failed"))
    reply = turn_end.get("assistant_text") or ""
    return {
        "ok": not failed,
        "session_id": sid,
        "reply": reply,
        "failed": failed,
        "error": turn_end.get("error", ""),
    }


def verify_case(runtime_key: str, case_id: str, session_id: str) -> dict:
    headers = {
        "Content-Type": "application/json",
        "Authorization": f"Bearer {runtime_key}",
    }
    status, raw = post_json(
        f"http://127.0.0.1:3400/v1/dashboard/eval/cases/{case_id}/verify",
        {"session_id": session_id},
        headers,
        timeout=120,
    )
    try:
        data = json.loads(raw)
    except json.JSONDecodeError:
        return {"ok": False, "http": status, "raw": raw[:800]}
    data["http"] = status
    return data


def run_case(runtime_key: str, mcp_token: str, case: dict) -> dict:
    cid = case["id"]
    session_id = ""
    for i, setup_msg in enumerate(case.get("setup") or []):
        r = chat_turn(runtime_key, mcp_token, setup_msg, session_id)
        if not r.get("session_id"):
            return {"case_id": cid, "passed": False, "stage": f"setup[{i}]", "error": r}
        session_id = r["session_id"]
        if not r.get("ok"):
            return {"case_id": cid, "passed": False, "stage": f"setup[{i}]", "error": r}

    r = chat_turn(runtime_key, mcp_token, case["message"], session_id)
    if not r.get("session_id"):
        return {"case_id": cid, "passed": False, "stage": "final", "error": r}
    session_id = r["session_id"]
    if not r.get("ok"):
        return {"case_id": cid, "passed": False, "stage": "final", "error": r, "reply_preview": r.get("reply", "")[:200]}

    v = verify_case(runtime_key, cid, session_id)
    passed = bool(v.get("ok")) and v.get("http") == 200
    return {
        "case_id": cid,
        "passed": passed,
        "session_id": session_id,
        "reply_preview": (r.get("reply") or "")[:240],
        "verify": v,
    }


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--case", help="run single case id (default: all)")
    args = parser.parse_args()

    runtime_key, mcp_token = load_env()
    cases = CASES
    only = os.environ.get("TURNPLAN_ONLY_CASE", "").strip()
    if only:
        cases = [c for c in CASES if c["id"] == only]
    if args.case:
        cases = [c for c in CASES if c["id"] == args.case]
        if not cases:
            print(f"unknown case: {args.case}", file=sys.stderr)
            return 2

    results = []
    all_ok = True
    for case in cases:
        print(f"\n======== {case['id']} ========", flush=True)
        res = run_case(runtime_key, mcp_token, case)
        results.append(res)
        if res.get("passed"):
            print(f"PASS  session={res.get('session_id')}", flush=True)
            preview = res.get("reply_preview") or ""
            if preview:
                print(textwrap.fill(preview, width=88), flush=True)
        else:
            all_ok = False
            print(f"FAIL  stage={res.get('stage', 'verify')}", flush=True)
            if res.get("reply_preview"):
                print("reply:", res["reply_preview"], flush=True)
            verify = res.get("verify") or res.get("error") or {}
            detail = verify.get("detail") if isinstance(verify, dict) else str(verify)
            if detail:
                print("detail:", detail, flush=True)
            checks = verify.get("checks") if isinstance(verify, dict) else None
            if checks:
                for c in checks:
                    if not c.get("passed"):
                        print(f"  - {c.get('type')}: {c.get('detail')}", flush=True)

    print("\n======== SUMMARY ========", flush=True)
    for r in results:
        mark = "PASS" if r.get("passed") else "FAIL"
        print(f"{mark}  {r['case_id']}", flush=True)
    return 0 if all_ok else 1


if __name__ == "__main__":
    raise SystemExit(main())
