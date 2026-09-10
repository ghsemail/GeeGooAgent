---
name: remote-deploy
description: >-
  GeeGoo 远程部署（Git 同步 + restart）。Cloud Agent 启动时由 environment.json 从 COS 安装完整 skill 到 ~/.cursor/skills/remote-deploy。
  SSH 密码从 Cursor 云环境变量 GEEGOO_SSH_PASSWORD* 或 ~/.cursor/credentials/remote-deploy.json 读取。
---

# remote-deploy（Cloud 占位）

完整 skill 在 Cloud Agent Build 阶段自动安装到 `~/.cursor/skills/remote-deploy/`。

若未自动安装，手动执行：

```bash
bash .cursor/install-remote-deploy.sh
```

凭证：在 Cursor Cloud 环境变量中设置 `GEEGOO_SSH_PASSWORD`（或 `GEEGOO_SSH_PASSWORD_118_195_135_97` 等按 host）。

详细文档见安装后的 `~/.cursor/skills/remote-deploy/SKILL.md`。
