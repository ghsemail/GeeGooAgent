# 代码仓库布局

与架构层一一对应的 `src/geegoo/` 结构：

```
GeeGooAgent/
├── docs/architecture/          # 本目录 — 设计蓝图
├── pyproject.toml
├── config.example.json
├── prompts/identity.md
├── rules/
├── skills/
│   ├── pre_market/             # L5 — MVP
│   ├── post_market/
│   ├── intraday/
│   ├── on_demand_analysis/
│   ├── strategy/
│   ├── bot_manager/
│   └── bundled/
├── references/
├── deploy/
└── src/geegoo/
    ├── cli.py                  # L5 入口
    ├── config.py
    ├── infra/                  # L0
    │   ├── events.py
    │   ├── scheduler.py
    │   ├── timer.py
    │   ├── checkpoint.py
    │   ├── state_store.py
    │   ├── sandbox/              # SandboxManager 六层
    │   │   ├── manager.py
    │   │   ├── policy.py
    │   │   ├── workspace.py
    │   │   ├── resource.py
    │   │   └── envelope.py
    │   ├── logging.py
    │   ├── tracing.py
    │   ├── secrets.py
    │   └── env.py
    ├── runtime/                # L4
    │   ├── agent_runtime.py
    │   ├── loop.py
    │   ├── session.py
    │   ├── skill_loader.py
    │   ├── context_builder.py
    │   └── triggers.py
    ├── memory/                 # L3
    │   ├── session.py
    │   ├── working.py
    │   ├── episodic.py
    │   └── compaction.py
    ├── tools/                  # L2
    │   ├── registry.py
    │   ├── perceive.py
    │   ├── analyze.py
    │   ├── decide.py
    │   ├── act.py
    │   └── meta.py
    ├── clients/                # L2 底层
    │   ├── base.py
    │   ├── market.py
    │   ├── geegoo_bot.py
    │   └── analysis.py
    ├── llm/                    # L1
    │   ├── gateway.py
    │   ├── cost.py
    │   ├── base.py
    │   ├── openai_provider.py
    │   └── anthropic_provider.py
    ├── subagents/              # L5 委派
    ├── supervisor/             # cross-cutting
    └── server/                 # Phase 3 webhook
```

## 依赖方向（只允许向下）

```text
cli → runtime → {memory, tools, llm/gateway}
tools → clients → HTTP
runtime → infra（所有层均可使用 infra 横切能力）
```

**禁止**：`clients` 直接调 `runtime`；`infra` 依赖 `tools`。
