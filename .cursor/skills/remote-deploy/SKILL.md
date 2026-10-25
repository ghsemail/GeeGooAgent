---
name: remote-deploy
description: >-
  GeeGoo 远程部署：用于把 GeeGooBot / GeeGooSignal / GeeGooData / GeeGooAgent / Trading*
  后端发布到服务器（先 commit + push main，再让服务器 git 对齐并重启），或上传 trading_operation Web
  静态资源、检查线上是否和 origin/main 对齐。
---
# GeeGoo 远程部署（Git 同步）

完整说明（主机表、各项目服务与端口、重启菜单）在同目录 `REFERENCE.md`，脚本在 `scripts/`，配置在 `deploy.json`（**不含密码**）。在我的电脑上路径是 `/home/box/agent-data/workflows/remote-deploy/`。

## 铁律

1. 先在对应仓库 commit 并 `git push origin main`，没 push 不部署。
2. 服务器只用 `git fetch origin main && git reset --hard origin/main` 对齐，不用 SSH 直传代码。`--files` 直传必须用户明确授权且加 `--emergency`，事后补 commit/push。
3. GitHub main、本地源码、线上运行代码三方一致。
4. 部署和重启会影响线上服务，执行前先向用户确认 target 和服务。
5. **密码永不写入** skill / `deploy.json` / COS zip；上传 COS 前必须剥离密码。

## 密码从哪读

按运行环境区分，脚本 `credentials.py` 解析顺序与此一致（环境变量优先，再凭证库）：

| 环境 | 怎么放密码 |
|---|---|
| **本机助手**（Grok Bot / 本机 Cursor） | 读**环境变量** `GEEGOO_SSH_PASSWORD_<IP 把点换成下划线>`，或 `GEEGOO_SSH_PASSWORD_<TARGET 大写下划线>`。例：`146.56.225.252` → `GEEGOO_SSH_PASSWORD_146_56_225_252`。部署前由用户写入机器变量。 |
| **Cursor 云 Agent** | 读**凭证库** `~/.cursor/credentials/remote-deploy.json`（模板 `credentials.example.json`，键为 target 名、主机 IP 或 `default`）。也可在 Cloud Secrets 注入同名环境变量作为补充。 |

都没有时用加密输入框（secret-request）向用户要，**不要**让人把密码贴在聊天里。COS 密钥用 `TENCENT_COS_SECRET_ID` / `TENCENT_COS_SECRET_KEY`（或凭证库 `cos` 段）。

## 常用命令

脚本依赖 `paramiko`（首次 `pip install paramiko`）。以下 `$S=/home/box/agent-data/workflows/remote-deploy/scripts`。

```bash
python $S/deploy.py --list                         # 列出 target 与服务
python $S/deploy.py --check-git                    # 检查各后端是否对齐 origin/main
python $S/deploy.py -t geegoo-bot --git-deploy     # 单项目 git 同步 + 重启
python $S/deploy.py -t geegoo-bot --git-deploy -s mcp-api
python $S/deploy.py -t geegoo-data --git-sync      # 只同步不重启
python $S/deploy.py -t geegoo-signal -s signal-api # 代码已对齐，只重启
python $S/upload_skill_to_cos.py                   # 打包 skill 上传 COS（deploy.json 无密码）
```

## target 速查

| target | 主机 | 方式 |
|---|---|---|
| geegoo-bot | 118.195.135.97 | git，`echo N \| bash start.sh` 菜单 |
| geegoo-signal | 146.56.225.252 | git，`./start.sh restart` |
| geegoo-data | 47.80.14.120（港美） | git，`./start.sh restart` |
| geegoo-data-cn | 82.157.97.76（A 股） | git，`./start.sh restart` |
| geegoo-agent | 82.157.97.76 | install.sh，验证 `~/.geegoo/bin/geegoo doctor` |
| trading-operation | 146.56.225.252:8088 | `flutter build web` 后跑 `deploy_trading_operation_web.py` |
| geegoo-trading* | 见 REFERENCE.md | Python 老栈 |

## 验证

部署后 `curl` 各服务 `/health`，并 `tail` 对应 `.out` 日志；再跑一次 `--check-git` 确认对齐。
