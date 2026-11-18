#!/usr/bin/env bash
# 七池 JEV 编排示例 cron（Asia/Shanghai 主机时区，工作日 1-5）。
# 复制到 Agent 主机后按需启用；勿将本示例直接写入生产 crontab，请用 install_macro_jev_crontab.sh。

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

# --- 确认工作日时刻表（Asia/Shanghai）---
# 05:30  us
# 05:32  commodity_fx
# 05:35  bond_us（美债分段；仍汇入单一 bond 池）
# 05:38  crypto
# 05:40  volatility
# 15:30  cn_a
# 16:00  bond_sh（上交所债分段）
# 16:30  hk
# 17:00  overall（账本七池齐全后再合成；缺池则先按 cron 重跑该池）

# 过渡期：本地一键全跑仍可用 --workflow all（非分时段生产路径）
# 15 7 * * 1-5 ... python3 run_macro_monitor_lab_batch.py --workflow all