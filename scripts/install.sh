#!/usr/bin/env bash
# GeeGoo Agent one-line installer (Hermes-style).
#
#   curl -fsSL https://raw.githubusercontent.com/ghsemail/GeeGooAgent/main/scripts/install.sh | bash
#
# Options (env):
#   GEEGOO_HOME       default ~/.geegoo
#   GEEGOO_REPO       default git@github.com:ghsemail/GeeGooAgent.git
#   GEEGOO_SKIP_SETUP=1   skip interactive geegoo setup after install

set -euo pipefail

GEEGOO_HOME="${GEEGOO_HOME:-$HOME/.geegoo}"
INSTALL_DIR="${GEEGOO_INSTALL_DIR:-$GEEGOO_HOME/geegoo-agent}"
GEEGOO_REPO="${GEEGOO_REPO:-git@github.com:ghsemail/GeeGooAgent.git}"
CONFIG_PATH="${GEEGOO_CONFIG:-$GEEGOO_HOME/config.json}"
DATA_DIR="${GEEGOO_HOME}/data"
VENV_DIR="$INSTALL_DIR/venv"
BIN_DIR="$GEEGOO_HOME/bin"

echo "==> GeeGoo Agent install"
echo "    home:   $GEEGOO_HOME"
echo "    repo:   $INSTALL_DIR"

mkdir -p "$GEEGOO_HOME" "$DATA_DIR" "$BIN_DIR"

if [ -d "$INSTALL_DIR/.git" ]; then
  echo "==> updating existing clone"
  git -C "$INSTALL_DIR" pull --ff-only
else
  echo "==> cloning $GEEGOO_REPO"
  git clone "$GEEGOO_REPO" "$INSTALL_DIR"
fi

if ! command -v python3 >/dev/null 2>&1; then
  echo "ERROR: python3 not found. Install Python 3.11+ first." >&2
  exit 1
fi

echo "==> creating venv"
if [ ! -d "$VENV_DIR" ]; then
  python3 -m venv "$VENV_DIR"
fi
# shellcheck disable=SC1091
source "$VENV_DIR/bin/activate"
pip install -U pip wheel -q
pip install -e "$INSTALL_DIR[dev]" -q

echo "==> linking geegoo into $BIN_DIR"
ln -sf "$VENV_DIR/bin/geegoo" "$BIN_DIR/geegoo"
ln -sf "$VENV_DIR/bin/geegoo-agent" "$BIN_DIR/geegoo-agent" 2>/dev/null || true

PATH_LINE="export PATH=\"$BIN_DIR:\$PATH\""
CONFIG_LINE="export GEEGOO_CONFIG=\"$CONFIG_PATH\""
HOME_LINE="export GEEGOO_HOME=\"$GEEGOO_HOME\""
for rc in "$HOME/.bashrc" "$HOME/.profile"; do
  if [ -f "$rc" ]; then
    grep -qF 'GEEGOO_HOME=' "$rc" 2>/dev/null || echo "$HOME_LINE" >> "$rc"
    grep -qF 'GEEGOO_CONFIG=' "$rc" 2>/dev/null || echo "$CONFIG_LINE" >> "$rc"
    grep -qF "$BIN_DIR" "$rc" 2>/dev/null || echo "$PATH_LINE" >> "$rc"
  fi
done

export PATH="$BIN_DIR:$PATH"
export GEEGOO_CONFIG="$CONFIG_PATH"
export GEEGOO_HOME="$GEEGOO_HOME"

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
  echo "==> created $CONFIG_PATH"
fi

echo ""
echo "安装完成。"
echo "  geegoo setup    # 配置 LLM + mcp_token"
echo "  geegoo doctor   # 检查连通性"
echo "  geegoo chat     # 开始对话"
echo "  geegoo update   # 更新到最新版"
echo ""
echo "若命令未找到，请执行: source ~/.bashrc"

if [ "${GEEGOO_SKIP_SETUP:-0}" != "1" ] && [ -t 0 ]; then
  echo ""
  read -r -p "是否现在运行 geegoo setup? [Y/n] " ans
  if [ -z "$ans" ] || [ "$ans" = "y" ] || [ "$ans" = "Y" ]; then
    geegoo setup --config "$CONFIG_PATH" --skip-install
  fi
fi
