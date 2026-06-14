#!/usr/bin/env python3
"""Demo: analyze Tencent via GeeGoo Agent tools (search → price → optional analysis)."""

from __future__ import annotations

import json
import os
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT / "src"))

from geegoo_agent.config import load_config
from geegoo_agent.tools.bootstrap import register_all_tools
from geegoo_agent.tools.registry import ToolRegistry
from geegoo_agent.tools.types import ToolCallRequest, ToolContext
from geegoo_agent.clients.geegoo_bot import GeeGooBotClient
from geegoo_agent.clients.market import MarketClient
from geegoo_agent.infra.sandbox import NetworkPolicy


def run_tool(
    registry: ToolRegistry,
    ctx: ToolContext,
    name: str,
    arguments: dict,
    *,
    optional: bool = False,
) -> dict:
    result = registry.execute(ToolCallRequest(name=name, arguments=arguments), ctx)
    print(f"\n=== {name} ===")
    print(f"status: {result.status}")
    print(f"summary: {result.summary}")
    if result.data is not None:
        print(json.dumps(result.data, ensure_ascii=False, indent=2)[:3000])
    if result.status == "error":
        if optional:
            print(f"[可选步骤失败] {name}")
            return {}
        raise SystemExit(1)
    return result.data or {}


def main() -> int:
    config_path = os.environ.get("GEEGOO_CONFIG", str(ROOT / "config.local.json"))
    config = load_config(config_path)
    if os.environ.get("MCP_TOKEN"):
        config = config.model_copy(update={"mcp_token": os.environ["MCP_TOKEN"]})

    network = NetworkPolicy(config.sandbox.allowed_hosts)
    market = MarketClient(config.base_url, config.api_key, network, retry_wait_seconds=0)
    geegoo = GeeGooBotClient(config.geegoo_url, config.geegoo_api_key, network, retry_wait_seconds=0)
    registry = register_all_tools(ToolRegistry())
    ctx = ToolContext(
        session_id="demo-tencent",
        mcp_token=config.mcp_token,
        dry_run=False,
        workspace_root=config.workspace_root,
        market_client=market,
        geegoo_bot_client=geegoo,
        project_root=ROOT,
    )

    code, name = "00700.HK", "腾讯控股"

    # Step 1: 搜标的（5700 /searchCode；下游 Signal 不可用时跳过）
    search = run_tool(
        registry, ctx, "search_code", {"regex": "腾讯", "market": ["HK"]}, optional=True
    )
    items = search.get("items", [])
    tencent = next(
        (i for i in items if "00700" in str(i.get("code", ""))),
        items[0] if items else None,
    )
    if tencent:
        code = str(tencent.get("code", code))
        name = str(tencent.get("name", name))
    print(f"\n选定: {name} ({code})")

    # Step 2: 最新价 — 优先 5700 getCurrentPrice；502 时改用 5900 get_ticker
    price = None
    price_data = run_tool(registry, ctx, "get_current_price", {"code": code}, optional=True)
    price = price_data.get("price")
    if price is None and config.mcp_token:
        ticker = run_tool(registry, ctx, "get_ticker", {"code": code, "num": 5})
        ticks = ticker.get("items") or []
        if ticks:
            price = ticks[-1].get("price")
            print(f"逐笔最新价: {price} @ {ticks[-1].get('time')}")

    # Step 3: 资金分布/流向（需 mcp_token）
    if config.mcp_token:
        run_tool(registry, ctx, "get_capital_distribution", {"code": code})
        run_tool(registry, ctx, "get_capital_flow", {"code": code, "period": "DAY"})
    else:
        print("\n[跳过] get_capital_* / get_mcp_analysis：config 无 mcp_token")

    # Step 4: 查 prompt 模板 → 技术分析（5700 502 时用盘前默认 prompt_id）
    if config.mcp_token:
        prompt_id = "69ec7035b9ccd3d9befc6c23"
        prompts = run_tool(
            registry,
            ctx,
            "get_single_prompt_template",
            {"type": "tech", "period": "daily"},
            optional=True,
        )
        prompt_items = prompts.get("items", [])
        if isinstance(prompt_items, list) and prompt_items:
            prompt_id = str(prompt_items[0].get("prompt_id", prompt_id))
        run_tool(
            registry,
            ctx,
            "get_mcp_analysis",
            {
                "name": name,
                "code": code,
                "prompt_id": prompt_id,
                "period": "daily",
            },
        )
    else:
        print("\n[提示] 设置 MCP_TOKEN 或 config.mcp_token 后可跑完整技术分析")

    print(f"\n完成：{name} 现价约 {price}（{code}）")
    market.close()
    geegoo.close()
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
