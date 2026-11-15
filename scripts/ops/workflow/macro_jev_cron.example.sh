#!/usr/bin/env bash
# 七池 JEV 编排示例 cron（Asia/Shanghai 主机时区）。
# 复制到 Agent 主机后按需启用；正式线可先保留工作日 07:15 的 --workflow all，再逐步拆池。

set -euo pipefail
AGENT_ROOT="${AGENT_ROOT:-/home/ubuntu/.geegoo/geegoo-agent}"
RUN="$AGENT_ROOT/scripts/ops/workflow/run_macro_monitor_lab_batch.py"

run_one() {
  local wf="$1"
  cd "$AGENT_ROOT"
  set -a && . /home/ubuntu/.geegoo/agent.env && set +a
  export GEEGOO_AGENT_OP_HTTP_URL="${GEEGOO_AGENT_OP_HTTP_URL:-http://127.0.0.1:3400}"
  export MACRO_API_BASE="${MACRO_API_BASE:-http://146.56.225.252:3220}"
  /usr/bin/python3 "$RUN" --workflow "$wf" >>"/tmp/macro_jev_${wf}.log" 2>&1
}

# --- 示例时刻（产品确认后写入 crontab）---
# 07:00  cn_a + hk   run_one cn_a; run_one hk
# 05:30  us + commodity_fx  （美东收盘后，上海清晨）
# 08:55  bond（池内 .SH 段；.US 段另排 05:35 示例）
# 05:35  bond US 段 — 需脚本扩展分标的 cron 后启用
# 16:15  ET 锚点 crypto +24h — 用纽约 TZ cron 或单次 timer
# 07:10  volatility
# 七池池级均成功后：run_one overall

# 过渡期：与现网一致的一键全跑（含七池 + overall）
# 15 7 * * 1-5 ... python3 run_macro_monitor_lab_batch.py --workflow all