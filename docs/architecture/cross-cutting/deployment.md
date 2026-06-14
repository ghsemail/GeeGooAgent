# 部署

## 拓扑

```text
Linux Server
├── /opt/geegoo-agent/              # 代码 + venv
├── /etc/geegoo-agent/config.json   # secrets (600)
├── /var/lib/geegoo-agent/reports/  # StateStore 根
└── systemd
    ├── geegoo-agent.service        # oneshot
    └── geegoo-agent-pre.timer      # Mon-Fri 08:00 Asia/Shanghai
```

## systemd 示例

```ini
# geegoo-agent-pre.timer
[Timer]
OnCalendar=Mon..Fri *-*-* 08:00:00
Persistent=true

# geegoo-agent.service
[Service]
Type=oneshot
ExecStart=/opt/geegoo-agent/venv/bin/geegoo-agent run pre_market
EnvironmentFile=/etc/geegoo-agent/env
```

## 迁移自 Hermes

1. 部署 GeeGoo Agent + dry-run
2. 并行跑一天对比产出
3. disable Hermes 盘前 cron

## MVP

pre.timer only；post.timer Phase 2。