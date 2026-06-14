"""Sample arguments for exercising every registered tool in tests."""

from __future__ import annotations

from geegoo_agent.tools.base import BaseTool
from geegoo_agent.tools.catalog import BESPOKE_TOOL_NAMES, SPEC_BY_NAME, FieldSpec, HttpToolSpec
from geegoo_agent.tools.http_api import HttpApiTool

_STR_SAMPLES: dict[str, str] = {
    "code": "00700.HK",
    "regex": "00700",
    "bot_id": "66494754fbe37cd6846ebd89",
    "report_id": "rpt-001",
    "name": "腾讯控股",
    "stock_name": "腾讯控股",
    "signal_id": "sig-test-001",
    "bot_type": "DCA",
    "report_date": "2026-06-05",
    "prompt_id": "69ec7035b9ccd3d9befc6c23",
    "title": "测试摘要",
    "summary": "测试内容",
    "step": "test_step",
    "message": "test message",
    "bot_name": "DCA",
    "reason": "test reason",
    "report": "report body",
    "language": "cn",
}

_BESPOKE_ARGS: dict[str, dict] = {
    "check_trading_day": {"code": "00700.HK"},
    "get_report_bot_codes": {},
    "fetch_market_news": {"market": "US"},
    "fetch_stock_news": {"code": "00700.HK", "stock_name": "腾讯控股"},
    "get_mcp_analysis": {
        "name": "腾讯控股",
        "code": "00700.HK",
        "period": "hourly",
    },
    "get_capital_flow": {"code": "00700.HK", "period": "DAY"},
    "get_capital_distribution": {"code": "00700.HK"},
    "get_bot_yesterday_attitude": {"bot_id": "bot-1", "code": "00700.HK"},
    "get_stock_daily_reports": {"code": "00700.HK", "report_date": "2026-06-05"},
    "list_today_reports": {"code": "00700.HK"},
    "recall": {"query": "腾讯 股价"},
    "recall_yesterday_summary": {"code": "00700.HK"},
    "read_working_state": {},
    "create_pre_market_report": {
        "code": "00700.HK",
        "stock_name": "腾讯控股",
        "bot_id": "bot-1",
        "bot_name": "DCA",
        "bot_type": "DCA",
        "result": "long",
        "confidence": "high",
        "reason": "test reason",
        "suggestion": "buy",
        "report": "full report",
    },
    "save_local_report": {
        "code": "00700.HK",
        "content": "# test report",
        "report_type": "premarket",
    },
    "send_feishu_summary": {"title": "测试", "summary": "摘要", "code": "00700.HK"},
    "write_execution_log": {"step": "test", "message": "ok"},
}


def _sample_for_field(field: FieldSpec, tool_name: str) -> object:
    if field.name == "payload":
        if tool_name.startswith("create_"):
            return {"botname": "test-bot", "code": "00700.HK"}
        if tool_name.startswith("update_"):
            return {"botname": "test-bot-renamed"}
        if tool_name == "loopback_strategy":
            return {
                "strategy_type": "dca",
                "code": "00700.HK",
                "frequency": "60m",
                "fund": 100000,
            }
        if "prompt_template" in tool_name:
            return {"name": "test-template", "list": []}
        return {}
    if field.name == "type" and tool_name == "get_bot_log_by_type":
        return "DCA"
    if field.name == "type" and tool_name == "get_single_prompt_template":
        return "tech"
    if field.name == "period" and tool_name == "get_single_prompt_template":
        return "monthly"
    if field.type_name == "str":
        if field.required or field.default is None:
            return _STR_SAMPLES.get(field.name, "test")
        return None
    if field.type_name == "int":
        return field.default if field.default is not None else 6
    if field.type_name == "bool":
        return True
    if field.type_name == "list":
        return ["HK"]
    if field.type_name == "dict":
        return field.default if field.default is not None else {}
    return "test"


def sample_arguments(tool: BaseTool) -> dict:
    if tool.name in _BESPOKE_ARGS:
        return dict(_BESPOKE_ARGS[tool.name])
    if isinstance(tool, HttpApiTool):
        spec = SPEC_BY_NAME[tool.name]
        return _sample_for_spec(spec)
    schema = tool.input_model.model_json_schema()
    props = schema.get("properties", {})
    required = set(schema.get("required", []))
    return {
        key: _STR_SAMPLES.get(key, "test")
        for key in props
        if key in required
    }


def _sample_for_spec(spec: HttpToolSpec) -> dict:
    args: dict = {}
    for field in spec.fields:
        value = _sample_for_field(field, spec.name)
        if value is not None:
            args[field.name] = value
    return args


def is_http_tool(tool: BaseTool) -> bool:
    return isinstance(tool, HttpApiTool)


def is_bespoke_tool(tool: BaseTool) -> bool:
    return tool.name in BESPOKE_TOOL_NAMES
