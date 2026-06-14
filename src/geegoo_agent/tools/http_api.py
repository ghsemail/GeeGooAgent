"""Generic HTTP API tools built from catalog specs."""

from __future__ import annotations

from typing import Any

from pydantic import BaseModel, Field, create_model

from geegoo_agent.exceptions import ConfigError
from geegoo_agent.tools.base import BaseTool
from geegoo_agent.tools.catalog import HttpToolSpec
from geegoo_agent.tools.types import ToolContext, ToolResult

_PYDANTIC_TYPES: dict[str, Any] = {
    "str": str,
    "int": int,
    "float": float,
    "bool": bool,
    "dict": dict[str, Any],
    "list": list[Any],
}


def _build_input_model(spec: HttpToolSpec) -> type[BaseModel]:
    fields: dict[str, Any] = {}
    for field_spec in spec.fields:
        py_type = _PYDANTIC_TYPES[field_spec.type_name]
        if field_spec.required:
            fields[field_spec.name] = (
                py_type,
                Field(description=field_spec.description),
            )
        elif field_spec.default is not None:
            fields[field_spec.name] = (
                py_type,
                Field(default=field_spec.default, description=field_spec.description),
            )
        else:
            fields[field_spec.name] = (
                py_type | None,
                Field(default=None, description=field_spec.description),
            )
    model_name = "".join(part.title() for part in spec.name.split("_")) + "Params"
    return create_model(model_name, **fields)  # type: ignore[call-overload]


class HttpApiTool(BaseTool):
    """Thin wrapper that forwards JSON to geegoo mcp (5700) APIs."""

    def __init__(self, spec: HttpToolSpec) -> None:
        self._spec = spec
        self.name = spec.name
        self.description = spec.description
        self.category = spec.category
        self.input_model = _build_input_model(spec)

    def run(self, params: BaseModel, ctx: ToolContext) -> ToolResult:
        if ctx.dry_run:
            return ToolResult(
                status="dry_run",
                summary=f"dry-run: skipped {self.name}",
                data={"tool": self.name, "path": self._spec.path},
            )
        client = self._resolve_client(ctx)
        body = self._build_body(params)
        if self._spec.requires_mcp_token:
            body["mcp_token"] = ctx.mcp_token
        if self._spec.response_mode == "direct":
            payload = client.post_direct(self._spec.path, body)
            data, summary = self._normalize_direct_response(payload)
        else:
            payload = client.post(self._spec.path, body)
            data = payload.get("data", payload)
            summary = f"{self.name} succeeded"
        return ToolResult(
            status="ok",
            summary=summary,
            data=data if isinstance(data, dict) else {"items": data},
        )

    def _resolve_client(self, ctx: ToolContext):
        if ctx.geegoo_bot_client is None:
            raise ConfigError("geegoo_bot_client not configured")
        return ctx.geegoo_bot_client

    def _normalize_direct_response(self, payload: Any) -> tuple[dict[str, Any], str]:
        if isinstance(payload, list):
            return {"items": payload, "count": len(payload)}, f"{self.name}: {len(payload)} item(s)"
        if isinstance(payload, dict):
            if payload.get("code") == 100 and "data" in payload:
                inner = payload["data"]
                if isinstance(inner, dict):
                    return inner, f"{self.name} succeeded"
                return {"data": inner}, f"{self.name} succeeded"
            if "price" in payload:
                return payload, f"{self.name}: price={payload['price']}"
            return payload, f"{self.name} succeeded"
        return {"value": payload}, f"{self.name} succeeded"

    def _build_body(self, params: BaseModel) -> dict[str, Any]:
        raw = params.model_dump(exclude_none=True)
        if self._spec.merge_payload and "payload" in raw:
            payload = raw.pop("payload") or {}
            if not isinstance(payload, dict):
                payload = {}
            return {**payload, **raw}
        return raw


def build_http_tools(specs: list[HttpToolSpec]) -> list[HttpApiTool]:
    return [HttpApiTool(spec) for spec in specs]
