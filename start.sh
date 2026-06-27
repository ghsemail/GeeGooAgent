#!/usr/bin/env bash
# GeeGooAgent process manager (Go main branch).
set -euo pipefail

APP_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$APP_DIR"

GEEGOO_HOME="${GEEGOO_HOME:-$HOME/.geegoo}"
BIN_DIR="${GEEGOO_BIN_DIR:-$GEEGOO_HOME/bin}"
CONFIG_PATH="${GEEGOO_CONFIG:-$GEEGOO_HOME/config.json}"
mkdir -p "$BIN_DIR"

export PATH="/usr/local/go/bin:${BIN_DIR}:${PATH:-}"
export GOPROXY="${GOPROXY:-https://goproxy.cn,direct}"

PORT_RUNTIME="${GEEGOO_AGENT_RUNTIME_PORT:-3400}"
LOG_RUNTIME="${APP_DIR}/agent-runtime.out"
PID_RUNTIME="${APP_DIR}/agent-runtime.pid"

log() { echo "[GeeGooAgent] $*"; }

build() {
  log "building geegoo + agentRuntimeServer..."
  go build -o "$BIN_DIR/geegoo" ./cmd/geegoo
  go build -o "$BIN_DIR/agentRuntimeServer" ./cmd/agent-runtime
}

start_runtime() {
  if [[ -f "$PID_RUNTIME" ]] && kill -0 "$(cat "$PID_RUNTIME")" 2>/dev/null; then
    log "agent-runtime already running (PID $(cat "$PID_RUNTIME"))"
    return 0
  fi
  build
  export GEEGOO_CONFIG="$CONFIG_PATH"
  nohup "$BIN_DIR/agentRuntimeServer" > "$LOG_RUNTIME" 2>&1 &
  echo $! > "$PID_RUNTIME"
  log "agent-runtime :${PORT_RUNTIME} PID=$(cat "$PID_RUNTIME") log=${LOG_RUNTIME}"
}

stop_runtime() {
  if [[ -f "$PID_RUNTIME" ]]; then
    kill "$(cat "$PID_RUNTIME")" 2>/dev/null || true
    rm -f "$PID_RUNTIME"
  fi
  pkill -f 'agentRuntimeServer' 2>/dev/null || true
  log "agent-runtime stopped"
}

status_runtime() {
  if [[ -f "$PID_RUNTIME" ]] && kill -0 "$(cat "$PID_RUNTIME")" 2>/dev/null; then
    echo "agent-runtime running PID=$(cat "$PID_RUNTIME")"
    curl -sf "http://127.0.0.1:${PORT_RUNTIME}/health" && echo || true
  else
    echo "agent-runtime not running"
  fi
}

case "${1:-help}" in
  build) build ;;
  start-runtime) start_runtime ;;
  stop-runtime) stop_runtime ;;
  restart-runtime) stop_runtime; start_runtime ;;
  status) status_runtime ;;
  *)
    echo "Usage: $0 {build|start-runtime|stop-runtime|restart-runtime|status}"
    exit 1
    ;;
esac
