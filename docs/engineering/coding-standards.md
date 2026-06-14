# 编码规范

> 适用于 `src/geegoo/` 与 `tests/`。与 [requirements.md](./requirements.md)、[testing-standards.md](./testing-standards.md) 配套执行。

---

## 1. 总则

| 规则 | 说明 |
|------|------|
| **测试先行或同步** | 新增/修改行为必须有对应 pytest；无测试的 PR 不合并 |
| **小模块** | 单文件 ≤400 行；超过则拆分 |
| **显式优于隐式** | 类型注解、Pydantic 模型、明确异常类型 |
| **无副作用的纯函数优先** | 映射、解析、校验抽成纯函数便于单测 |
| **配置外置** | 禁止硬编码 `mcp_token`、API Key、host |
| **不造框架** | 禁止 LangChain；HTTP 用 httpx，重试用 tenacity |

---

## 2. 目录与命名

### 2.1 包结构

```text
src/geegoo/
  infra/       # L0：无业务语义
  llm/         # L1
  clients/     # L2 底层 HTTP
  tools/       # L2 Tool 实现
  memory/      # L3
  runtime/     # L4
  supervisor/  # 横切
  cli.py
  config.py
```

### 2.2 命名约定

| 对象 | 风格 | 示例 |
|------|------|------|
| 模块/文件 | `snake_case` | `state_store.py` |
| 类 | `PascalCase` | `FileStateStore` |
| 函数/方法 | `snake_case` | `check_trading_day` |
| 常量 | `UPPER_SNAKE` | `DEFAULT_TIMEOUT_SEC` |
| Tool 名（对外） | `snake_case` | `get_mcp_analysis` |
| GeeGoo HTTP 路径 | 保持服务端 `camelCase` | `/checkTradingDay` |

### 2.3 导入顺序

```python
# 1. 标准库
# 2. 第三方
# 3. geegoo.*
```

层间导入只允许向下（见 requirements §4）。

---

## 3. 类型与数据模型

### 3.1 必须使用 Pydantic v2 的场景

- Tool 入参 / 出参 schema
- `create_pre_market_report` 请求体
- `PreMarketWorking` 等工作状态
- LLM 结构化输出（`llm_tasks`）

```python
from pydantic import BaseModel, Field

class PreMarketReportCreate(BaseModel):
    code: str
    stock_name: str
    bot_id: str = Field(min_length=1)
    result: Literal["long", "short", "neutral"]
    # ...
```

### 3.2 枚举与映射

业务映射（如 `bullish → long`）放在**独立模块** `geegoo/domain/mappings.py` 或 `tools/_mappings.py`，禁止散落在 Tool 内：

```python
def attitude_to_result(attitude: str) -> str:
    ...
```

必须有单测覆盖全部枚举值 + 未知值行为。

### 3.3 可选类型

- Python 3.11+：用 `str | None`，不用 `Optional[str]`（新代码）
- 对外 API 返回：优先明确字段，少用大段 `dict[str, Any]`

---

## 4. 错误处理

### 4.1 异常层次

```text
GeeGooAgentError（基类）
├── ConfigError
├── SandboxError
├── ClientError（HTTP / GeeGoo code != 100）
├── ToolValidationError（Pydantic / 业务校验）
├── ModelGatewayError
└── WorkflowError（步骤失败可 resume）
```

### 4.2 规则

| 场景 | 做法 |
|------|------|
| GeeGoo `code != 100` | `ClientError`，附带 `code`、`message` |
| `get_bot_yesterday_attitude` 404 | Tool 内转为 `neutral`，**不抛**到 Workflow |
| 非交易日 | Workflow 正常终止，`status=skipped`，写 log |
| 单股失败 | 记录错误，**继续**下一只；Supervisor 汇总 |
| Sandbox 拒绝 | `SandboxError`，不可重试 |
| LLM 失败 | Gateway 重试 3 次后 `ModelGatewayError` |

### 4.3 禁止

- 裸 `except:` 或吞掉异常
- 用返回值 `None` 代替异常表示「配置缺失」
- 在 Client 里 `print` 调试

---

## 5. HTTP Client（clients/）

```python
class MarketClient:
    def __init__(self, base_url: str, api_key: str, *, timeout: float = 60.0): ...

    def check_trading_day(self, mcp_token: str, code: str) -> CheckTradingDayResponse:
        ...
```

| 规则 | 要求 |
|------|------|
| Header | 每次请求 `Authorization: Bearer`、`Content-Type: application/json` |
| Body | `mcp_token` 由调用方传入，Client 不读 config |
| 重试 | `tenacity`：最多 3 次，间隔 5s，仅对超时/5xx |
| 响应 | 解析为 Pydantic；`code != 100` 抛 `ClientError` |
| 测试 | **禁止**单测访问真实 5700；用 `pytest-httpx` |

---

## 6. Tool 实现（tools/）

### 6.1 结构模板

```python
@registry.register("check_trading_day")
async def check_trading_day(ctx: ToolContext, params: CheckTradingDayParams) -> ToolResult:
    # 1. params 已由 registry 校验
    # 2. 调 client
    # 3. 返回 ToolResult(status="ok", summary="...", data={...})
```

### 6.2 规则

| 规则 | 说明 |
|------|------|
| 薄封装 | 单 Tool ≤50 行（不含 schema 定义） |
| summary | 给 LLM 看的摘要 ≤500 字；原始 JSON 放 `data` |
| dry_run | 写操作 Tool 必须检查 `ctx.dry_run`，跳过 HTTP |
| 日志 | 经 `ctx.logger` 或 EventBus，不 `print` |

---

## 7. Workflow（runtime/）

```python
class WorkflowStep(TypedDict):
    name: str
    handler: Callable[[WorkflowContext], Awaitable[StepResult]]
    checkpoint: bool  # 默认 True
```

| 规则 | 说明 |
|------|------|
| 步骤表 | 步骤名常量 `STEP_CHECK_TRADING_DAY = "check_trading_day"` |
| 幂等 | 每步前读 `working` 是否已完成，已完成则 skip |
| Checkpoint | `checkpoint=True` 的步骤结束后必须 `save` |
| LLM | 仅 `llm_tasks.py` 调 Gateway；Workflow 不直连 SDK |

---

## 8. 配置（config.py）

```python
@dataclass(frozen=True)
class AppConfig:
    base_url: str
    api_key: str
    mcp_token: str
    output_dir: Path
    dry_run: bool
    # ...
```

- 从 `config.json` 加载；文件不存在抛 `ConfigError`
- `config.example.json` 提交仓库；`config.json` 在 `.gitignore`
- LLM key 优先 `os.environ[api_key_env]`

---

## 9. 日志

```python
logger.info("tool_completed", tool="check_trading_day", latency_ms=120, status="ok")
```

- 结构化字段：`session_id`、`step`、`tool`、`code`（股票）
- 禁止日志打印完整 `mcp_token` / `api_key`（可打 `***` 或后四位）

---

## 10. 注释与文档字符串

| 写 | 不写 |
|----|------|
| 非显而易见的业务规则（404→neutral） | 复述函数名 |
| 公共类/模块 docstring（一行） | 大段架构说明（放 docs/） |
| 复杂算法步骤 | 被删掉代码的注释块 |

---

## 11. 依赖与工具链

### 11.1 pyproject.toml 开发依赖（建议）

```toml
[project.optional-dependencies]
dev = [
  "pytest>=8",
  "pytest-httpx>=0.30",
  "pytest-asyncio>=0.23",
  "pytest-cov>=5",
  "ruff>=0.4",
  "mypy>=1.10",
]
```

### 11.2 本地检查命令（每步必跑）

```bash
pytest -q --cov=geegoo --cov-report=term-missing
ruff check src tests
ruff format --check src tests
mypy src/geegoo   # Step 3 起逐步启用
```

---

## 12. Git 与审查

- 一步一提交（用户要求时）；message：`feat(step-N): 简述`
- 不提交：`config.json`、`data/`、`.env`
- Review 对照 [requirements.md §12 质量门禁](./requirements.md#12-质量门禁code-review-检查项)

---

## 13. 与测试规范关系

| 编码产出 | 必配测试 |
|----------|----------|
| 纯函数/映射 | `tests/unit/` |
| Client | `tests/integration/` + httpx mock |
| Tool | unit（mock client）+ 可选 integration |
| Workflow 步骤 | `tests/integration/test_workflow_*.py` |
| 全流程 | `tests/e2e/` dry-run |

详见 [testing-standards.md](./testing-standards.md)。
