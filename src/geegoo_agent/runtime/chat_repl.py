"""Interactive CLI REPL for ``geegoo chat``."""

from __future__ import annotations

import json
import sys
from dataclasses import dataclass, field
from pathlib import Path
from typing import TextIO

from geegoo_agent.cli_meta import CLI_NAME
from geegoo_agent.exceptions import ConfigError
from geegoo_agent.llm.presets import (
    PROVIDER_PRESETS,
    current_model,
    list_provider_models,
    model_supports_thinking,
    pick_model,
    resolve_thinking_enabled,
)
from geegoo_agent.runtime.app import GeeGooApp
from geegoo_agent.runtime.chat_session import ChatSession, ChatSessionStore, StepRecord
from geegoo_agent.runtime.chat_tools import CHAT_SYSTEM_PROMPT, ON_DEMAND_CHAT_TOOLS
from geegoo_agent.tools.domains import format_tools_listing
from geegoo_agent.runtime.chat_progress import make_progress_writer
from geegoo_agent.paths import default_install_dir
from geegoo_agent.runtime.chat_banner import api_hosts_from_config
from geegoo_agent.runtime.chat_commands import build_help_text
from geegoo_agent.runtime.chat_input import read_chat_line
from geegoo_agent.runtime.chat_ui import ChatUI
from geegoo_agent.runtime.react_loop import ReActLoop, require_llm_gateway
from geegoo_agent.tools.bootstrap import register_all_tools
from geegoo_agent.tools.registry import ToolRegistry
from geegoo_agent.llm.types import Message

HELP_TEXT = build_help_text()


@dataclass
class ChatRepl:
    app: GeeGooApp
    session: ChatSession
    session_store: ChatSessionStore
    registry: ToolRegistry
    loop: ReActLoop
    dry_run: bool = False
    config_path: Path | None = None
    verbose: bool = True
    ui: ChatUI | None = None
    stdin: TextIO = field(default_factory=lambda: sys.stdin)
    stdout: TextIO = field(default_factory=lambda: sys.stdout)

    @classmethod
    def from_app(
        cls,
        app: GeeGooApp,
        *,
        session_id: str | None = None,
        dry_run: bool = False,
        config_path: str | Path | None = None,
    ) -> ChatRepl:
        store = ChatSessionStore(app.state_store)
        if session_id:
            session = store.load(session_id)
            if session is None:
                raise ConfigError(f"chat session not found: {session_id}")
        else:
            session = store.create()
            session.append_message(Message(role="system", content=CHAT_SYSTEM_PROMPT))
            store.save(session)

        registry = register_all_tools(ToolRegistry(), tool_filter=ON_DEMAND_CHAT_TOOLS)
        gateway = require_llm_gateway(app.llm_gateway)
        cfg_path = Path(config_path).resolve() if config_path else None
        repl = cls(
            app=app,
            session=session,
            session_store=store,
            registry=registry,
            loop=ReActLoop(gateway, app.executor),  # progress wired after repl exists
            dry_run=dry_run or app.config.dry_run,
            config_path=cfg_path,
        )
        repl.ui = ChatUI(repl.stdout)
        repl._attach_progress()
        return repl

    def _attach_progress(self) -> None:
        ui = self.ui or ChatUI(self.stdout)
        self.ui = ui
        self.loop.set_progress(make_progress_writer(self.stdout, enabled=self.verbose, ui=ui))

    def _ctx(self):
        ctx = self.app._tool_context(self.session.id)
        ctx.dry_run = self.dry_run
        return ctx

    def run(self) -> int:
        ui = self.ui or ChatUI(self.stdout)
        self.ui = ui
        provider = self.app.config.llm.provider
        model = current_model(provider, self.app.config.llm.model or None)
        preset = PROVIDER_PRESETS[provider]
        thinking = resolve_thinking_enabled(
            provider,
            model,
            thinking=self.app.config.llm.thinking,
        )
        ui.print_banner(
            session_id=self.session.id,
            provider=preset.label,
            model=model,
            registry=self.registry,
            thinking=thinking,
            dry_run=self.dry_run,
            workspace=self.app.config.workspace_root,
            install_dir=default_install_dir(),
            project_root=self.app.project_root,
            api_hosts=api_hosts_from_config(self.app.config),
        )
        while True:
            try:
                line = self._read_line()
            except (EOFError, KeyboardInterrupt):
                ui.print_info("\n再见。")
                return 0
            if not line.strip():
                continue
            if line.startswith("/"):
                if self._handle_slash(line):
                    return 0
                continue
            self._handle_user_message(line)

    def _read_line(self) -> str:
        ui = self.ui or ChatUI(self.stdout)
        return read_chat_line(plain=ui.plain, stdin=self.stdin, stdout=self.stdout)

    def _handle_slash(self, line: str) -> bool:
        parts = line.strip().split()
        cmd = parts[0].lower()
        args = parts[1:]

        if cmd in {"/exit", "/quit"}:
            self.session.status = "closed"
            self.session_store.save(self.session)
            (self.ui or ChatUI(self.stdout)).print_info("会话已保存。")
            return True
        if cmd == "/help":
            (self.ui or ChatUI(self.stdout)).print_help(HELP_TEXT)
            return False
        if cmd == "/session":
            provider = self.app.config.llm.provider
            model = current_model(provider, self.app.config.llm.model or None)
            preset = PROVIDER_PRESETS[provider]
            (self.ui or ChatUI(self.stdout)).print_info(
                f"session={self.session.id} messages={len(self.session.messages)} "
                f"steps={len(self.session.step_records)} dry_run={self.dry_run} "
                f"llm={preset.label}/{model} verbose={self.verbose} "
                f"think={resolve_thinking_enabled(provider, model, thinking=self.app.config.llm.thinking)}"
            )
            return False
        if cmd == "/think":
            self._handle_think(args)
            return False
        if cmd == "/model":
            if args:
                self._set_model(args[0])
            else:
                self._print_models()
            return False
        if cmd == "/tools":
            descriptions = {name: self.registry.get(name).description for name in self.registry.list_names()}
            self.stdout.write(format_tools_listing(self.registry.list_names(), descriptions) + "\n")
            return False
        if cmd == "/trace":
            limit = int(args[0]) if args else 10
            self._print_trace(limit)
            return False
        if cmd == "/flow":
            limit = int(args[0]) if args else 15
            self._print_flow(limit)
            return False
        if cmd == "/dry-run":
            if not args or args[0] not in {"on", "off"}:
                self.stdout.write("用法: /dry-run on|off\n")
                return False
            self.dry_run = args[0] == "on"
            self.stdout.write(f"dry_run={self.dry_run}\n")
            return False
        if cmd == "/verbose":
            if not args or args[0] not in {"on", "off"}:
                self.stdout.write("用法: /verbose on|off\n")
                return False
            self.verbose = args[0] == "on"
            self._attach_progress()
            self.stdout.write(f"verbose={self.verbose}\n")
            return False
        if cmd == "/run":
            self._run_workflow(args)
            return False

        self.stdout.write(f"未知命令: {cmd}，输入 /help\n")
        return False

    def _sync_chat_system_prompt(self) -> None:
        activity = self.session.tool_activity_summary()
        content = CHAT_SYSTEM_PROMPT
        if activity:
            content = f"{CHAT_SYSTEM_PROMPT}\n\n本会话 Tool 活动：\n{activity}"
        if self.session.messages and self.session.messages[0].get("role") == "system":
            self.session.messages[0]["content"] = content
        else:
            self.session.messages.insert(
                0,
                {"role": "system", "content": content},
            )

    def _handle_user_message(self, text: str) -> None:
        ui = self.ui or ChatUI(self.stdout)
        ui.print_user(text)
        self._sync_chat_system_prompt()

        schemas = self.registry.schemas(mode="chat")
        result = self.loop.run_turn(self.session, text, self._ctx(), schemas)
        for record in result.step_records:
            self.session.add_step_record(record)
        self.session_store.save(self.session)

        ui.print_assistant(result.assistant_text)
        provider = self.app.config.llm.provider
        model = current_model(provider, self.app.config.llm.model or None)
        thinking = resolve_thinking_enabled(
            provider,
            model,
            thinking=self.app.config.llm.thinking,
        )
        ui.print_turn_footer(
            model=model,
            thinking=thinking,
            dry_run=self.dry_run,
            steps=len(self.session.step_records),
        )

    def _print_trace(self, limit: int) -> None:
        records = self.session.step_records[-limit:]
        if not records:
            self.stdout.write("（暂无步骤记录）\n")
            return
        for rec in records:
            self._print_step(rec)

    def _print_step(self, rec: StepRecord) -> None:
        if rec.kind == "tool":
            self.stdout.write(
                f"  #{rec.step} [{rec.kind}] {rec.tool_name} {rec.tool_status}: {rec.summary[:100]}\n"
            )
        else:
            self.stdout.write(f"  #{rec.step} [{rec.kind}] {rec.summary[:100]}\n")

    def _print_flow(self, limit: int) -> None:
        history = self.app.event_bus.history[-limit:]
        if not history:
            self.stdout.write("（暂无事件）\n")
            return
        for event, payload in history:
            detail = ", ".join(f"{k}={v}" for k, v in payload.items())
            self.stdout.write(f"  {event}: {detail}\n")

    def _print_models(self) -> None:
        provider = self.app.config.llm.provider
        preset = PROVIDER_PRESETS[provider]
        active = current_model(provider, self.app.config.llm.model or None)
        self.stdout.write(f"当前: {preset.label} / {active}\n")
        models = list_provider_models(provider)
        if not models:
            self.stdout.write("（该提供商无预设列表，可用 /model <model_id> 手动指定）\n")
            return
        for index, (model_id, desc) in enumerate(models, start=1):
            mark = " *" if model_id == active else ""
            self.stdout.write(f"  {index}. {model_id} — {desc}{mark}\n")
        self.stdout.write("切换: /model <序号> 或 /model <model_id>\n")

    def _handle_think(self, args: list[str]) -> None:
        ui = self.ui or ChatUI(self.stdout)
        provider = self.app.config.llm.provider
        model = current_model(provider, self.app.config.llm.model or None)
        if not model_supports_thinking(provider, model):
            ui.print_error("当前模型不支持思考模式，请 /model deepseek-v4-pro 或 v4-flash")
            return
        if not args or args[0] not in {"on", "off", "auto"}:
            active = resolve_thinking_enabled(provider, model, thinking=self.app.config.llm.thinking)
            ui.print_info(f"思考模式: {active}（用法: /think on|off|auto）")
            return
        enabled: bool | None
        if args[0] == "auto":
            enabled = None
        else:
            enabled = args[0] == "on"
        try:
            active = self.app.set_llm_thinking(enabled)
            self.loop.gateway = require_llm_gateway(self.app.llm_gateway)
            self._persist_thinking(enabled)
            ui.print_info(f"思考模式已设为: {active}")
        except ConfigError as exc:
            ui.print_error(str(exc))

    def _persist_thinking(self, enabled: bool | None) -> None:
        if self.config_path is None or not self.config_path.is_file():
            return
        raw = json.loads(self.config_path.read_text(encoding="utf-8"))
        llm = raw.setdefault("llm", {})
        if enabled is None:
            llm.pop("thinking", None)
        else:
            llm["thinking"] = enabled
        self.config_path.write_text(
            json.dumps(raw, indent=2, ensure_ascii=False) + "\n",
            encoding="utf-8",
        )

    def _set_model(self, choice: str) -> None:
        provider = self.app.config.llm.provider
        try:
            resolved = pick_model(provider, choice, current=self.app.config.llm.model or None)
            applied = self.app.set_llm_model(resolved)
            self.loop.gateway = require_llm_gateway(self.app.llm_gateway)
            self._persist_model(applied)
            preset = PROVIDER_PRESETS[provider]
            self.stdout.write(f"已切换模型: {preset.label} / {applied}\n")
        except (ConfigError, ValueError) as exc:
            self.stdout.write(f"切换失败: {exc}\n")

    def _persist_model(self, model: str) -> None:
        if self.config_path is None or not self.config_path.is_file():
            return
        raw = json.loads(self.config_path.read_text(encoding="utf-8"))
        llm = raw.setdefault("llm", {})
        llm["model"] = model
        self.config_path.write_text(
            json.dumps(raw, indent=2, ensure_ascii=False) + "\n",
            encoding="utf-8",
        )

    def _run_workflow(self, args: list[str]) -> None:
        skill = args[0] if args else "pre_market"
        self.stdout.write(f"启动 workflow: {skill} ...\n")
        prev = self.app.config.dry_run
        self.app.config = self.app.config.model_copy(update={"dry_run": self.dry_run})
        try:
            result = self.app.run_skill(skill)
        finally:
            self.app.config = self.app.config.model_copy(update={"dry_run": prev})

        self.stdout.write(
            f"workflow 完成: session={result.session_id} status={result.status}\n"
        )
        if result.last_error:
            self.stdout.write(f"  error: {result.last_error}\n")
        log_dir = self.app.config.workspace_root
        self.stdout.write(f"  查看 execution-log: {log_dir}\n")
        self.stdout.write("  使用 /flow 查看本次 workflow 触发的 Tool 事件\n")
