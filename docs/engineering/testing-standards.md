# 测试规范

> **每执行一步（Step 0–15）都必须完成本步「测试交付物」**，否则该步视为未完成。  
> 配套：[coding-standards.md](./coding-standards.md)、[cursor-workflow.md](./cursor-workflow.md)

---

## 1. 测试原则

| # | 原则 |
|---|------|
| 1 | **不访问生产 API**：单测/集成测默认 mock；仅 Step 15 冒烟可打真机 |
| 2 | **不依赖 LLM**：E2E 用 fixture 固定 LLM 响应；单独 `tests/llm/` 可选、手动跑 |
| 3 | **可重复**：测试用 `tmp_path`；不读写仓库外路径 |
| 4 | **快**：全量 `pytest` 目标 <30s（E2E 除外）；慢的标 `@pytest.mark.slow` |
| 5 | **失败可定位**：断言带清晰 message；步骤名出现在断言里 |
| 6 | **先写失败用例再实现**（推荐）：至少覆盖 happy path + 1 失败 path |

---

## 2. 测试金字塔与覆盖率

```text
        / E2E \           1–3 个场景，dry-run 全链路
       / 集成  \          每 Client、每 Workflow 阶段
      /  单元   \         每个模块、映射、校验
```

| 层级 | 目录 | 覆盖率目标（累计） |
|------|------|-------------------|
| 单元 | `tests/unit/` | 核心模块行覆盖 ≥80% |
| 集成 | `tests/integration/` | 每个 Client 方法 ≥1 用例 |
| E2E | `tests/e2e/` | Phase 1 至少 1 条完整 dry-run |

**每步最低要求**：新增代码**必须有**对应层级测试；禁止「先提交代码后补测」跨 Step。

```bash
# 每步结束必跑
pytest -q
pytest --cov=geegoo --cov-fail-under=0   # Step 0 起记录；Step 5 起建议 ≥60%；Step 14 起 ≥70%
```

---

## 3. 测试基础设施

### 3.1 目录结构

```text
tests/
├── conftest.py              # 全局 fixtures
├── fixtures/
│   ├── geegoo/                # JSON：API 响应样例
│   ├── llm/                 # 固定 LLM 返回
│   └── workflow/            # working 快照
├── unit/
├── integration/
└── e2e/
```

### 3.2 conftest.py 必备 fixtures

```python
@pytest.fixture
def tmp_output_dir(tmp_path: Path) -> Path: ...

@pytest.fixture
def sample_config(tmp_output_dir: Path) -> AppConfig: ...

@pytest.fixture
def mock_mcp_token() -> str:
    return "test-mcp-token"
```

### 3.3 Fixture 数据

- GeeGoo 响应放在 `tests/fixtures/geegoo/check_trading_day_ok.json`
- 从 skill 文档或真实响应**脱敏**复制，勿在测试里手写简化到失真

---

## 4. 分层测试规范

### 4.1 单元测试（unit）

**测什么**：纯逻辑、无 IO。

| 模块 | 必测用例 |
|------|----------|
| `state_store` | save→load 往返；不存在 key 返回 None；list_keys 前缀 |
| `checkpoint` | 同 step 覆盖写；load 最新；损坏 JSON 抛错 |
| `event_bus` | 注册 handler 被调用；handler 异常不影响其他 |
| `sandbox` | 合法路径通过；`../` 拒绝；非 allowlist host 拒绝 |
| `mappings` | attitude 三值 + 未知值 |
| `PreMarketReportCreate` | 缺 bot_id 校验失败；非法 enum 失败 |
| `compaction` | 超长字符串截断保留头尾 |

**写法**：

```python
def test_attitude_bullish_maps_to_long():
    assert attitude_to_result("bullish") == "long"
```

### 4.2 集成测试（integration）

**测什么**：模块协作 + HTTP mock。

使用 `pytest-httpx`：

```python
def test_check_trading_day_ok(httpx_mock, market_client):
    httpx_mock.add_response(
        url="http://test:5700/checkTradingDay",
        json={"code": 100, "data": {"is_trading_day": True}},
    )
    r = market_client.check_trading_day("token", "00700.HK")
    assert r.is_trading_day is True
```

| 模块 | 必测用例 |
|------|----------|
| 每个 Client 方法 | HTTP 200 + code 100；code 102 抛 ClientError；超时重试 |
| `ToolRegistry` | 注册→执行→ToolResult；未知 tool 报错 |
| `Gateway` | mock SDK 返回 tool_calls；重试后成功 |
| `Workflow` 阶段 A | mock 全部依赖，断言 working.indices_done |
| `Workflow` 阶段 B | 单股 happy path；404 attitude |

### 4.3 E2E（e2e）

**一条命令跑通**：

```python
@pytest.mark.e2e
def test_pre_market_dry_run_full(tmp_output_dir, httpx_mock, mock_llm):
    exit_code = run_cli(["run", "pre_market", "--dry-run", "--config", ...])
    assert exit_code == 0
    assert (tmp_output_dir / today / "execution-log.md").exists()
    assert supervisor_check(tmp_output_dir).ok
```

E2E 断言清单：

- [ ] `execution-log.md` 含所有步骤名
- [ ] `checkpoints/` 最后 step 存在
- [ ] 每股有 `{code}-premarket.md`（mock 股票数）
- [ ] dry-run 下无真实 POST createPreMarketReport（httpx 记录断言）
- [ ] `working` 终态 `phase=done`

---

## 5. 分 Step 测试交付物（强制）

> 与 [cursor-workflow.md](./cursor-workflow.md) 一一对应。  
> **该 Step 合并前，下表对应行必须全部打勾。**

| Step | 必交付测试文件 | 最少用例数 | 必覆盖场景 |
|------|----------------|------------|------------|
| **0** | `tests/conftest.py`（空 pytest 绿） | 0 | `pip install -e ".[dev]"` 成功 |
| **1** | `test_state_store.py`, `test_checkpoint.py`, `test_event_bus.py` | 各 ≥3 | 往返、前缀列表、handler 异常隔离 |
| **2** | `test_sandbox.py`, `test_config.py` | 各 ≥4 | 路径越界、host 拒绝、缺 config |
| **3** | `test_gateway.py`, `test_openai_provider.py` | ≥5 | tool_calls 解析、重试、失败抛错 |
| **4** | `test_market_client.py` | ≥6 | 3 API happy；code≠100；401 |
| **5** | `test_tool_registry.py`, `test_tools_perceive.py` | ≥5 | 执行 3 Tool；dry_run 不写 |
| **6** | `test_working_memory.py` | ≥4 | save/load；字段默认；merge |
| **7** | `test_workflow_runner.py`, `test_cli.py` | ≥5 | 1 假步骤；resume；--dry-run |
| **8** | `test_skill_loader.py` 或 manifest 校验 | ≥2 | manifest tools 列表完整 |
| **9** | `test_*_client.py` 补全, `test_tools_*.py` | ≥20 累计 | 每 MVP Tool ≥1；report schema |
| **10** | `test_workflow_phase_a.py` | ≥4 | 非交易日终止；5 指数完成标记 |
| **11** | `test_workflow_phase_b.py` | ≥6 | 单股全流程；404 attitude；幂等 skip |
| **12** | `test_llm_tasks.py` | ≥4 | mock LLM；pydantic 校验失败 |
| **13** | `test_supervisor.py` | ≥3 | 缺 report 失败；全通过 |
| **14** | `test_pre_market_dry_run.py` | ≥1 完整 | 全链路；见 §4.3 清单 |
| **15** | `tests/smoke/README.md` + 手动清单 | — | 真机 checkTradingDay（文档记录） |

---

## 6. Mock 规范

### 6.1 HTTP

- 使用 `pytest-httpx`；`base_url` 测试中用 `http://test:5700`
- 每个接口至少 2 个 fixture：`ok` + `error`

### 6.2 LLM

```python
@pytest.fixture
def mock_gateway(monkeypatch):
    def fake_chat(messages, tools):
        return LLMResponse(content=None, tool_calls=[], usage=...)
    monkeypatch.setattr(gateway, "chat", fake_chat)
```

- 结构化任务：fixture 返回**固定 JSON 字符串**
- 禁止 E2E 调真实 OpenAI

### 6.3 时间

```python
@pytest.fixture
def frozen_today(monkeypatch):
    monkeypatch.setattr("geegoo.utils.dates", "today", lambda: date(2026, 6, 5))
```

报告路径依赖日期时必须 freeze。

### 6.4 新闻脚本

- 单元测 mock `subprocess` 或抽 `NewsFetcher` 接口
- 集成测用 fixture JSON 代替真实 RSS

---

## 7. 断言与命名

### 7.1 测试函数命名

```text
test_<模块>_<场景>_<预期>
```

示例：`test_market_client_check_trading_day_returns_false_on_holiday`

### 7.2 应用 pytest.mark

```python
@pytest.mark.unit
@pytest.mark.integration
@pytest.mark.e2e
@pytest.mark.slow
```

默认 CI 跑：`pytest -m "not slow"`。

### 7.3 禁止

- `assert True`
- 无断言的测试
- `time.sleep` 等待（用 mock 或 pytest-timeout）
- 测试顺序依赖（用 fixture 隔离）

---

## 8. 每步验收命令（复制执行）

```bash
# === Step 完成门禁 ===
pip install -e ".[dev]"

# 1. 全量测试
pytest -q

# 2. 仅本步新增测试（示例 Step 4）
pytest -q tests/integration/test_market_client.py -v

# 3. 覆盖率（Step 5 起）
pytest --cov=geegoo --cov-report=term-missing --cov-fail-under=60

# 4. 静态检查
ruff check src tests
ruff format --check src tests

# 5. 类型（Step 9 起建议）
mypy src/geegoo --ignore-missing-imports
```

**任一项失败 = 本 Step 未完成。**

---

## 9. Cursor Agent 测试指令（每步附加）

在每个 Step 的 Cursor 提示末尾追加：

```markdown
## 测试要求（必做）
- 按 docs/engineering/testing-standards.md §5 本 Step 行实现测试
- 禁止访问真实 GeeGoo API / 真实 LLM
- 交付后运行：pytest -q && ruff check src tests
- 列出：新增测试文件、用例数、覆盖的场景
```

---

## 10. 回归与冒烟

| 时机 | 动作 |
|------|------|
| 每 Step 结束 | `pytest -q` 全绿 |
| Step 14 后 | 固定 `tests/e2e/` 为回归套件 |
| Step 15 真机 | 手动冒烟表（非交易日 / 单股 dry-run / 交易日 1 股） |
| 改 Client | 重跑对应 `test_*_client.py` + e2e |

### 真机冒烟表（Step 15，不自动化）

| # | 操作 | 预期 |
|---|------|------|
| 1 | `geegoo-agent run pre_market --dry-run` | exit 0 |
| 2 | 非交易日 `check_trading_day` | 跳过，log 有记录 |
| 3 | 交易日 1 股（可限 `--stocks 00700.HK`） | md + API 有记录 |
| 4 | 杀进程后 `resume` | 从 checkpoint 继续 |

---

## 11. CI 建议（可选，Step 14 后）

```yaml
# .github/workflows/test.yml 示意
- run: pip install -e ".[dev]"
- run: pytest -q --cov=geegoo --cov-fail-under=70
- run: ruff check src tests
```

---

## 12. 缺陷与测试同步

| 事件 | 要求 |
|------|------|
| 修 bug | 先写失败用例，再修代码 |
| 改 API 字段 | 同步改 fixture JSON + schema 测试 |
| 改 workflow 步骤 | 同步改 `test_workflow_*.py` + e2e |

---

## 13. 检查清单（Step 合并前自评）

```markdown
- [ ] testing-standards.md §5 本 Step 行已全部实现
- [ ] pytest -q 全绿
- [ ] 无真实网络请求（除标记 slow/manual）
- [ ] 新增代码有类型注解
- [ ] fixture 已脱敏
- [ ] 已向用户报告：测试文件列表 + 用例数
```
