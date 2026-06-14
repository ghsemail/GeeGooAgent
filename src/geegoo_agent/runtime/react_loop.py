"""ReAct loop for one chat turn (plan → act → observe)."""

from __future__ import annotations

import json
from dataclasses import dataclass
from datetime import UTC, datetime
from typing import Any, Callable

from geegoo_agent.exceptions import ConfigError, ModelGatewayError
from geegoo_agent.llm.gateway import ModelGateway
from geegoo_agent.llm.types import Message, ToolCall, ToolSchema
from geegoo_agent.runtime.chat_session import ChatSession, StepRecord
from geegoo_agent.runtime.executor import Executor
from geegoo_agent.tools.types import ToolCallRequest, ToolContext, ToolResult


@dataclass
class ChatTurnResult:
    assistant_text: str
    step_records: list[StepRecord]
    failed: bool = False
    error: str | None = None


def _tool_result_content(result: ToolResult) -> str:
    payload: dict = {"status": result.status, "summary": result.summary}
    if result.data:
        payload["data"] = result.data
    text = json.dumps(payload, ensure_ascii=False)
    return text[:6000]


ProgressFn = Callable[[str, dict[str, Any]], None]


class ReActLoop:
    def __init__(
        self,
        gateway: ModelGateway,
        executor: Executor,
        *,
        max_tool_rounds: int = 8,
        on_progress: ProgressFn | None = None,
    ) -> None:
        self._gateway = gateway
        self._executor = executor
        self._max_tool_rounds = max_tool_rounds
        self._on_progress = on_progress

    def set_progress(self, on_progress: ProgressFn | None) -> None:
        self._on_progress = on_progress

    def _emit(self, event: str, **data: Any) -> None:
        if self._on_progress is not None:
            self._on_progress(event, data)

    def run_turn(
        self,
        session: ChatSession,
        user_text: str,
        ctx: ToolContext,
        tool_schemas: list[ToolSchema],
    ) -> ChatTurnResult:
        session.append_message(Message(role="user", content=user_text))
        messages = session.to_llm_messages()
        records: list[StepRecord] = []
        self._emit("turn_start", user_text=user_text)

        try:
            for round_no in range(1, self._max_tool_rounds + 1):
                session.step_counter += 1
                step = session.step_counter
                self._emit("round_start", round=round_no, step=step)
                response = self._gateway.chat(
                    messages,
                    tool_schemas,
                    session_id=session.id,
                    step=step,
                )
                reasoning = (response.reasoning_content or "").strip()
                content_preview = (response.content or "").strip()
                tool_names = [call.name for call in response.tool_calls]
                plan_summary = reasoning[:300] or content_preview[:300]
                if not plan_summary and tool_names:
                    plan_summary = f"决策: 调用 {', '.join(tool_names)}"
                records.append(
                    StepRecord(
                        step=step,
                        timestamp=datetime.now(UTC).isoformat(),
                        kind="plan",
                        summary=plan_summary or "（无显式计划文本）",
                        tokens=response.usage.prompt_tokens + response.usage.completion_tokens,
                    )
                )
                self._emit(
                    "llm_plan",
                    step=step,
                    reasoning=reasoning,
                    content=content_preview,
                    tool_names=tool_names,
                )

                if not response.tool_calls:
                    text = (response.content or "").strip() or "（无文本回复）"
                    self._emit("reply_start", step=step)
                    assistant = Message(
                        role="assistant",
                        content=text,
                        reasoning_content=response.reasoning_content,
                    )
                    session.append_message(assistant)
                    records.append(
                        StepRecord(
                            step=step,
                            timestamp=datetime.now(UTC).isoformat(),
                            kind="reply",
                            summary=text[:300],
                            tokens=response.usage.completion_tokens,
                        )
                    )
                    return ChatTurnResult(assistant_text=text, step_records=records)

                assistant = Message(
                    role="assistant",
                    content=response.content,
                    tool_calls=response.tool_calls,
                    reasoning_content=response.reasoning_content,
                )
                session.append_message(assistant)
                messages.append(assistant)

                for call in response.tool_calls:
                    ctx.step = step
                    self._emit(
                        "tool_start",
                        step=step,
                        name=call.name,
                        arguments=call.arguments,
                    )
                    result = self._execute_tool(call, ctx)
                    self._emit(
                        "tool_done",
                        step=step,
                        name=call.name,
                        status=result.status,
                        summary=result.summary,
                    )
                    records.append(
                        StepRecord(
                            step=step,
                            timestamp=datetime.now(UTC).isoformat(),
                            kind="tool",
                            tool_name=call.name,
                            tool_status=result.status,
                            summary=result.summary[:300],
                        )
                    )
                    tool_message = Message(
                        role="tool",
                        content=_tool_result_content(result),
                        tool_call_id=call.id,
                    )
                    session.append_message(tool_message)
                    messages.append(tool_message)

            msg = "已达到单轮 Tool 调用上限，请缩小问题范围后重试。"
            self._emit("error", message=msg)
            return ChatTurnResult(
                assistant_text=msg,
                step_records=records,
                failed=True,
                error="max_tool_rounds",
            )
        except ModelGatewayError as exc:
            self._emit("error", message=f"模型调用失败: {exc}")
            return ChatTurnResult(
                assistant_text=f"模型调用失败: {exc}",
                step_records=records,
                failed=True,
                error=str(exc),
            )

    def _execute_tool(self, call: ToolCall, ctx: ToolContext) -> ToolResult:
        try:
            return self._executor.execute(
                ToolCallRequest(name=call.name, arguments=call.arguments),
                ctx,
            )
        except Exception as exc:
            return ToolResult(status="error", summary=str(exc), exit_code=1)


def require_llm_gateway(gateway: ModelGateway | None) -> ModelGateway:
    if gateway is None:
        raise ConfigError(
            "LLM 未配置。请先运行 geegoo setup 并填写 llm.token_key（OpenAI / DeepSeek / Minimax）。"
        )
    return gateway
