#!/usr/bin/env python3
"""Run TurnPlan live eval cases via agent-runtime (chat/stream + verify).

Reads case manifest from turnplan_cases.json (generated from Go source).
Intended to run on the agent host (localhost :3400).
"""
from __future__ import annotations

import argparse
import json
import os
import sys
import textwrap
import urllib.error
import urllib.request
from pathlib import Path

MANIFEST = Path(__file__).with_name("turnplan_cases.json")
CHAT_TIMEOUT = int(os.environ.get("TURNPLAN_CHAT_TIMEOUT", "900"))
VERIFY_TIMEOUT = int(os.environ.get("TURNPLAN_VERIFY_TIMEOUT", "120"))


def load_manifest() -> dict:
    return json.loads(MANIFEST.read_text(encoding="utf-8"))


def load_env() -> tuple[str, str]:
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


def post_json(url: str, body: dict, headers: dict, timeout: int) -> tuple[int, str]:
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
    status, raw = post_json("http://127.0.0.1:3400/v1/chat/stream", body, headers, CHAT_TIMEOUT)
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
    return {
        "ok": not bool(turn_end.get("failed")),
        "session_id": sid,
        "reply": turn_end.get("assistant_text") or "",
        "failed": bool(turn_end.get("failed")),
        "error": turn_end.get("error", ""),
    }


def verify_case(runtime_key: str, case_id: str, session_id: str) -> dict:
    headers = {"Content-Type": "application/json", "Authorization": f"Bearer {runtime_key}"}
    status, raw = post_json(
        f"http://127.0.0.1:3400/v1/dashboard/eval/cases/{case_id}/verify",
        {"session_id": session_id},
        headers,
        VERIFY_TIMEOUT,
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
        return {
            "case_id": cid,
            "passed": False,
            "stage": "final",
            "error": r,
            "reply_preview": (r.get("reply") or "")[:200],
        }

    v = verify_case(runtime_key, cid, session_id)
    passed = bool(v.get("ok")) and v.get("http") == 200
    return {
        "case_id": cid,
        "passed": passed,
        "session_id": session_id,
        "reply_preview": (r.get("reply") or "")[:240],
        "verify": v,
    }


def select_cases(manifest: dict, *, case_id: str, category: str) -> list[dict]:
    cases = manifest.get("cases") or []
    only = os.environ.get("TURNPLAN_ONLY_CASE", "").strip()
    if only:
        case_id = only
    if case_id:
        picked = [c for c in cases if c["id"] == case_id]
        if not picked:
            raise SystemExit(f"unknown case: {case_id}")
        return picked
    if category:
        picked = [c for c in cases if c.get("category") == category]
        if not picked:
            raise SystemExit(f"unknown category: {category}")
        return picked
    return cases


def print_case_result(res: dict) -> None:
    if res.get("passed"):
        print(f"PASS  session={res.get('session_id')}", flush=True)
        preview = res.get("reply_preview") or ""
        if preview:
            print(textwrap.fill(preview, width=88), flush=True)
        return
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


def main() -> int:
    parser = argparse.ArgumentParser(description="Run TurnPlan live eval via agent-runtime")
    parser.add_argument("--list", action="store_true", help="list cases grouped by category")
    parser.add_argument("--case", help="run one case id, e.g. turn_plan_stock_price")
    parser.add_argument("--category", help="run one category, e.g. stock_analysis")
    args = parser.parse_args()

    manifest = load_manifest()
    if args.list:
        cats = {c["id"]: c["title"] for c in manifest.get("categories", [])}
        by_cat: dict[str, list[str]] = {}
        for c in manifest.get("cases", []):
            by_cat.setdefault(c.get("category", "?"), []).append(c["id"])
        for cat in manifest.get("categories", []):
            title = cat.get("title") or cat["id"]
            ids = by_cat.get(cat["id"], [])
            print(f"\n[{cat['id']}] {title} ({len(ids)})")
            for cid in ids:
                print(f"  - {cid}")
        return 0

    runtime_key, mcp_token = load_env()
    try:
        cases = select_cases(manifest, case_id=args.case or "", category=args.category or "")
    except SystemExit as e:
        print(e, file=sys.stderr)
        return 2

    results = []
    all_ok = True
    for case in cases:
        print(f"\n======== {case['id']} ========", flush=True)
        res = run_case(runtime_key, mcp_token, case)
        results.append(res)
        print_case_result(res)
        if not res.get("passed"):
            all_ok = False

    print("\n======== SUMMARY ========", flush=True)
    for r in results:
        mark = "PASS" if r.get("passed") else "FAIL"
        print(f"{mark}  {r['case_id']}", flush=True)
    return 0 if all_ok else 1


if __name__ == "__main__":
    raise SystemExit(main())
