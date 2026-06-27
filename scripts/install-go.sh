#!/usr/bin/env bash
# GeeGooAgent Go installer (main branch). Replaces Python venv install on servers.
#
#   curl -fsSL .../install-go.sh | bash
# Options (env):
#   GEEGOO_HOME           default ~/.geegoo
#   GEEGOO_INSTALL_DIR    default $GEEGOO_HOME/geegoo-agent
#   GEEGOO_REPO           default git@github.com:ghsemail/GeeGooAgent.git
#   GEEGOO_SKIP_BUILD=1   skip go build

set -euo pipefail

GEEGOO_HOME="${GEEGOO_HOME:-$HOME/.geegoo}"
INSTALL_DIR="${GEEGOO_INSTALL_DIR:-$GEEGOO_HOME/geegoo-agent}"
GEEGOO_REPO="${GEEGOO_REPO:-git@github.com:ghsemail/GeeGooAgent.git}"
CONFIG_PATH="${GEEGOO_CONFIG:-$GEEGOO_HOME/config.json}"
DATA_DIR="${GEEGOO_HOME}/data"
BIN_DIR="${GEEGOO_HOME}/bin"

echo "==> GeeGoo Agent (Go) install"
echo "    home:   $GEEGOO_HOME"
echo "    repo:   $INSTALL_DIR"

mkdir -p "$GEEGOO_HOME" "$DATA_DIR" "$BIN_DIR"
export PATH="/usr/local/go/bin:${PATH:-}"

if ! command -v go >/dev/null 2>&1; then
  echo "ERROR: go not found. Install Go 1.22+ (e.g. /usr/local/go) first." >&2
  exit 1
fi

if [ -d "$INSTALL_DIR/.git" ]; then
  echo "==> updating existing clone"
  git -C "$INSTALL_DIR" fetch origin main
  git -C "$INSTALL_DIR" reset --hard origin/main
else
  echo "==> cloning $GEEGOO_REPO"
  git clone "$GEEGOO_REPO" "$INSTALL_DIR"
  git -C "$INSTALL_DIR" checkout main
fi

if [ "${GEEGOO_SKIP_BUILD:-0}" != "1" ]; then
  echo "==> building binaries"
  (cd "$INSTALL_DIR" && GOPROXY="${GOPROXY:-https://goproxy.cn,direct}" bash start.sh build)
fi

if [ ! -f "$CONFIG_PATH" ]; then
  cp "$INSTALL_DIR/config.example.json" "$CONFIG_PATH"
  chmod 600 "$CONFIG_PATH"
  python3 - <<PY
import json
from pathlib import Path
p = Path("$CONFIG_PATH")
raw = json.loads(p.read_text(encoding="utf-8"))
raw["output_dir"] = "$DATA_DIR"
p.write_text(json.dumps(raw, indent=2, ensure_ascii=False) + "\n", encoding="utf-8")
PY
  echo "==> created $CONFIG_PATH (edit geegoo_url → GeeGooBot :3120)"
fi

PATH_LINE="export PATH=\"$BIN_DIR:/usr/local/go/bin:\$PATH\""
CONFIG_LINE="export GEEGOO_CONFIG=\"$CONFIG_PATH\""
HOME_LINE="export GEEGOO_HOME=\"$GEEGOO_HOME\""
for rc in "$HOME/.bashrc" "$HOME/.profile"; do
  if [ -f "$rc" ]; then
    grep -qF 'GEEGOO_HOME=' "$rc" 2>/dev/null || echo "$HOME_LINE" >> "$rc"
    grep -qF 'GEEGOO_CONFIG=' "$rc" 2>/dev/null || echo "$CONFIG_LINE" >> "$rc"
    grep -qF "$BIN_DIR" "$rc" 2>/dev/null || echo "$PATH_LINE" >> "$rc"
  fi
done

echo ""
echo "安装完成 (Go)。"
echo "  geegoo doctor"
echo "  bash $INSTALL_DIR/start.sh start-runtime   # :3400"
echo "  geegoo run pre_market --dry-run"
