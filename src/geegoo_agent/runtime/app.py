"""Application wiring for CLI and workflows."""

from __future__ import annotations

from dataclasses import dataclass
from pathlib import Path

from geegoo_agent.clients.geegoo_bot import GeeGooBotClient
from geegoo_agent.clients.market import MarketClient
from geegoo_agent.config import AppConfig, load_config
from geegoo_agent.exceptions import ConfigError
from geegoo_agent.infra.checkpoint import CheckpointManager
from geegoo_agent.infra.events import InProcessEventBus
from geegoo_agent.infra.sandbox import NetworkPolicy
from geegoo_agent.infra.secrets import ConfigSecrets
from geegoo_agent.infra.state_store import FileStateStore
from geegoo_agent.llm.cost import CostManager
from geegoo_agent.llm.gateway import GatewayConfig, ModelGateway
from geegoo_agent.llm.presets import build_llm_provider
from geegoo_agent.memory.models import PreMarketWorking
from geegoo_agent.memory.working import WorkingMemoryStore
from geegoo_agent.runtime.executor import Executor
from geegoo_agent.runtime.pre_market_workflow import (
    PRE_MARKET_PER_STOCK_STEPS,
    PRE_MARKET_PHASE_A_STEPS,
)
from geegoo_agent.runtime.session import SessionManager
from geegoo_agent.runtime.workflow import RunResult, WorkflowRunner
from geegoo_agent.supervisor.engine import SupervisorResult
from geegoo_agent.supervisor.pre_market import run_pre_market_supervisor
from geegoo_agent.tools.bootstrap import register_all_tools
from geegoo_agent.tools.registry import ToolRegistry
from geegoo_agent.tools.types import ToolCallRequest, ToolContext


@dataclass
class GeeGooApp:
    config: AppConfig
    secrets: ConfigSecrets
    state_store: FileStateStore
    event_bus: InProcessEventBus
    session_mgr: SessionManager
    working_store: WorkingMemoryStore
    checkpoint_mgr: CheckpointManager
    registry: ToolRegistry
    executor: Executor
    workflow: WorkflowRunner
    market_client: MarketClient
    geegoo_bot_client: GeeGooBotClient
    llm_gateway: ModelGateway | None
    project_root: Path

    @classmethod
    def from_config_path(cls, path: str, *, dry_run: bool | None = None) -> GeeGooApp:
        config = load_config(path)
        if dry_run is not None:
            config = config.model_copy(update={"dry_run": dry_run})
        return cls.from_config(config)

    @classmethod
    def from_config(cls, config: AppConfig) -> GeeGooApp:
        secrets = ConfigSecrets(config)
        store = FileStateStore(config.workspace_root)
        bus = InProcessEventBus()
        network = NetworkPolicy(config.sandbox.allowed_hosts)
        geegoo_bot_client = GeeGooBotClient(
            config.geegoo_url,
            secrets.get("geegoo_api_key"),
            network,
            retry_wait_seconds=0,
        )
        market_client = geegoo_bot_client
        project_root = Path(__file__).resolve().parents[3]
        registry = register_all_tools(ToolRegistry())
        executor = Executor(registry, bus)
        working_store = WorkingMemoryStore(store)
        checkpoint_mgr = CheckpointManager(store)
        llm_gateway = cls._build_llm_gateway(config, secrets)
        return cls(
            config=config,
            secrets=secrets,
            state_store=store,
            event_bus=bus,
            session_mgr=SessionManager(store),
            working_store=working_store,
            checkpoint_mgr=checkpoint_mgr,
            registry=registry,
            executor=executor,
            workflow=WorkflowRunner(executor, working_store, checkpoint_mgr, bus),
            market_client=market_client,
            geegoo_bot_client=geegoo_bot_client,
            llm_gateway=llm_gateway,
            project_root=project_root,
        )

    def set_llm_thinking(self, enabled: bool | None) -> bool:
        """Toggle DeepSeek thinking mode (None=auto). Rebuilds gateway."""
        from geegoo_agent.llm.presets import resolve_thinking_enabled, resolve_model

        resolved_model = resolve_model(self.config.llm.provider, self.config.llm.model or None)
        active = resolve_thinking_enabled(
            self.config.llm.provider,
            resolved_model,
            thinking=enabled,
        )
        llm = self.config.llm.model_copy(update={"thinking": enabled})
        self.config = self.config.model_copy(update={"llm": llm})
        gateway = self._build_llm_gateway(self.config, self.secrets)
        if gateway is None:
            raise ConfigError("LLM not configured — run geegoo setup")
        self.llm_gateway = gateway
        return active

    def set_llm_model(self, model: str) -> str:
        """Switch active LLM model and rebuild gateway."""
        from geegoo_agent.llm.presets import pick_model, resolve_model

        resolved = pick_model(
            self.config.llm.provider,
            model,
            current=self.config.llm.model or None,
        )
        llm = self.config.llm.model_copy(update={"model": resolved})
        self.config = self.config.model_copy(update={"llm": llm})
        gateway = self._build_llm_gateway(self.config, self.secrets)
        if gateway is None:
            raise ConfigError("LLM not configured — run geegoo setup")
        self.llm_gateway = gateway
        return resolve_model(self.config.llm.provider, resolved)

    @staticmethod
    def _build_llm_gateway(config: AppConfig, secrets: ConfigSecrets) -> ModelGateway | None:
        token_key = secrets.get_optional("llm_token_key")
        if not token_key or token_key.startswith("REPLACE"):
            return None
        model = config.llm.model or None
        provider = build_llm_provider(
            config.llm.provider,
            token_key,
            model=model,
            thinking=config.llm.thinking,
            reasoning_effort=config.llm.reasoning_effort,
        )
        gw_config = GatewayConfig(
            temperature=config.llm.temperature,
            max_tokens=config.llm.max_tokens,
        )
        return ModelGateway(provider, CostManager(), gw_config)

    def _tool_context(self, session_id: str) -> ToolContext:
        return ToolContext(
            session_id=session_id,
            mcp_token=self.secrets.get("mcp_token"),
            dry_run=self.config.dry_run,
            workspace_root=self.config.workspace_root,
            market_client=self.market_client,
            geegoo_bot_client=self.geegoo_bot_client,
            working_store=self.working_store,
            state_store=self.state_store,
            project_root=self.project_root,
            feishu_webhook_url=self.config.feishu_webhook_url,
            event_bus=self.event_bus,
            llm_gateway=self.llm_gateway,
        )

    def _run_supervisor(self, session_id: str, working: PreMarketWorking) -> SupervisorResult:
        sup = run_pre_market_supervisor(
            working,
            workspace_root=self.config.workspace_root,
            project_root=self.project_root,
        )
        log_status = "ok" if sup.ok else "error"
        self.executor.execute(
            ToolCallRequest(
                name="write_execution_log",
                arguments={
                    "step": "supervisor",
                    "message": sup.summary[:500],
                    "status": log_status,
                },
            ),
            self._tool_context(session_id),
        )
        return sup

    def _finalize_run(self, session_id: str, result: RunResult) -> RunResult:
        if not result.ok:
            return result
        sup = self._run_supervisor(session_id, result.working)
        if sup.ok:
            return result
        return RunResult(
            session_id=result.session_id,
            status="failed",
            working=result.working,
            last_error=sup.summary,
        )

    def run_skill(self, skill_name: str) -> RunResult:
        if skill_name != "pre_market":
            raise ConfigError(f"unsupported skill in Step 7: {skill_name}")
        session = self.session_mgr.create(skill_name)
        working = self.working_store.create(session.id, skill=skill_name)
        self.event_bus.emit("RunStarted", {"session_id": session.id, "skill": skill_name})
        result = self.workflow.run(
            session,
            PRE_MARKET_PHASE_A_STEPS,
            self._tool_context(session.id),
            working,
            per_stock_steps=PRE_MARKET_PER_STOCK_STEPS,
        )
        result = self._finalize_run(session.id, result)
        self.session_mgr.save(session)
        return result

    def resume_session(self, session_id: str) -> RunResult:
        session = self.session_mgr.load(session_id)
        if session is None:
            raise ConfigError(f"session not found: {session_id}")
        checkpoint = self.checkpoint_mgr.load_latest(session_id)
        if checkpoint is None:
            raise ConfigError(f"no checkpoint for session: {session_id}")
        working = self.working_store.load(session_id)
        if working is None:
            working = PreMarketWorking.model_validate(self.checkpoint_mgr.load_working(checkpoint))
        start_index = checkpoint.step
        self.event_bus.emit("RunStarted", {"session_id": session.id, "skill": session.skill_name})
        result = self.workflow.run(
            session,
            PRE_MARKET_PHASE_A_STEPS,
            self._tool_context(session.id),
            working,
            per_stock_steps=PRE_MARKET_PER_STOCK_STEPS,
            start_index=start_index,
        )
        result = self._finalize_run(session.id, result)
        self.session_mgr.save(session)
        return result

    def close(self) -> None:
        self.market_client.close()
        self.geegoo_bot_client.close()
