#!/usr/bin/env bash
# GeeGoo Agent one-line installer.
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
DATA_DIR="$GEEGOO_HOME/data"
BIN_DIR="$GEEGOO_HOME/bin"
BINARY_PATH="$INSTALL_DIR/geegoo"

echo "==> GeeGoo Agent install"
echo "    home: $GEEGOO_HOME"
echo "    repo: $INSTALL_DIR"

mkdir -p "$GEEGOO_HOME" "$DATA_DIR" "$BIN_DIR"

if [ -d "$INSTALL_DIR/.git" ]; then
  echo "==> updating existing clone"
  git -C "$INSTALL_DIR" pull --ff-only
else
  echo "==> cloning $GEEGOO_REPO"
  git clone "$GEEGOO_REPO" "$INSTALL_DIR"
fi

if ! command -v go >/dev/null 2>&1; then
  echo "ERROR: go not found. Install Go 1.20+ first." >&2
  exit 1
fi

GO_VERSION="$(go env GOVERSION | sed 's/^go//')"
GO_MAJOR="${GO_VERSION%%.*}"
GO_MINOR_PATCH="${GO_VERSION#*.}"
GO_MINOR="${GO_MINOR_PATCH%%.*}"
if [ "${GO_MAJOR:-0}" -lt 1 ] || { [ "$GO_MAJOR" -eq 1 ] && [ "${GO_MINOR:-0}" -lt 20 ]; }; then
  echo "ERROR: Go 1.20+ is required. Found $(go env GOVERSION)." >&2
  exit 1
fi

echo "==> building geegoo"
(
  cd "$INSTALL_DIR"
  go build -o "$BINARY_PATH" ./cmd/geegoo
)

echo "==> linking geegoo into $BIN_DIR"
ln -sf "$BINARY_PATH" "$BIN_DIR/geegoo"

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
  geegoo setup --config "$CONFIG_PATH" --force
  echo "==> created $CONFIG_PATH"
fi

echo ""
echo "Install complete."
echo "  geegoo setup    # write default config"
echo "  geegoo doctor   # check connectivity"
echo "  geegoo chat     # start chat"
echo "  geegoo update   # pull and rebuild"
echo ""
echo "If geegoo is not found, run: source ~/.bashrc"

if [ "${GEEGOO_SKIP_SETUP:-0}" != "1" ] && [ -t 0 ]; then
  echo ""
  read -r -p "Run geegoo setup now? [y/N] " ans
  if [ "$ans" = "y" ] || [ "$ans" = "Y" ]; then
    geegoo setup --config "$CONFIG_PATH" --force
  fi
fi
