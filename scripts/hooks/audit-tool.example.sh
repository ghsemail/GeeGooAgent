#!/usr/bin/env sh
# Example hook: append tool audit lines to ~/.geegoo/hooks/audit.log
# Configure in config.json:
#   "hooks": {
#     "tool_before": ["/path/to/geegoo-agent/scripts/hooks/audit-tool.example.sh"],
#     "tool_after": ["/path/to/geegoo-agent/scripts/hooks/audit-tool.example.sh"]
#   }
set -eu
LOG="${GEEGOO_HOOK_LOG:-$HOME/.geegoo/hooks/audit.log}"
mkdir -p "$(dirname "$LOG")"
payload="$(cat)"
printf '%s %s\n' "$(date -u +%Y-%m-%dT%H:%M:%SZ)" "$payload" >>"$LOG"
