#!/usr/bin/env python3
"""Single-pool crypto diagnose_macro_overall E2E (mock JEV if no live data-api)."""

from __future__ import annotations

import json
import os
import sys
import time
from http.server import BaseHTTPRequestHandler, HTTPServer
from pathlib import Path
from threading import Thread
from urllib import error, request

_SCRIPT_DIR = Path(__file__).resolve().parent
if str(_SCRIPT_DIR) not in sys.path:
    sys.path.insert(0, str(_SCRIPT_DIR))

from macro_overall_diagnose_skill import build_macro_overall_skill_body  # noqa: E402

ARTIFACT = Path("/opt/cursor/artifacts/single_crypto_overall_skill_e2e.json")
CRYPTO_REASON = (
    "BTC 空头信号一致性强。日线呈 LH+LL 下行结构并出现 BOS，EMA13/26 斜率、MACD 柱及 Force Index 均偏弱，"
    "ADX 与 SAR 也给出空头信号。ETH 同样由模型判定为空头占优：日线趋势汇总为空头占优（ADX、SAR 均为空，net=-2），"
    "且 EMA13/26 斜率均为负、MACD 柱与 Force Index fi13 均为负，显示下行压力。"
    "需跟踪反弹风险：BTC RSI 25.35、KD 处于低位；ETH价格结构部分分歧，RSI与Stochastic提示超卖。"
)


class _MockJEV(BaseHTTPRequestHandler):
    def do_POST(self) -> None:
        if self.path.endswith("/v1/decision/choice"):
            body = json.dumps({
                "code": 0,
                "data": {
                    "model": "mock-jev",
                    "answers": {
                        "direction": {
                            "choice": "看空",
                            "probabilities": {"看多": 0.18, "看空": 0.67, "不明确": 0.15},
                            "confidence": 0.67,
                        }
                    },
                },
            }).encode()
        else:
            body = json.dumps({
                "code": 0,
                "data": {
                    "content": "## 整体结论\n风险偏好偏空（risk-off），置信 0.67。\n"
                    "1. 加密池空头结构一致\n2. 仅基于已参与池，不覆盖全资产",
                },
            }).encode()
        self.send_response(200)
        self.send_header("Content-Type", "application/json")
        self.end_headers()
        self.wfile.write(body)

    def log_message(self, *_args) -> None:
        return


def main() -> int:
    live_data = os.environ.get("GEEGOO_DATA_CN_TOKEN") or os.environ.get("DATA_API_KEY")
    agent_base = os.environ.get("GEEGOO_AGENT_OP_HTTP_URL", "http://127.0.0.1:3400")
    runtime_key = os.environ.get("GEEGOO_AGENT_RUNTIME_API_KEY", "test-runtime-key")
    mode = "live_jev" if live_data else "mock_jev"
    mock_srv: HTTPServer | None = None
    mock_thread: Thread | None = None

    report: dict = {
        "mode": mode,
        "orchestration_date": "2026-11-22",
        "pool_workflow": "crypto",
        "live_model_called": mode == "live_jev",
        "blocked_reason": None if live_data else "无 GeeGooData decision token；使用本地 mock JEV",
    }

    body = build_macro_overall_skill_body(
        orchestration_date="2026-11-22",
        pools=[{"pool_workflow": "crypto", "jev_direction": "short", "jev_reason": CRYPTO_REASON}],
    )
    report["request_body_redacted"] = body

    t0 = time.perf_counter()
    if mode == "mock_jev":
        mock_srv = HTTPServer(("127.0.0.1", 0), _MockJEV)
        port = mock_srv.server_address[1]
        mock_thread = Thread(target=mock_srv.serve_forever, daemon=True)
        mock_thread.start()
        os.environ["GEEGOO_DATA_HTTP_URL"] = f"http://127.0.0.1:{port}"
        os.environ["GEEGOO_DATA_CN_TOKEN"] = "mock-token"
        agent_base = os.environ.get("GEEGOO_AGENT_OP_HTTP_URL", "")
        if not agent_base.strip():
            report["note"] = "mock JEV only — agent runtime not configured; Go unit tests cover skill path"
            report["elapsed_ms"] = int((time.perf_counter() - t0) * 1000)
            ARTIFACT.parent.mkdir(parents=True, exist_ok=True)
            ARTIFACT.write_text(json.dumps(report, ensure_ascii=False, indent=2), encoding="utf-8")
            print(json.dumps(report, ensure_ascii=False, indent=2))
            return 0

    req = request.Request(
        f"{agent_base.rstrip('/')}/v1/skills/run",
        data=json.dumps(body).encode(),
        headers={
            "Content-Type": "application/json",
            "Authorization": f"Bearer {runtime_key}",
        },
        method="POST",
    )
    try:
        with request.urlopen(req, timeout=300) as resp:
            raw = resp.read().decode()
            report["http_status"] = resp.status
            report["response"] = json.loads(raw)
    except error.HTTPError as exc:
        report["http_status"] = exc.code
        report["response"] = exc.read().decode()[:2000]
    except Exception as exc:  # noqa: BLE001
        report["error"] = str(exc)
    report["elapsed_ms"] = int((time.perf_counter() - t0) * 1000)
    ARTIFACT.parent.mkdir(parents=True, exist_ok=True)
    ARTIFACT.write_text(json.dumps(report, ensure_ascii=False, indent=2), encoding="utf-8")
    print(json.dumps(report, ensure_ascii=False, indent=2))
    if mock_srv:
        mock_srv.shutdown()
    return 0 if report.get("response", {}).get("status") == "ok" else 1


if __name__ == "__main__":
    raise SystemExit(main())