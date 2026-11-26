#!/usr/bin/env bash
# Example seven-pool JEV cron (Asia/Shanghai host TZ, weekdays Mon–Fri).
# Copy to the Agent host and enable as needed; do not paste into prod crontab — use install_macro_jev_crontab.sh.

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

# --- Weekday schedule (Asia/Shanghai) ---
# 05:30  us
# 05:32  commodity_fx
# 05:35  bond_us (US Treasury segment; still rolls into single bond pool)
# 05:38  crypto
# 05:40  volatility
# 15:30  cn_a
# 16:00  bond_sh (SSE bond segment)
# 16:30  hk
# 17:00  overall (reads today's ledger synthesis by default; missing pools block without rerunning others)
#         To overall after backfilling missing pools: --workflow overall --rerun-missing-pools

# Transition: local run-all still works via --workflow all (not staged prod path)
# 15 7 * * 1-5 ... python3 run_macro_monitor_lab_batch.py --workflow all