#!/usr/bin/env python3
"""Upsert strategy_signal_explain_followup into agent_eval_cases on geegoo-agent PG."""
from __future__ import annotations

import json
from pathlib import Path

import paramiko

DEPLOY = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")
REMOTE_SQL = "/tmp/upsert_strategy_explain_eval.sql"

SQL = """
INSERT INTO agent_eval_cases (
    id, user_id, title, description, steps_json, supports_random_stock, options_json, sort_order
) VALUES (
    'strategy_signal_explain_followup',
    '',
    '单股 · 零信号解释 · 2轮剧本',
    '首轮 probe 后追问为何零信号；第2轮应 diagnose 解释规则与指标，不再 probe。',
    '["随机选 1 股 + 1 策略，发送首轮信号测试","发送「为什么没信号」追问","校验 diagnose 与解释文案"]',
    TRUE,
    '{"category":"strategy_signal","task":"signal_probe","scenario":"single","stock_count":1,"strategy_count":1,"random_stock_enabled":true,"min_reply_chars":80,"pass_keywords":["信号","阈值","没","触发","RSI","参数"],"session_cleanup":"before_run","explain_followup":true,"require_tools":["diagnose_bot_signal_series"],"forbid_tools":["run_strategy_backtest"],"dialogue":[{"role":"user","text":"为什么一个信号都没有？帮我从数据和规则上解释一下","judge":true}]}',
    14
) ON CONFLICT (id) DO UPDATE SET
    title = EXCLUDED.title,
    description = EXCLUDED.description,
    steps_json = EXCLUDED.steps_json,
    supports_random_stock = EXCLUDED.supports_random_stock,
    options_json = EXCLUDED.options_json,
    sort_order = EXCLUDED.sort_order;
"""


def main() -> int:
    cfg = json.loads(DEPLOY.read_text(encoding="utf-8"))
    ssh = cfg["targets"]["geegoo-agent"]["ssh"]
    client = paramiko.SSHClient()
    client.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    client.connect(
        hostname=ssh["host"],
        username=ssh["user"],
        password=ssh.get("password"),
        timeout=30,
    )
    sftp = client.open_sftp()
    with sftp.file(REMOTE_SQL, "w") as f:
        f.write(SQL)
    sftp.close()
    cmd = (
        f"set -a && source ~/.geegoo/agent.env && set +a && "
        f'psql "$GEEGOO_PG_DSN" -v ON_ERROR_STOP=1 -f {REMOTE_SQL} && '
        f'psql "$GEEGOO_PG_DSN" -tAc "SELECT id FROM agent_eval_cases WHERE id=\'strategy_signal_explain_followup\'"'
    )
    _, stdout, stderr = client.exec_command(cmd, timeout=60)
    out = (stdout.read() + stderr.read()).decode("utf-8", errors="replace")
    print(out.strip())
    client.close()
    return 0 if "strategy_signal_explain_followup" in out else 1


if __name__ == "__main__":
    raise SystemExit(main())
