#!/usr/bin/env python3
from __future__ import annotations

import json
from pathlib import Path

import paramiko

DEPLOY = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")


def ssh(cmd: str, timeout: int = 60) -> str:
    cfg = json.loads(DEPLOY.read_text(encoding="utf-8"))
    s = cfg["targets"]["geegoo-agent"]["ssh"]
    c = paramiko.SSHClient()
    c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    c.connect(s["host"], username=s["user"], password=s.get("password"), timeout=30)
    _, o, e = c.exec_command(cmd, timeout=timeout)
    out = o.read().decode("utf-8", errors="replace")
    err = e.read().decode("utf-8", errors="replace")
    c.close()
    return out + err


def main() -> None:
    print(ssh("cd /home/ubuntu/.geegoo/geegoo-agent && git log -1 --oneline"))
    print(ssh("grep GEEGOO_AGENT_RUNTIME_API_KEY /home/ubuntu/.geegoo/agent.env || true"))
    print(
        ssh(
            'python3 -c "import json; c=json.load(open(\'/home/ubuntu/.geegoo/config.json\')); '
            'llm=c.get(\'llm\',{}); print({k: (v[:8]+\'...\'+v[-4:] if k==\'token_key\' and isinstance(v,str) and len(v)>12 else v) for k,v in llm.items()})"'
        )
    )
    print(
        ssh(
            "curl -s -o /dev/null -w 'completions:%{http_code}\\n' "
            "-X POST http://127.0.0.1:3400/v1/chat/completions -H 'Content-Type: application/json' -d '{}'"
        )
    )
    print(
        ssh(
            "curl -s -o /dev/null -w 'stream:%{http_code}\\n' "
            "-X POST http://127.0.0.1:3400/v1/chat/stream -H 'Content-Type: application/json' -d '{}'"
        )
    )


if __name__ == "__main__":
    main()
