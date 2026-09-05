#!/usr/bin/env python3
"""Fix agent chat 503: postgres session store requested but GEEGOO_PG_DSN is not connected.

Typical cause: agent-runtime started while PostgreSQL was down; process keeps running
with a.PG=nil even after PG becomes reachable. This script:
  1) probes GEEGOO_PG_DSN from agent.env
  2) restarts agent-runtime (re-open PG pool)
  3) if PG still unreachable, sets GEEGOO_SESSION_STORE=sqlite and restarts again
  4) smoke-tests POST /v1/chat/stream locally

Requires remote-deploy deploy.json (geegoo-agent target).
"""
from __future__ import annotations

import json
import re
import sys
import textwrap
from pathlib import Path

import paramiko

DEPLOY = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")
AGENT_ENV = "/home/ubuntu/.geegoo/agent.env"
REPO = "/home/ubuntu/.geegoo/geegoo-agent"

REMOTE = textwrap.dedent(
    r"""
    import json, os, re, subprocess, urllib.error, urllib.request

    AGENT_ENV = "/home/ubuntu/.geegoo/agent.env"
    REPO = "/home/ubuntu/.geegoo/geegoo-agent"

    def load_env(path):
        env = {}
        if not os.path.isfile(path):
            return env
        for line in open(path, encoding="utf-8"):
            line = line.strip()
            if not line or line.startswith("#") or "=" not in line:
                continue
            k, v = line.split("=", 1)
            env[k] = v.strip().strip('"').strip("'")
        return env

    def set_kv(path, key, val):
        pat = re.compile(rf"^{re.escape(key)}=")
        lines = open(path, encoding="utf-8").read().splitlines() if os.path.isfile(path) else []
        out, done = [], False
        for ln in lines:
            if pat.match(ln):
                out.append(f'{key}="{val}"')
                done = True
            else:
                out.append(ln)
        if not done:
            out.append(f'{key}="{val}"')
        open(path, "w", encoding="utf-8").write("\n".join(out) + "\n")

    def restart_runtime():
        cmd = f"bash -lc 'set -a; source {AGENT_ENV} 2>/dev/null || true; set +a; cd {REPO} && bash start.sh restart-runtime && sleep 2'"
        return subprocess.run(cmd, shell=True, capture_output=True, text=True)

    def pg_ping(dsn):
        if not dsn:
            return False, "GEEGOO_PG_DSN empty"
        r = subprocess.run(["psql", dsn, "-At", "-c", "SELECT 1"], capture_output=True, text=True, timeout=15)
        if r.returncode == 0 and "1" in (r.stdout or ""):
            return True, "psql ok"
        err = (r.stderr or r.stdout or "psql failed").strip()
        return False, err[:300]

    def runtime_pg_connected():
        pid = subprocess.run(["bash", "-lc", "pgrep -f agentRuntimeServer | head -1"], capture_output=True, text=True)
        pid = (pid.stdout or "").strip()
        if not pid:
            return False, "agentRuntimeServer not running"
        env = {}
        for item in open(f"/proc/{pid}/environ", "rb").read().split(b"\0"):
            if b"=" in item:
                k, v = item.split(b"=", 1)
                env[k.decode()] = v.decode()
        dsn = env.get("GEEGOO_PG_DSN", "")
        store = env.get("GEEGOO_SESSION_STORE", "")
        # indirect check: chat stream should not 503 on session store
        key = env.get("GEEGOO_AGENT_RUNTIME_API_KEY", "")
        if not key:
            return False, "runtime api key missing in process env"
        req = urllib.request.Request(
            "http://127.0.0.1:3400/v1/chat/stream",
            data=json.dumps({"message": "ping"}).encode(),
            headers={
                "Authorization": f"Bearer {key}",
                "Content-Type": "application/json",
                "X-MCP-Token": "probe",
            },
            method="POST",
        )
        try:
            with urllib.request.urlopen(req, timeout=20) as resp:
                body = resp.read(400).decode("utf-8", errors="replace")
                return True, f"chat HTTP {resp.status}: {body[:120]}"
        except urllib.error.HTTPError as e:
            body = e.read(400).decode("utf-8", errors="replace")
            if "postgres session store requested" in body:
                return False, body[:200]
            return True, f"chat HTTP {e.code} (session store ok): {body[:120]}"
        except Exception as ex:
            return False, str(ex)

    env = load_env(AGENT_ENV)
    dsn = env.get("GEEGOO_PG_DSN", "")
    store = env.get("GEEGOO_SESSION_STORE", "sqlite")
    print("GEEGOO_SESSION_STORE", store)
    print("GEEGOO_PG_DSN", "set" if dsn else "missing")

    ok, detail = pg_ping(dsn)
    print("pg_ping", ok, detail)

    print("\n=== restart runtime (attempt 1) ===")
    r = restart_runtime()
    print((r.stdout or r.stderr)[-800:])
    ok2, detail2 = runtime_pg_connected()
    print("session_store_probe", ok2, detail2)
    if ok2:
        print("RESULT ok restart_only")
        raise SystemExit(0)

    if store.lower() in ("postgres", "pg") and not ok:
        print("\n=== PG unreachable; fallback GEEGOO_SESSION_STORE=sqlite ===")
        set_kv(AGENT_ENV, "GEEGOO_SESSION_STORE", "sqlite")
        r = restart_runtime()
        print((r.stdout or r.stderr)[-800:])
        ok3, detail3 = runtime_pg_connected()
        print("session_store_probe", ok3, detail3)
        if ok3:
            print("RESULT ok sqlite_fallback")
            raise SystemExit(0)

    print("RESULT fail", detail2)
    raise SystemExit(1)
    """
)


def main() -> int:
    sys.stdout.reconfigure(encoding="utf-8", errors="replace")
    if not DEPLOY.is_file():
        print(f"Missing {DEPLOY}; run from a machine with remote-deploy credentials.")
        return 1
    cfg = json.loads(DEPLOY.read_text(encoding="utf-8"))
    s = cfg["targets"]["geegoo-agent"]["ssh"]
    c = paramiko.SSHClient()
    c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    c.connect(s["host"], username=s["user"], password=s.get("password"), timeout=60)
    try:
        _, o, e = c.exec_command(f"python3 - <<'PY'\n{REMOTE}\nPY", timeout=300)
        out = (o.read() + e.read()).decode("utf-8", errors="replace")
        print(out)
        return 0 if "RESULT ok" in out else 1
    finally:
        c.close()


if __name__ == "__main__":
    raise SystemExit(main())
