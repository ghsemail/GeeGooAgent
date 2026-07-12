# L0 — Secrets Manager

## 职责

API Key、mcp_token 管理——禁止进 git。

## 接口

```python
class SecretsProvider(Protocol):
    def get(self, key: str) -> str: ...
    def get_optional(self, key: str) -> str | None: ...

class EnvSecrets(SecretsProvider):
    """OPENAI_API_KEY, MCP_TOKEN 等环境变量"""

class FileSecrets(SecretsProvider):
    """/etc/geegoo-agent/config.json，权限 600"""
```

## 解析顺序

```text
1. 环境变量（覆盖）
2. config.json
3. 缺失 → 启动失败（明确报错）
```

## 密钥清单

| Key                           | 用途          |
| ----------------------------- | ----------- |
| `api_key` / `SK_API_KEY`      | 5900        |
| `geegoo_api_key` / `SK_API_KEY` | 3120        |
| `mcp_token`                   | GeeGoo 用户     |
| `OPENAI_API_KEY`              | L1          |
| `ANTHROPIC_API_KEY`           | L1 fallback |

## 后期

HashiCorp Vault adapter。

## 代码

`src/geegoo/infra/secrets.py`