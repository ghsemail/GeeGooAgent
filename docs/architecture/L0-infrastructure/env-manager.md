# L0 — Environment Manager

## 职责

运行环境配置：profile、路径、时区、venv——**非** Coding Agent 的 Dev Container。

## 接口

```python
@dataclass
class AppConfig:
    profile: Literal["dev", "prod"]
    output_dir: Path
    timezone: str
    geegoo_url: str
    base_url: str
    indices: dict[str, str]
    agent: AgentConfig
    llm: LLMConfig

class EnvManager:
    def load(self, profile: str | None = None) -> AppConfig: ...
```

## profile 差异


|            | dev             | prod                          |
| ---------- | --------------- | ----------------------------- |
| output_dir | `./var/reports` | `/var/lib/geegoo-agent/reports` |
| log json   | false           | true                          |
| dry_run 默认 | 可 true          | false                         |


## 与 geegoo skill Env 差异

不管理多 Python 版本 / Node / Docker 工作区；单一 `pyproject.toml` + venv。

## 代码

`src/geegoo/infra/env.py` + 根 `config.py` 门面。