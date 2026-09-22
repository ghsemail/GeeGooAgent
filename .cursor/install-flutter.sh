#!/usr/bin/env bash
# Idempotent Flutter SDK for Cloud Agent (trading_operation web builds).
set -euo pipefail

FLUTTER_ROOT="${FLUTTER_ROOT:-$HOME/flutter}"
export PATH="$FLUTTER_ROOT/bin:$PATH"

if [ ! -d "$FLUTTER_ROOT/.git" ]; then
  git clone https://github.com/flutter/flutter.git -b stable --depth 1 "$FLUTTER_ROOT"
fi

flutter config --enable-web --no-analytics >/dev/null 2>&1 || true
flutter --version | head -1
