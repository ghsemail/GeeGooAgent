#!/usr/bin/env python3
"""GeeGooAgent optional cognition advisor (suggestion-only sidecar).

Endpoints:
  GET  /health
  POST /v1/advisor/rank      — reorder RankItem list
  POST /v1/advisor/evaluate  — post-turn advisory judgment

Must NOT return tool_calls, state mutations, or workflow decisions.
"""
from __future__ import annotations

import json
import os
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from typing import Any


def rank_items(items: list[dict[str, Any]]) -> list[dict[str, Any]]:
    """Default: sort by score descending, stable on id."""
    return sorted(items, key=lambda x: (-float(x.get("score") or 0), str(x.get("id", ""))))


def evaluate_turn(body: dict[str, Any]) -> dict[str, Any]:
    text = (body.get("assistant_text") or "").strip()
    if body.get("failed"):
        return {"accept": False, "retry_suggested": True, "reason": "turn failed"}
    if len(text) < 4:
        return {"accept": False, "retry_suggested": False, "reason": "reply too short"}
    return {"accept": True, "retry_suggested": False, "reason": ""}


class Handler(BaseHTTPRequestHandler):
    server_version = "GeeGooAdvisor/1.0"

    def log_message(self, fmt: str, *args: Any) -> None:
        return

    def _json(self, code: int, payload: dict[str, Any]) -> None:
        raw = json.dumps(payload, ensure_ascii=False).encode("utf-8")
        self.send_response(code)
        self.send_header("Content-Type", "application/json; charset=utf-8")
        self.send_header("Content-Length", str(len(raw)))
        self.end_headers()
        self.wfile.write(raw)

    def do_GET(self) -> None:
        if self.path.rstrip("/") == "/health":
            self._json(200, {"status": "ok"})
            return
        self._json(404, {"error": "not found"})

    def do_POST(self) -> None:
        length = int(self.headers.get("Content-Length", "0"))
        body = self.rfile.read(length) if length else b"{}"
        try:
            data = json.loads(body.decode("utf-8") or "{}")
        except json.JSONDecodeError:
            self._json(400, {"error": "invalid json"})
            return

        if self.path == "/v1/advisor/rank":
            items = data.get("items") or []
            if not isinstance(items, list):
                self._json(400, {"error": "items must be a list"})
                return
            self._json(200, {"items": rank_items(items)})
            return

        if self.path == "/v1/advisor/evaluate":
            if not isinstance(data, dict):
                self._json(400, {"error": "body must be object"})
                return
            self._json(200, evaluate_turn(data))
            return

        self._json(404, {"error": "not found"})


def main() -> None:
    host = os.environ.get("GEEGOO_ADVISOR_HOST", "127.0.0.1")
    port = int(os.environ.get("GEEGOO_ADVISOR_PORT", "3410"))
    httpd = ThreadingHTTPServer((host, port), Handler)
    print(f"advisor listening on http://{host}:{port}", flush=True)
    httpd.serve_forever()


if __name__ == "__main__":
    main()
