# agent-runtime Web 接入（trading_operation）

## 1. CORS（开发 / 直连）

在 GeeGooAgent 服务器设置：

```bash
export GEEGOO_CORS_ORIGINS="http://localhost:8080,http://127.0.0.1:8080,https://你的运营域名"
systemctl restart geegoo-agent-runtime   # 或你的启动方式
```

Flutter Web 本地默认端口可能是随机；开发期可临时：

```bash
export GEEGOO_CORS_ORIGINS="*"
```

生产不要用 `*`，改为精确 Origin + Nginx 反代。

## 2. Nginx 反代（推荐生产）

```nginx
# 运营站点
server {
    listen 443 ssl;
    server_name ops.example.com;

    root /var/www/trading_operation/build/web;
    index index.html;

    location / {
        try_files $uri $uri/ /index.html;
    }

    # Agent Runtime — 浏览器同源访问，无需 CORS
    location /agent-runtime/ {
        proxy_pass http://127.0.0.1:3400/;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_read_timeout 320s;
        proxy_buffering off;   # SSE
    }
}
```

`server_url.dart` 改为同源：

```dart
const String agent_runtime_ip = '';  // 空 = 相对路径
// agent_runtime_server.dart 使用 '/agent-runtime' 作为 base
```

## 3. PostgreSQL（Session SSOT）

```bash
sudo apt install postgresql postgresql-contrib
sudo -u postgres createdb geegoo_agent
export GEEGOO_PG_DSN="postgres://user:pass@127.0.0.1:5432/geegoo_agent?sslmode=disable"
export GEEGOO_SESSION_STORE=postgres   # 可选；有 DSN 时默认 postgres
```

Schema 在 agent-runtime 启动时自动应用（`internal/infra/pgschema/`）。

从 SQLite 迁移会话：

```bash
geegoo migrate --from sqlite --to postgres
# 或从旧文件存储
geegoo migrate --from file --to postgres
```

## 4. pgvector 语义记忆

```bash
# 安装 pgvector 后
export GEEGOO_VECTOR_ENABLE=1
systemctl restart geegoo-agent-runtime
```

Cockpit **Memory** Tab → `GET /v1/memory/chunks`。

启用向量检索需配置 OpenAI Embedding：

```bash
export OPENAI_API_KEY=sk-...
export GEEGOO_EMBEDDING_MODEL=text-embedding-3-small   # 可选，默认此模型
```

会话结束后 summary 会自动写入 `agent_memory_chunks`（含 embedding）。

## 5. GeeGooBot BFF（推荐生产）

运营 Web 经 **agent-api :3110** 访问，不直连 agent-runtime：

```bash
# GeeGooBot 服务器 .env
GEEGOO_AGENT_RUNTIME_URL=http://127.0.0.1:3400
GEEGOO_AGENT_RUNTIME_API_KEY=<与 runtime 一致>
GEEGOO_BOT_AGENT_API_KEY=<运营 Web 携带的 Bearer>
```

`trading_operation/lib/api/server_url.dart`：

```dart
const bool agent_use_bff = true;
const String agent_bff_port = '3110';
const String agent_api_key = '<GEEGOO_BOT_AGENT_API_KEY>';
```

BFF 路由：`/op_agent/chat/stream` → runtime `/v1/chat/stream`，Cockpit 同理 `/op_agent/metrics/overview` 等。

生产 Nginx 完整示例见 [nginx-trading-operation.conf](./nginx-trading-operation.conf)。

`server_url.dart` 同源模式：

```dart
const bool agent_bff_use_proxy = true;
const String agent_bff_proxy_path = '/op_agent';
```

### MCP Token（BFF 校验）

Chat 写操作（`/v1/chat/*`）可要求运营用户 MCP Token：

```bash
GEEGOO_BOT_MONGO_URI=mongodb://...        # validate 时必填
GEEGOO_AGENT_REQUIRE_MCP_TOKEN=true       # 缺少 X-MCP-Token → 401
GEEGOO_AGENT_VALIDATE_MCP_TOKEN=true      # 校验 QT_DB.user.mcp.mcp_token，并注入 X-User-Id
```

Cockpit 只读 API 不受 MCP 限制（仍受 Bearer `GEEGOO_BOT_AGENT_API_KEY` 保护）。

Nginx 同源反代示例：

```nginx
location /op_agent/ {
    proxy_pass http://127.0.0.1:3110/op_agent/;
    proxy_http_version 1.1;
    proxy_read_timeout 320s;
    proxy_buffering off;
}
```

## 6. 向量库（外部 Qdrant，可选）

```bash
# 示例：Qdrant
docker run -p 6333:6333 qdrant/qdrant
export GEEGOO_VECTOR_URL="http://127.0.0.1:6333"
```

语义检索接入 `memport` 后 Memory Tab 会展示向量后端。
