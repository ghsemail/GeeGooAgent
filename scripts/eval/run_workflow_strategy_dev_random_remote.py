#!/usr/bin/env python3
"""Upload eval scripts, upsert workflow eval cases, run random strategy_dev cognition."""
from __future__ import annotations

import json
import sys
from pathlib import Path

import paramiko

ROOT = Path(__file__).resolve().parent
DEPLOY = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")
REMOTE_DIR = "/tmp/geegoo_workflow_eval"
UPSERT_SQL = REMOTE_DIR + "/upsert_workflow_eval.sql"

SQL = r"""
DELETE FROM agent_eval_cases WHERE id = 'taskflow_multi_strategy_compare';
INSERT INTO agent_eval_cases (
    id, user_id, title, description, steps_json, supports_random_stock, options_json, sort_order, enabled
) VALUES (
    'workflow_strategy_dev_cognition', '', 'Workflow · 策略认知（Macd4H）',
    'strategy_dev Step1：读策略库 → LLM 合成 Agent 认知 → 写入并读回 WeKnora 知识库。',
    '["发送策略认知请求（Macd4H）","校验 workflow 完成策略认知报告","校验回复含策略库/知识库与读回验证"]',
    FALSE,
    '{"category":"workflow","task":"strategy_dev","scenario":"cognition","message":"策略认知 Macd4H","min_reply_chars":80,"pass_keywords":["策略认知","Macd4H","知识库","策略库"],"session_cleanup":"before_run","wait_timeout_sec":900}',
    10, TRUE
) ON CONFLICT (id) DO UPDATE SET
    title = EXCLUDED.title, description = EXCLUDED.description,
    steps_json = EXCLUDED.steps_json, options_json = EXCLUDED.options_json,
    sort_order = EXCLUDED.sort_order, enabled = EXCLUDED.enabled;
INSERT INTO agent_eval_cases (
    id, user_id, title, description, steps_json, supports_random_stock, options_json, sort_order, enabled
) VALUES (
    'workflow_multi_strategy_compare', '', 'Workflow · 多策略信号对比',
    '固定标的，Workflow 串行 probe 多策略并输出对比表（非 ReAct）。',
    '["发送多策略对比请求","校验 workflow 对比表与买/卖次","校验回复含多策略信号对比"]',
    FALSE,
    '{"category":"workflow","message":"帮我在腾讯上对比 Macd4H 和 共振的信号买卖点","min_reply_chars":80,"pass_keywords":["多策略","对比","买","卖"],"session_cleanup":"before_run"}',
    11, TRUE
) ON CONFLICT (id) DO UPDATE SET
    title = EXCLUDED.title, description = EXCLUDED.description,
    steps_json = EXCLUDED.steps_json, options_json = EXCLUDED.options_json,
    sort_order = EXCLUDED.sort_order, enabled = EXCLUDED.enabled;
"""


def main() -> int:
    cfg = json.loads(DEPLOY.read_text(encoding="utf-8"))
    ssh = cfg["targets"]["geegoo-agent"]["ssh"]
    client = paramiko.SSHClient()
    client.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    client.connect(
        ssh["host"],
        port=int(ssh.get("port", 22)),
        username=ssh["user"],
        password=ssh.get("password"),
        timeout=30,
    )
    client.exec_command(f"mkdir -p {REMOTE_DIR}")
    sftp = client.open_sftp()
    sftp.put(str(ROOT / "run_workflow_strategy_dev_random_live.py"), f"{REMOTE_DIR}/run_workflow_strategy_dev_random_live.py")
    with sftp.file(UPSERT_SQL, "w") as f:
        f.write(SQL)
    sftp.close()

    upsert_cmd = (
        f"set -a && source ~/.geegoo/agent.env && set +a && "
        f'psql "$GEEGOO_PG_DSN" -v ON_ERROR_STOP=1 -f {UPSERT_SQL}'
    )
    _, stdout, stderr = client.exec_command(upsert_cmd, timeout=120)
    upsert_out = (stdout.read() + stderr.read()).decode("utf-8", errors="replace")
    if upsert_out.strip():
        print("upsert:", upsert_out.strip())

    cmd = f"cd {REMOTE_DIR} && python3 run_workflow_strategy_dev_random_live.py"
    if len(sys.argv) > 1:
        cmd += " " + " ".join(json.dumps(a) for a in sys.argv[1:])
    _, stdout, stderr = client.exec_command(cmd, timeout=960)
    out = stdout.read().decode("utf-8", errors="replace")
    err = stderr.read().decode("utf-8", errors="replace")
    print(out)
    if err.strip():
        print("STDERR:", err)
    client.close()
    return 0 if "SUMMARY: PASS" in out else 1


if __name__ == "__main__":
    raise SystemExit(main())
