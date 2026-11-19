#!/usr/bin/env bash
# Install seven-pool staggered macro Lab crons on GeeGooAgent host.
# Times are Asia/Shanghai, weekdays (Mon–Fri) only — see docs/MACRO_MONITOR_BATCH.md.
set -euo pipefail

AGENT_ROOT="${AGENT_ROOT:-/home/ubuntu/.geegoo/geegoo-agent}"
RUN="$AGENT_ROOT/scripts/ops/workflow/run_macro_monitor_lab_batch.py"
PREFIX="geegoo-macro-jev"

TMP="$(mktemp)"
{
  crontab -l 2>/dev/null | grep -v "$PREFIX" | grep -v 'geegoo-macro-monitor-lab-batch' || true
  cat <<EOF
# $PREFIX — 七池分时段（Asia/Shanghai，工作日 1-5）
30 5 * * 1-5 cd $AGENT_ROOT && set -a && . /home/ubuntu/.geegoo/agent.env && set +a && export GEEGOO_AGENT_OP_HTTP_URL=\${GEEGOO_AGENT_OP_HTTP_URL:-http://127.0.0.1:3400} MACRO_API_BASE=\${MACRO_API_BASE:-http://146.56.225.252:3220} MACRO_BATCH_CHAT_TIMEOUT=\${MACRO_BATCH_CHAT_TIMEOUT:-900} && /usr/bin/python3 $RUN --workflow us --date auto >>/tmp/macro_jev_us.log 2>&1 # $PREFIX-us
32 5 * * 1-5 cd $AGENT_ROOT && set -a && . /home/ubuntu/.geegoo/agent.env && set +a && export GEEGOO_AGENT_OP_HTTP_URL=\${GEEGOO_AGENT_OP_HTTP_URL:-http://127.0.0.1:3400} MACRO_API_BASE=\${MACRO_API_BASE:-http://146.56.225.252:3220} MACRO_BATCH_CHAT_TIMEOUT=\${MACRO_BATCH_CHAT_TIMEOUT:-900} && /usr/bin/python3 $RUN --workflow commodity_fx --date auto >>/tmp/macro_jev_commodity_fx.log 2>&1 # $PREFIX-commodity_fx
35 5 * * 1-5 cd $AGENT_ROOT && set -a && . /home/ubuntu/.geegoo/agent.env && set +a && export GEEGOO_AGENT_OP_HTTP_URL=\${GEEGOO_AGENT_OP_HTTP_URL:-http://127.0.0.1:3400} MACRO_API_BASE=\${MACRO_API_BASE:-http://146.56.225.252:3220} MACRO_BATCH_CHAT_TIMEOUT=\${MACRO_BATCH_CHAT_TIMEOUT:-900} && /usr/bin/python3 $RUN --workflow bond_us --date auto >>/tmp/macro_jev_bond_us.log 2>&1 # $PREFIX-bond_us
38 5 * * 1-5 cd $AGENT_ROOT && set -a && . /home/ubuntu/.geegoo/agent.env && set +a && export GEEGOO_AGENT_OP_HTTP_URL=\${GEEGOO_AGENT_OP_HTTP_URL:-http://127.0.0.1:3400} MACRO_API_BASE=\${MACRO_API_BASE:-http://146.56.225.252:3220} MACRO_BATCH_CHAT_TIMEOUT=\${MACRO_BATCH_CHAT_TIMEOUT:-900} && /usr/bin/python3 $RUN --workflow crypto --date auto >>/tmp/macro_jev_crypto.log 2>&1 # $PREFIX-crypto
40 5 * * 1-5 cd $AGENT_ROOT && set -a && . /home/ubuntu/.geegoo/agent.env && set +a && export GEEGOO_AGENT_OP_HTTP_URL=\${GEEGOO_AGENT_OP_HTTP_URL:-http://127.0.0.1:3400} MACRO_API_BASE=\${MACRO_API_BASE:-http://146.56.225.252:3220} MACRO_BATCH_CHAT_TIMEOUT=\${MACRO_BATCH_CHAT_TIMEOUT:-900} && /usr/bin/python3 $RUN --workflow volatility --date auto >>/tmp/macro_jev_volatility.log 2>&1 # $PREFIX-volatility
30 15 * * 1-5 cd $AGENT_ROOT && set -a && . /home/ubuntu/.geegoo/agent.env && set +a && export GEEGOO_AGENT_OP_HTTP_URL=\${GEEGOO_AGENT_OP_HTTP_URL:-http://127.0.0.1:3400} MACRO_API_BASE=\${MACRO_API_BASE:-http://146.56.225.252:3220} MACRO_BATCH_CHAT_TIMEOUT=\${MACRO_BATCH_CHAT_TIMEOUT:-900} && /usr/bin/python3 $RUN --workflow cn_a --date auto >>/tmp/macro_jev_cn_a.log 2>&1 # $PREFIX-cn_a
0 16 * * 1-5 cd $AGENT_ROOT && set -a && . /home/ubuntu/.geegoo/agent.env && set +a && export GEEGOO_AGENT_OP_HTTP_URL=\${GEEGOO_AGENT_OP_HTTP_URL:-http://127.0.0.1:3400} MACRO_API_BASE=\${MACRO_API_BASE:-http://146.56.225.252:3220} MACRO_BATCH_CHAT_TIMEOUT=\${MACRO_BATCH_CHAT_TIMEOUT:-900} && /usr/bin/python3 $RUN --workflow bond_sh --date auto >>/tmp/macro_jev_bond_sh.log 2>&1 # $PREFIX-bond_sh
30 16 * * 1-5 cd $AGENT_ROOT && set -a && . /home/ubuntu/.geegoo/agent.env && set +a && export GEEGOO_AGENT_OP_HTTP_URL=\${GEEGOO_AGENT_OP_HTTP_URL:-http://127.0.0.1:3400} MACRO_API_BASE=\${MACRO_API_BASE:-http://146.56.225.252:3220} MACRO_BATCH_CHAT_TIMEOUT=\${MACRO_BATCH_CHAT_TIMEOUT:-900} && /usr/bin/python3 $RUN --workflow hk --date auto >>/tmp/macro_jev_hk.log 2>&1 # $PREFIX-hk
0 17 * * 1-5 cd $AGENT_ROOT && set -a && . /home/ubuntu/.geegoo/agent.env && set +a && export GEEGOO_AGENT_OP_HTTP_URL=\${GEEGOO_AGENT_OP_HTTP_URL:-http://127.0.0.1:3400} MACRO_API_BASE=\${MACRO_API_BASE:-http://146.56.225.252:3220} MACRO_BATCH_CHAT_TIMEOUT=\${MACRO_BATCH_CHAT_TIMEOUT:-900} && /usr/bin/python3 $RUN --workflow overall --date auto >>/tmp/macro_jev_overall.log 2>&1 # $PREFIX-overall
EOF
} >"$TMP"

crontab "$TMP"
rm -f "$TMP"
echo "Installed macro JEV staggered crontab ($PREFIX)."
crontab -l | grep "$PREFIX" || true