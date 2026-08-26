#!/usr/bin/env bash
# GeeGooAgent process manager (Go main branch).
set -euo pipefail

APP_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$APP_DIR"

GEEGOO_HOME="${GEEGOO_HOME:-$HOME/.geegoo}"
set -a
# shellcheck disable=SC1091
source "$GEEGOO_HOME/agent.env" 2>/dev/null || true
set +a
BIN_DIR="${GEEGOO_BIN_DIR:-$GEEGOO_HOME/bin}"
CONFIG_PATH="${GEEGOO_CONFIG:-$GEEGOO_HOME/config.json}"
mkdir -p "$BIN_DIR"

export PATH="/usr/local/go/bin:${BIN_DIR}:${PATH:-}"
export GOPROXY="${GOPROXY:-https://goproxy.cn,direct}"

PORT_RUNTIME="${GEEGOO_AGENT_RUNTIME_PORT:-3400}"
LOG_RUNTIME="${APP_DIR}/agent-runtime.out"
PID_RUNTIME="${APP_DIR}/agent-runtime.pid"
LOG_SCHEDULER="${APP_DIR}/scheduler.out"
PID_SCHEDULER="${APP_DIR}/scheduler.pid"
LOG_GATEWAY="${APP_DIR}/gateway.out"
PID_GATEWAY="${APP_DIR}/gateway.pid"

log() { echo "[GeeGooAgent] $*"; }

install_geegoo_wrapper() {
  local wrapper="$BIN_DIR/geegoo"
  local tmp="$BIN_DIR/.geegoo.wrapper.new"
  if [[ -L "$wrapper" ]]; then
    rm -f "$wrapper"
  fi
  cat > "$tmp" <<EOF
#!/usr/bin/env bash
set -a
# shellcheck disable=SC1091
source "\${GEEGOO_HOME:-$HOME/.geegoo}/agent.env" 2>/dev/null || true
set +a
exec "$BIN_DIR/geegoo.bin" "\$@"
EOF
  chmod +x "$tmp"
  mv -f "$tmp" "$wrapper"
  chmod +x "$BIN_DIR/geegoo.bin" 2>/dev/null || true
}

build() {
  log "building geegoo + agentRuntimeServer..."
  go build -o "$BIN_DIR/geegoo.bin" ./cmd/geegoo
  go build -o "$BIN_DIR/agentRuntimeServer" ./cmd/agent-runtime
  install_geegoo_wrapper
}

start_runtime() {
  if [[ -f "$PID_RUNTIME" ]] && kill -0 "$(cat "$PID_RUNTIME")" 2>/dev/null; then
    log "agent-runtime already running (PID $(cat "$PID_RUNTIME"))"
    return 0
  fi
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
  pkill -f '/geegoo-agent/bin/agent-runtime' 2>/dev/null || true
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

start_scheduler() {
  if [[ -f "$PID_SCHEDULER" ]] && kill -0 "$(cat "$PID_SCHEDULER")" 2>/dev/null; then
    log "scheduler already running (PID $(cat "$PID_SCHEDULER"))"
    return 0
  fi
  export GEEGOO_CONFIG="$CONFIG_PATH"
  nohup "$BIN_DIR/geegoo" scheduler run --config "$CONFIG_PATH" > "$LOG_SCHEDULER" 2>&1 &
  echo $! > "$PID_SCHEDULER"
  log "scheduler started PID=$(cat "$PID_SCHEDULER") log=${LOG_SCHEDULER}"
}

stop_scheduler() {
  if [[ -f "$PID_SCHEDULER" ]]; then
    kill "$(cat "$PID_SCHEDULER")" 2>/dev/null || true
    rm -f "$PID_SCHEDULER"
  fi
  pkill -f 'geegoo.*scheduler run' 2>/dev/null || true
  pkill -f 'geegoo.bin scheduler run' 2>/dev/null || true
  log "scheduler stopped"
}

status_scheduler() {
  if [[ -f "$PID_SCHEDULER" ]] && kill -0 "$(cat "$PID_SCHEDULER")" 2>/dev/null; then
    echo "scheduler running PID=$(cat "$PID_SCHEDULER")"
    "$BIN_DIR/geegoo" scheduler list --config "$CONFIG_PATH" 2>/dev/null || true
  else
    echo "scheduler not running"
  fi
}

start_gateway() {
  if [[ -f "$PID_GATEWAY" ]] && kill -0 "$(cat "$PID_GATEWAY")" 2>/dev/null; then
    log "gateway already running (PID $(cat "$PID_GATEWAY"))"
    return 0
  fi
  export GEEGOO_CONFIG="$CONFIG_PATH"
  cd "$APP_DIR"
  nohup "$BIN_DIR/geegoo" gateway run --config "$CONFIG_PATH" > "$LOG_GATEWAY" 2>&1 &
  echo $! > "$PID_GATEWAY"
  log "gateway started PID=$(cat "$PID_GATEWAY") log=${LOG_GATEWAY}"
}

stop_gateway() {
  if [[ -f "$PID_GATEWAY" ]]; then
    kill "$(cat "$PID_GATEWAY")" 2>/dev/null || true
    rm -f "$PID_GATEWAY"
  fi
  pkill -f 'geegoo.*gateway run' 2>/dev/null || true
  pkill -f 'geegoo.bin gateway run' 2>/dev/null || true
  log "gateway stopped"
}

status_gateway() {
  if [[ -f "$PID_GATEWAY" ]] && kill -0 "$(cat "$PID_GATEWAY")" 2>/dev/null; then
    echo "gateway running PID=$(cat "$PID_GATEWAY")"
    "$BIN_DIR/geegoo" gateway status --config "$CONFIG_PATH" 2>/dev/null || true
  else
    echo "gateway not running"
  fi
}

case "${1:-help}" in
  build) build ;;
  start-runtime) start_runtime ;;
  stop-runtime) stop_runtime ;;
  restart-runtime) stop_runtime; build; start_runtime ;;
  status) status_runtime; status_scheduler; status_gateway ;;
  start-scheduler) start_scheduler ;;
  stop-scheduler) stop_scheduler ;;
  restart-scheduler) stop_scheduler; build; start_scheduler ;;
  start-gateway) start_gateway ;;
  stop-gateway) stop_gateway ;;
  restart-gateway) stop_gateway; build; start_gateway ;;
  start-all) start_runtime; start_scheduler; start_gateway ;;
  stop-all) stop_gateway; stop_scheduler; stop_runtime ;;
  restart-all) stop_gateway; stop_scheduler; stop_runtime; build; start_runtime; start_scheduler; start_gateway ;;
  *)
    echo "Usage: $0 {build|start-runtime|stop-runtime|restart-runtime|start-scheduler|stop-scheduler|restart-scheduler|start-gateway|stop-gateway|restart-gateway|start-all|stop-all|restart-all|status}"
    exit 1
    ;;
esac
