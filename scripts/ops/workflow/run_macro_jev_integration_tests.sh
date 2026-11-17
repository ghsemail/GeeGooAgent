#!/usr/bin/env bash
# Integration + regression for macro JEV canonical PUT (Signal b3eb834 + Agent client).
set -euo pipefail
ROOT="$(cd "$(dirname "$0")" && pwd)"
cd "$ROOT"

UNIT_FAIL=0
INT_FAIL=0
SIGNAL_FAIL=0

echo "== Agent unit tests =="
python3 -m py_compile run_macro_monitor_lab_batch.py macro_jev_results_client.py macro_jev_orchestration.py
python3 -m unittest -v \
  macro_jev_results_client_test.py \
  macro_jev_orchestration_test.py \
  macro_jev_audit_schema_test.py \
  agent_chat_stream_test.py || UNIT_FAIL=1

if [[ "${SKIP_MACRO_JEV_INTEGRATION:-}" == "1" ]]; then
  echo "SKIP_MACRO_JEV_INTEGRATION=1 — skip Signal HTTP integration"
else
  echo "== Agent × Signal HTTP integration (requires Mongo 127.0.0.1:27017) =="
  python3 -m unittest -v macro_jev_signal_integration_test.py || INT_FAIL=1
fi

if [[ -d /agent/repos/GeeGooSignal ]]; then
  echo "== Signal macro JEV tests (Go, b3eb834 tree) =="
  (
    cd /agent/repos/GeeGooSignal
    export GEEGOO_SIGNAL_MONGO_URI="${GEEGOO_SIGNAL_MONGO_URI:-mongodb://127.0.0.1:27017}"
    go test -count=1 -run 'Jev' ./internal/macro/ -timeout 120s
  ) || SIGNAL_FAIL=1
else
  echo "Skip Signal go tests: /agent/repos/GeeGooSignal missing"
fi

TOTAL=$((UNIT_FAIL + INT_FAIL + SIGNAL_FAIL))
echo "summary unit_fail=$UNIT_FAIL integration_fail=$INT_FAIL signal_go_fail=$SIGNAL_FAIL"
exit "$TOTAL"