#!/usr/bin/env python3
"""Deploy TurnPlan live eval: sync main on agent host + migrate eval DB seeds.

Prefer:
  python scripts/ops/deploy/deploy_geegoo_agent.py
  python scripts/eval/migrate_turnplan_eval_db.py
"""
from __future__ import annotations

import subprocess
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[3]


def main() -> int:
    deploy = ROOT / "scripts/ops/deploy/deploy_geegoo_agent.py"
    migrate = ROOT / "scripts/eval/migrate_turnplan_eval_db.py"
    for script in (deploy, migrate):
        print(f"\n=== {script.name} ===\n", flush=True)
        code = subprocess.call([sys.executable, str(script)])
        if code != 0:
            return code
    print("\n=== TurnPlan live eval deploy OK ===")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
