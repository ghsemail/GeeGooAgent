# WeKnora 知识库旁路部署（第一期）

**日期:** 2026-08-22  
**状态:** 待实现  
**范围仓库:** `GeeGooAgent`（服务器编排与种子脚本）、`GeeGooSignal`（运营台模型路由）、`trading_operation`（Embedding 配置 UI）

## 背景

要在 GeeGooAgent 服务器上安装腾讯开源知识库 [WeKnora](https://github.com/Tencent/WeKnora)，先把服务跑通并配好模型，文档管理走 WeKnora 自带 Web。Agent 对话检索、运营台「知识库」模块留到下一期。

现网已确认：

- 主机 `82.157.97.76`（与 GeeGooData CN、GeeGooAgent Runtime 同机）
- Agent 已有 PostgreSQL 16 + pgvector，仅监听 `127.0.0.1:5432`，库名 `geegoo_agent`（会话 / 语义记忆）
- 运营台「模型配置」已有对话模型一主一备（GeeGooSignal `ai_model_runtime_config`）
- Agent Embedding 默认 `kinfra-text-embedding-4b`（tencent-maas，2560 维）

## 已确认决策

| 项 | 选择 |
|----|------|
| 部署形态 | 官方 Docker Compose 标准栈，旁路独立进程 |
| 数据库 | WeKnora 自带 ParadeDB 容器；**不**复用 `geegoo_agent` |
| 管理界面 | WeKnora 自带 Web；本期不在运营台做知识库 Tab |
| 对话模型 | 运营台主模型写入默认知识库；备模型只注册到 WeKnora，供手动切换 |
| Embedding | 复用现网 kinfra；运营台「模型配置」增加 Embedding 下拉 |
| 模型热更新 | 本期不把运营台改路由自动同步到 WeKnora |
| Agent 检索 | 不做 |
| 对外端口 | 只公开 Web `:3480`；后端 API 仅本机 `127.0.0.1:3481` 供种子脚本 |

## 非目标（第一期不做）

- GeeGooAgent Tool / Memory 调用 WeKnora 检索
- trading_operation 新建「知识库」模块（建库、上传、删除）
- WeKnora 内对话模型自动主备故障转移
- 与 Agent Postgres 共用实例或共用库
- 为 WeKnora 配独立公网域名 / TLS（可用 IP:3480；Nginx 有需要再加）

## 架构

```text
运营员浏览器
    │
    ├─ trading_operation :8088
    │     模型配置：主 / 备 / Embedding
    │     └── GeeGooSignal catalog-api :3210
    │           ai_model + ai_model_runtime_config
    │
    └─ http://82.157.97.76:3480   WeKnora Web
          └── WeKnora-frontend (container :80)
                └── WeKnora-app :8080
                      ├── ParadeDB (docker 内网，不映射宿主机 5432)
                      ├── Redis (docker 内网)
                      └── docreader (docker 内网)

宿主机已有（互不占用端口）：
  GeeGooAgent runtime :3400
  GeeGooData CN       :3300
  Agent Postgres      127.0.0.1:5432
  Mongo               127.0.0.1:27017
```

## 组件

### 1. 服务器上的 WeKnora 栈

路径：`/home/ubuntu/apps/WeKnora`（官方仓库 clone，不进 GeeGoo git）。

GeeGoo 覆盖文件进 `GeeGooAgent/deploy/weknora/`，服务器对齐后拷到该目录：

| 文件 | 作用 |
|------|------|
| `docker-compose.override.yml` | `FRONTEND_PORT`→3480；app 端口改为 `127.0.0.1:3481:8080`；不映射 ParadeDB/Redis |
| `.env.example` | 端口、密码占位、`DISABLE_REGISTRATION`、`OLLAMA_OPTIONAL=true` |
| `scripts/bootstrap_weknora.sh` | 等 healthy → 注册管理员 → 关公开注册 → 写模型 → 建默认空库 |
| `scripts/install_weknora.sh` | 机器上 clone 官方仓、放覆盖文件、`docker compose pull && up -d` |

只启官方默认 profile（frontend / app / postgres / redis / docreader）。不加 `full`、`neo4j`、`minio`、`langfuse`。

`WEKNORA_VERSION` 在服务器 `.env` 钉死 `v0.7.2`（Docker Hub 标签带 `v` 前缀）。

官方 `postgres` 服务默认**不**映射宿主机 5432，与 Agent Postgres 不冲突。覆盖文件仍禁止给 ParadeDB 加 host port。

### 2. 管理员与默认库

- 用户名 `geegoo-admin`，邮箱 `admin@geegoo.local`
- 密码部署时随机生成，写入 `/home/ubuntu/apps/WeKnora/.geegoo-admin`（权限 `600`），**不入库**
- 第一个账号注册成功后设 `DISABLE_REGISTRATION=true` 并重启 app（或等价系统设置 `auth.registration_mode=invite_only`）
- 将该邮箱提升为 System Admin（`WEKNORA_BOOTSTRAP_SYSTEM_ADMIN_EMAIL`）
- 预建空知识库，名称 `GeeGoo`，类型 `document`；不预置文档

### 3. 模型写入 WeKnora

种子脚本跑在 Agent 主机上，凭证只读本机已有配置、不新造一套：

- catalog：`~/.geegoo/config.json` 的 `signal_catalog` URL + key（与 Agent 现网一致，`http://146.56.225.252:3210`）
- Embedding 回退：Agent `ResolvedEmbedding()`（`config.json` 的 `embedding` 或 env `OPENAI_API_KEY` / `GEEGOO_EMBEDDING_MODEL`）
- 若 catalog 已有 `embedding_model_id`，优先用目录里那条模型的 base_url / token / name

| WeKnora 用途 | 来源 | 行为 |
|--------------|------|------|
| 默认库 LLM | `configured_model`（主） | `source` 按 OpenAI 兼容远程写入；`baseUrl`/`apiKey`/`modelName` 来自 catalog 模型 |
| 租户第二 LLM | `fallback_model_ids[0]`（备） | 只注册，不绑默认库 |
| 默认库 Embedding | catalog `embedding_model_id`，否则 Agent 默认 kinfra | `dimension=2560`；建库后不再更换 |

连通失败则种子脚本退出非 0，不创建知识库。Rerank / 多模态 / 图谱第一期关闭。

WeKnora 每个知识库只绑定一个对话模型，因此**没有**运营台那种自动切备。

### 4. 运营台 Embedding 配置

仓库：`GeeGooSignal` + `trading_operation`。

`ai_model_runtime_config` 增加：

```json
{
  "fallback_model_ids": ["..."],
  "embedding_model_id": "<catalog model_id or empty>"
}
```

- 空 = Agent / WeKnora 继续用现网 kinfra 默认
- 「模型配置」路由卡片在主/备旁增加「Embedding 模型」下拉（可选「使用 Agent 默认」）
- 保存走现有 runtime config API，一次提交
- 若目录没有 kinfra 条目：种子脚本或运营台首次保存时，按 Agent 默认值补一条（display_name `kinfra-text-embedding-4b`，base `https://tokenhub.tencentmaas.com/v1`）
- GeeGooAgent `internal/clients/admin` 的 view 同步带上 `embedding_model_id`，供后续 Agent 使用；本期不改 Agent 检索路径

### 5. 密钥

| 秘密 | 位置 |
|------|------|
| WeKnora `DB_PASSWORD` / `REDIS_PASSWORD` / `JWT_SECRET` / `SYSTEM_AES_KEY` | 服务器 `.env`，gitignore |
| 管理员密码 | `.geegoo-admin`，权限 600 |
| 模型 API Key | 从 catalog / Agent env 读入 WeKnora，不写进 GeeGoo git |

`deploy/weknora/.env.example` 只含占位符。

## 数据流

1. 部署 `install_weknora.sh`：clone → overlay → `docker compose up -d`
2. `bootstrap_weknora.sh`：`curl 127.0.0.1:3481/health` → 注册 → 登录 JWT → 拉 catalog 主备与 Embedding → `POST /initialization/remote/check` 与 embedding test → 建库 `GeeGoo` → `POST /initialization/initialize/:kbId`
3. 运营员打开 `http://82.157.97.76:3480`，用 `.geegoo-admin` 登录，在 Web 里上传/删除文档、再建库
4. 运营员在 trading_operation「模型配置」里改 Embedding，只更新 catalog；WeKnora 已建库的 Embedding **不自动变**（换 Embedding 需重建索引，留给下一期或手工在 WeKnora 新建库）

## 失败处理

| 现象 | 处理 |
|------|------|
| 无 Docker / 内存明显不足（建议 ≥8GB 空闲） | 安装脚本拒绝启动并打印内存与 `docker` 状态 |
| 3480 或 3481 已被占用 | 拒绝启动，列出占用进程 |
| ParadeDB 起不来 | `docker compose logs postgres`，不继续 bootstrap |
| 主模型或 Embedding 连通失败 | bootstrap 失败退出；容器可保留，不建默认库 |
| 公开注册未关掉 | bootstrap 必须把注册模式打到 invite_only，否则视为失败 |
| Agent Postgres 与 WeKnora 冲突 | 覆盖文件禁止映射 5432；验收检查宿主机 5432 仍是 Agent |

不把部分成功当成完成：没有默认库、没有管理员、或模型测试失败，均不算部署成功。

## 验收

1. `docker compose ps`：`WeKnora-frontend` / `WeKnora-app` / `WeKnora-postgres` / `WeKnora-redis` / docreader 为 healthy 或 running
2. `curl -sf http://127.0.0.1:3481/health` 返回 ok
3. 浏览器打开 `http://82.157.97.76:3480` 出现登录页；`geegoo-admin` 可登录；注册入口关闭
4. 存在知识库 `GeeGoo`；初始化配置含主 LLM 与 kinfra Embedding
5. 宿主机 `ss -lntp`：`5432` 仍为系统 PostgreSQL；WeKnora ParadeDB 无 `0.0.0.0:5432`
6. catalog `getModelRuntimeConfig` 响应含 `embedding_model_id`
7. 运营台「模型配置」能看到并保存 Embedding 下拉

## 测试

- GeeGooSignal：`runtime_config` 读写 `embedding_model_id`（空 / 合法 id / 未知 id 拒绝）
- trading_operation：路由卡片增加 Embedding 下拉，保存 payload 含该字段
- 种子脚本：对 catalog 与 WeKnora API 用固定 fixture 做 dry-run 解析测试（不在 CI 里真拉镜像）

线上验证按上一节验收清单，在 `82.157.97.76` 执行。

## 部署纪律

- GeeGoo 侧改动：本机 commit + push，服务器 `git fetch && reset --hard` 后再跑安装脚本
- 禁止把 WeKnora 业务镜像或 `.env` 当「业务代码」用 SCP 代替 git
- 服务器 clone 官方 `Tencent/WeKnora` 视为第三方运行时，与安装 Postgres 同类
- trading_operation 静态资源仍走现有 `deploy_trading_operation_web.py`

## 下一期（本文不实施）

运营台知识库模块、GeeGooAgent 检索 Tool、运营台改模型后同步 WeKnora、域名与 TLS。
