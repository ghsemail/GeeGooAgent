#!/usr/bin/env bash
# Install seven-pool staggered macro Lab crons on GeeGooAgent host.
set -euo pipefail

AGENT_ROOT="${AGENT_ROOT:-/home/ubuntu/.geegoo/geegoo-agent}"
RUN="$AGENT_ROOT/scripts/ops/workflow/run_macro_monitor_lab_batch.py"
PREFIX="geegoo-macro-jev"

TMP="$(mktemp)"
{
  crontab -l 2>/dev/null | grep -v "$PREFIX" | grep -v 'geegoo-macro-monitor-lab-batch' || true
  cat <<EOF
# $PREFIX — 七池分时段（Asia/Shanghai）
35 5 * * 1-5 cd $AGENT_ROOT && set -a && . /home/ubuntu/.geegoo/agent.env && set +a && export GEEGOO_AGENT_OP_HTTP_URL=\${GEEGOO_AGENT_OP_HTTP_URL:-http://127.0.0.1:3400} MACRO_API_BASE=\${MACRO_API_BASE:-http://146.56.225.252:3220} && /usr/bin/python3 $RUN --workflow bond_us --date auto >>/tmp/macro_jev_bond_us.log 2>&1 # $PREFIX-bond_us
30 5 * * 1-5 cd $AGENT_ROOT && set -a && . /home/ubuntu/.geegoo/agent.env && set +a && export GEEGOO_AGENT_OP_HTTP_URL=\${GEEGOO_AGENT_OP_HTTP_URL:-http://127.0.0.1:3400} MACRO_API_BASE=\${MACRO_API_BASE:-http://146.56.225.252:3220} && /usr/bin/python3 $RUN --workflow us --date auto >>/tmp/macro_jev_us.log 2>&1 # $PREFIX-us
32 5 * * 1-5 cd $AGENT_ROOT && set -a && . /home/ubuntu/.geegoo/agent.env && set +a && export GEEGOO_AGENT_OP_HTTP_URL=\${GEEGOO_AGENT_OP_HTTP_URL:-http://127.0.0.1:3400} MACRO_API_BASE=\${MACRO_API_BASE:-http://146.56.225.252:3220} && /usr/bin/python3 $RUN --workflow commodity_fx --date auto >>/tmp/macro_jev_commodity_fx.log 2>&1 # $PREFIX-commodity_fx
0 7 * * 1-5 cd $AGENT_ROOT && set -a && . /home/ubuntu/.geegoo/agent.env && set +a && export GEEGOO_AGENT_OP_HTTP_URL=\${GEEGOO_AGENT_OP_HTTP_URL:-http://127.0.0.1:3400} MACRO_API_BASE=\${MACRO_API_BASE:-http://146.56.225.252:3220} && /usr/bin/python3 $RUN --workflow cn_a --date auto >>/tmp/macro_jev_cn_a.log 2>&1 # $PREFIX-cn_a
2 7 * * 1-5 cd $AGENT_ROOT && set -a && . /home/ubuntu/.geegoo/agent.env && set +a && export GEEGOO_AGENT_OP_HTTP_URL=\${GEEGOO_AGENT_OP_HTTP_URL:-http://127.0.0.1:3400} MACRO_API_BASE=\${MACRO_API_BASE:-http://146.56.225.252:3220} && /usr/bin/python3 $RUN --workflow hk --date auto >>/tmp/macro_jev_hk.log 2>&1 # $PREFIX-hk
10 7 * * 1-5 cd $AGENT_ROOT && set -a && . /home/ubuntu/.geegoo/agent.env && set +a && export GEEGOO_AGENT_OP_HTTP_URL=\${GEEGOO_AGENT_OP_HTTP_URL:-http://127.0.0.1:3400} MACRO_API_BASE=\${MACRO_API_BASE:-http://146.56.225.252:3220} && /usr/bin/python3 $RUN --workflow volatility --date auto >>/tmp/macro_jev_volatility.log 2>&1 # $PREFIX-volatility
55 8 * * 1-5 cd $AGENT_ROOT && set -a && . /home/ubuntu/.geegoo/agent.env && set +a && export GEEGOO_AGENT_OP_HTTP_URL=\${GEEGOO_AGENT_OP_HTTP_URL:-http://127.0.0.1:3400} MACRO_API_BASE=\${MACRO_API_BASE:-http://146.56.225.252:3220} && /usr/bin/python3 $RUN --workflow bond_sh --date auto >>/tmp/macro_jev_bond_sh.log 2>&1 # $PREFIX-bond_sh
10 9 * * 1-5 cd $AGENT_ROOT && set -a && . /home/ubuntu/.geegoo/agent.env && set +a && export GEEGOO_AGENT_OP_HTTP_URL=\${GEEGOO_AGENT_OP_HTTP_URL:-http://127.0.0.1:3400} MACRO_API_BASE=\${MACRO_API_BASE:-http://146.56.225.252:3220} && /usr/bin/python3 $RUN --workflow overall --date auto >>/tmp/macro_jev_overall.log 2>&1 # $PREFIX-overall
CRON_TZ=America/New_York
15 16 * * * cd $AGENT_ROOT && set -a && . /home/ubuntu/.geegoo/agent.env && set +a && export GEEGOO_AGENT_OP_HTTP_URL=\${GEEGOO_AGENT_OP_HTTP_URL:-http://127.0.0.1:3400} MACRO_API_BASE=\${MACRO_API_BASE:-http://146.56.225.252:3220} && /usr/bin/python3 $RUN --workflow crypto --date auto >>/tmp/macro_jev_crypto.log 2>&1 # $PREFIX-crypto
CRON_TZ=Asia/Shanghai
EOF
} >"$TMP"

crontab "$TMP"
rm -f "$TMP"
echo "Installed macro JEV staggered crontab ($PREFIX)."
crontab -l | grep "$PREFIX" || true