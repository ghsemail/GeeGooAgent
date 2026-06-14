"""Slash commands for ``geegoo chat`` + prompt_toolkit completion."""

from __future__ import annotations

from dataclasses import dataclass

from geegoo_agent.cli_meta import CLI_NAME


@dataclass(frozen=True)
class SlashCommand:
    command: str
    description: str


SLASH_COMMANDS: tuple[SlashCommand, ...] = (
    SlashCommand("/help", "显示帮助"),
    SlashCommand("/exit", "退出并保存会话"),
    SlashCommand("/quit", "退出并保存会话"),
    SlashCommand("/session", "当前会话 ID 与消息数"),
    SlashCommand("/tools", "列出可用 Tool"),
    SlashCommand("/trace", "最近执行步骤（可加数量）"),
    SlashCommand("/flow", "最近事件总线记录（可加数量）"),
    SlashCommand("/run pre_market", "运行盘前 workflow"),
    SlashCommand("/dry-run on", "开启 dry-run（跳过写 API）"),
    SlashCommand("/dry-run off", "关闭 dry-run"),
    SlashCommand("/model", "列出或切换 LLM 模型"),
    SlashCommand("/verbose on", "显示思考与 Tool 过程"),
    SlashCommand("/verbose off", "隐藏思考与 Tool 过程"),
    SlashCommand("/think on", "开启 DeepSeek 思考模式"),
    SlashCommand("/think off", "关闭 DeepSeek 思考模式"),
)


def build_help_text() -> str:
    lines = [
        f"{CLI_NAME} chat — 与 GeeGoo Agent 对话，并查看 Tool / workflow 轨迹",
        "",
        "对话：直接输入问题，例如「分析一下腾讯」",
        "",
        "斜杠命令（输入 / 可自动补全）：",
    ]
    seen: set[str] = set()
    for item in SLASH_COMMANDS:
        if item.command in seen:
            continue
        seen.add(item.command)
        lines.append(f"  {item.command:<18} {item.description}")
    return "\n".join(lines) + "\n"


def slash_command_completer():
    """Build a prompt_toolkit completer for slash commands."""
    from prompt_toolkit.completion import Completer, Completion

    entries = list(SLASH_COMMANDS)

    class _SlashCompleter(Completer):
        def get_completions(self, document, complete_event):
            text = document.text_before_cursor
            stripped = text.lstrip()
            if not stripped.startswith("/"):
                return
            prefix = stripped
            for item in entries:
                if item.command.startswith(prefix):
                    yield Completion(
                        item.command,
                        start_position=-len(prefix),
                        display_meta=item.description,
                    )

    return _SlashCompleter()
