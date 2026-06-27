#!/usr/bin/env bash
# Install Go toolchain to /usr/local/go if missing.
set -euo pipefail

GO_VERSION="${GO_VERSION:-1.22.10}"
ARCH="$(uname -m)"
case "$ARCH" in
  x86_64) GO_ARCH=amd64 ;;
  aarch64|arm64) GO_ARCH=arm64 ;;
  *) echo "unsupported arch: $ARCH" >&2; exit 1 ;;
esac

if command -v go >/dev/null 2>&1 || [ -x /usr/local/go/bin/go ]; then
  echo "go already installed: $(/usr/local/go/bin/go version 2>/dev/null || go version)"
  exit 0
fi

TMP="/tmp/go${GO_VERSION}.linux-${GO_ARCH}.tar.gz"
URL="https://go.dev/dl/go${GO_VERSION}.linux-${GO_ARCH}.tar.gz"
echo "==> downloading $URL"
curl -fsSL "$URL" -o "$TMP"
sudo rm -rf /usr/local/go
sudo tar -C /usr/local -xzf "$TMP"
rm -f "$TMP"
echo "==> $(/usr/local/go/bin/go version)"
