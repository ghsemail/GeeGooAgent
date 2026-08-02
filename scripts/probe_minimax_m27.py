#!/usr/bin/env python3
"""Probe listRemoteModels + MiniMax M2.7 chat on catalog/analyze-api."""
from __future__ import annotations

import json
import sys
import urllib.error
import urllib.request
from pathlib import Path

# trading_operation/lib/api/server_url.dart (catalog)
CATALOG = "http://146.56.225.252:3210"
CATALOG_KEY = "850367bc6d5fe8a4a53f267f5c308ac6d2ec1474d1764fe9"


def post(url: str, body: dict, token: str) -> dict:
    req = urllib.request.Request(
        url,
        data=json.dumps(body).encode(),
        headers={
            "Content-Type": "application/json",
            "Authorization": f"Bearer {token}",
        },
        method="POST",
    )
    with urllib.request.urlopen(req, timeout=60) as resp:
        return json.loads(resp.read().decode())


def main() -> int:
    print("=== 1) listRemoteModels (MiniMax base) ===")
    try:
        res = post(
            f"{CATALOG}/listRemoteModels",
            {"base_url": "https://api.minimaxi.com", "token": ""},
            CATALOG_KEY,
        )
    except urllib.error.HTTPError as e:
        print("HTTP", e.code, e.read().decode()[:500])
        return 1
    print(json.dumps(res, ensure_ascii=False, indent=2))
    data = res.get("data") or []
    has_m27 = any("M2.7" in str(x) for x in data)
    print("has M2.7:", has_m27)

    print("\n=== 2) getModel (configured minimax row) ===")
    models = post(f"{CATALOG}/getModel", {}, CATALOG_KEY)
    minimax_rows = []
    if isinstance(models, list):
        for m in models:
            hay = f"{m.get('base_url','')} {m.get('name','')} {m.get('display_name','')}".lower()
            if "minimax" in hay:
                minimax_rows.append(m)
    print("minimax models:", len(minimax_rows))
    for m in minimax_rows:
        print(
            " -",
            m.get("display_name"),
            "|",
            m.get("name"),
            "|",
            m.get("type"),
            "|",
            (m.get("base_url") or "")[:40],
            "| token:",
            "yes" if (m.get("token") or "").strip() else "no",
        )

    configured = next((m for m in (models if isinstance(models, list) else []) if m.get("type") == "configured"), None)
    if configured:
        print("\nconfigured:", configured.get("display_name"), configured.get("name"))

    print("\n=== 3) MiniMax M2.7 direct API (if token in DB) ===")
    row = minimax_rows[0] if minimax_rows else None
    if not row or not (row.get("token") or "").strip():
        print("skip: no minimax token in catalog DB")
        return 0 if has_m27 else 2

    token = row["token"].strip()
    base = (row.get("base_url") or "https://api.minimaxi.com").rstrip("/")
    if base.endswith("/v1"):
        base = base[:-3]
    url = f"{base}/v1/text/chatcompletion_v2"
    payload = {
        "model": "MiniMax-M2.7",
        "messages": [{"role": "user", "content": "只回复 ok"}],
        "max_tokens": 32,
        "temperature": 0.1,
        "stream": False,
    }
    req = urllib.request.Request(
        url,
        data=json.dumps(payload).encode(),
        headers={"Authorization": f"Bearer {token}", "Content-Type": "application/json"},
        method="POST",
    )
    try:
        with urllib.request.urlopen(req, timeout=90) as resp:
            raw = json.loads(resp.read().decode())
    except urllib.error.HTTPError as e:
        print("HTTP", e.code, e.read().decode()[:800])
        return 3
    base_resp = raw.get("base_resp") or {}
    choices = raw.get("choices") or []
    content = ""
    if choices:
        content = ((choices[0].get("message") or {}).get("content") or "").strip()
    print("base_resp:", base_resp)
    print("content:", repr(content[:200]))
    ok = base_resp.get("status_code", 0) in (0, 1000) and bool(content)
    print("M2.7 direct OK:", ok)
    return 0 if ok else 4


if __name__ == "__main__":
    sys.exit(main())
