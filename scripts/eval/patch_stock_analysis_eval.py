#!/usr/bin/env python3
"""Patch stock_analysis TurnPlan eval cases on production via dashboard API."""
from __future__ import annotations

import json
import os
import sys
import urllib.error
import urllib.request

BASE = os.environ.get("EVAL_API_BASE", "http://118.195.135.97:3110/op_agent/v1")
API_KEY = os.environ["GEEGOO_BOT_AGENT_API_KEY"]
OPS_USER = os.environ.get("EVAL_OPS_USER_ID", "64afddf8c2a269ac1846fe70")

PATCHES: dict[str, dict] = {
    "turn_plan_stock_price": {
        "title": "TurnPlan · 单轮 · 查股价",
        "description": "独立 session：查询腾讯控股现价（snapshot，非 MCP 分析）。",
        "expect_act": "quote_price",
    },
    "turn_plan_stock_price_trend": {
        "title": "TurnPlan · 单轮 · 分析价格走势",
        "description": "独立 session：分析腾讯最近一个月价格走势（MCP）。",
        "expect_act": "technical_analysis",
        "expect_reply": {
            "rubric": "应分析腾讯最近一个月的价格走势（趋势、涨跌、关键价位等），走 MCP 深度分析而非只报现价。",
            "must_cover": ["腾讯", "走势"],
        },
    },
    "turn_plan_stock_technical_chain": {
        "title": "TurnPlan · 多轮 · 查价后看K线",
        "description": "同 session：先查腾讯股价，再自然续问 K 线图/技术面分析。",
        "expect_act": "technical_analysis",
        "expect_reply": {
            "rubric": "在已查腾讯股价的基础上，应补充 K 线图或技术面分析，而非只重复报价。",
            "must_cover": ["腾讯"],
        },
    },
    "turn_plan_stock_symbol_switch": {
        "title": "TurnPlan · 多轮 · 切换分析标的",
        "description": "同 session：先问中际旭创价格趋势，再切换分析贵州茅台。",
        "expect_act": "symbol_resolve",
        "expect_reply": {
            "rubric": "应识别用户切换到贵州茅台，并给出茅台相关分析或行情，而非继续只聊中际旭创。",
            "must_cover": ["茅台"],
        },
    },
    "turn_plan_stock_colloquial_ref": {
        "title": "TurnPlan · 多轮 · 代词指代续问",
        "description": "同 session：建立中际旭创价格走势上下文后，用「它」续问信号趋势。",
        "expect_act": "context_followup",
        "expect_reply": {
            "rubric": "应理解「它」指代上一轮的中际旭创，并继续给出该标的信号趋势或技术面解读，而非换标的或跑回测。",
            "must_cover": ["中际", "信号"],
        },
    },
}


def request(method: str, path: str, body: dict | None = None) -> dict:
    headers = {
        "Authorization": f"Bearer {API_KEY}",
        "X-Client-Source": "trading_operation",
        "X-User-Id": OPS_USER,
    }
    data = None
    if body is not None:
        headers["Content-Type"] = "application/json"
        data = json.dumps(body).encode()
    req = urllib.request.Request(f"{BASE}{path}", data=data, headers=headers, method=method)
    with urllib.request.urlopen(req, timeout=60) as resp:
        return json.load(resp)


def build_steps(dialogue: list[dict]) -> list[str]:
    texts = [str(t.get("text", "")).strip() for t in dialogue if str(t.get("text", "")).strip()]
    steps = ["新 session：运行前清空 Dock Chat"]
    if len(texts) == 1:
        steps.append(f"单轮发送：「{texts[0]}」")
    elif len(texts) > 1:
        joined = " → ".join(f"「{t}」" for t in texts)
        steps.append(f"同 session 按序发送：{joined}")
    steps.append("verify：校验路由/工具/回复关键词 + LLM 语义评判")
    return steps


def patch_case(case_id: str, patch: dict) -> None:
    row = request("GET", f"/dashboard/eval/cases/{case_id}")["case"]
    opts = dict(row.get("options") or {})
    dialogue = opts.get("dialogue") or []
    if patch.get("expect_reply"):
        opts["expect_reply"] = patch["expect_reply"]
        opts["judge"] = {"enabled": True, "model_slot": "auxiliary", "min_score": 0.7}
    act = patch.get("expect_act", "")
    if act:
        opts["expect_act"] = act
        intent = dict(opts.get("expect_intent") or {})
        intent.update({"domain": "stock_analysis", "mode": "gather", "sop": False, "act": act})
        opts["expect_intent"] = intent
    payload = {
        "title": patch.get("title", row.get("title")),
        "description": patch.get("description", row.get("description")),
        "steps": build_steps(dialogue),
        "supports_random_stock": row.get("supports_random_stock", False),
        "options": opts,
        "sort_order": row.get("sort_order", 0),
        "enabled": row.get("enabled", True),
    }
    request("PUT", f"/dashboard/eval/cases/{case_id}", payload)
    print(f"OK {case_id}")


def main() -> int:
    for case_id, patch in PATCHES.items():
        try:
            patch_case(case_id, patch)
        except urllib.error.HTTPError as exc:
            print(f"FAIL {case_id}: {exc.read().decode()}", file=sys.stderr)
            return 1
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
