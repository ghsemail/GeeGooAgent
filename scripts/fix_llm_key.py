#!/usr/bin/env python3
"""Fix agent LLM credentials from catalog queryModel and verify chat."""
from __future__ import annotations

import json
from pathlib import Path

import paramiko

DEPLOY = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")


def ssh(target: str, cmd: str, timeout: int = 120) -> str:
    cfg = json.loads(DEPLOY.read_text(encoding="utf-8"))
    s = cfg["targets"][target]["ssh"]
    c = paramiko.SSHClient()
    c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    c.connect(s["host"], username=s["user"], password=s.get("password"), timeout=30)
    _, o, e = c.exec_command(cmd, timeout=timeout)
    out = o.read().decode("utf-8", errors="replace")
    err = e.read().decode("utf-8", errors="replace")
    code = o.channel.recv_exit_status()
    c.close()
    if code != 0 and not out.strip():
        raise RuntimeError(err.strip() or f"exit {code}")
    return out + err


REMOTE_PY = r"""
import json
import urllib.error
import urllib.request
from pathlib import Path

cfg_path = Path.home() / ".geegoo" / "config.json"
cfg = json.loads(cfg_path.read_text(encoding="utf-8"))
cat_key = (cfg.get("signal_catalog_api_key") or cfg.get("signal_api_key") or "").strip()
if not cat_key:
    raise SystemExit("missing signal_catalog_api_key in config.json")

body = json.dumps({"type": "configured"}).encode()
req = urllib.request.Request(
    cfg.get("signal_base_url", "http://146.56.225.252:3210").rstrip("/") + "/queryModel",
    data=body,
    headers={"Content-Type": "application/json", "Authorization": f"Bearer {cat_key}"},
    method="POST",
)
doc = json.loads(urllib.request.urlopen(req, timeout=15).read().decode())
token = (doc.get("token") or "").strip()
base = (doc.get("base_url") or "").strip().rstrip("/")
model = (doc.get("name") or doc.get("display_name") or "gpt-5.5").strip()
if not token or not base:
    raise SystemExit(f"queryModel incomplete: {doc}")

# OpenAI provider uses baseURL + /chat/completions
url = base + "/chat/completions"
payload = json.dumps({
    "model": model,
    "messages": [{"role": "user", "content": "reply ok"}],
    "max_tokens": 16,
}).encode()
test_req = urllib.request.Request(
    url,
    data=payload,
    headers={"Content-Type": "application/json", "Authorization": f"Bearer {token}"},
    method="POST",
)
try:
    with urllib.request.urlopen(test_req, timeout=30) as resp:
        print("probe_ok", resp.status, url, "model", model, "token_suffix", token[-4:])
except urllib.error.HTTPError as he:
    print("probe_fail", he.code, he.read()[:300].decode("utf-8", "replace"))
    raise SystemExit(1)

llm = cfg.setdefault("llm", {})
llm["token_key"] = token
llm["model"] = model
llm["base_url"] = base
llm["provider"] = "openai"
llm["use_ops_model"] = True
cfg["llm"] = llm
cfg_path.write_text(json.dumps(cfg, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
print("patched_config", "token_suffix", token[-4:], "base_url", base, "model", model)
"""


def main() -> None:
    print("=== agent-runtime log (LLM lines) ===")
    print(
        ssh(
            "geegoo-agent",
            "grep -E 'LLM|运营|gateway|401|queryModel' /home/ubuntu/.geegoo/geegoo-agent/agent-runtime.out | tail -20 || true",
        )
    )

    print("\n=== patch config from catalog queryModel + probe ===")
    print(ssh("geegoo-agent", f"python3 <<'PY'\n{REMOTE_PY}\nPY"))

    print("\n=== restart agent-runtime ===")
    repo = "/home/ubuntu/.geegoo/geegoo-agent"
    print(ssh("geegoo-agent", f"cd {repo} && bash start.sh restart-runtime && sleep 2 && curl -sf http://127.0.0.1:3400/health"))

    print("\n=== geegoo doctor ===")
    print(ssh("geegoo-agent", "export PATH=$HOME/.geegoo/bin:$PATH; geegoo doctor 2>&1 | tail -15"))

    smoke = """
import json, os, urllib.request

def load_env(path):
    env = {}
    for line in open(path):
        line = line.strip()
        if not line or line.startswith("#"):
            continue
        if line.startswith("export "):
            line = line[7:]
        if "=" not in line:
            continue
        k, v = line.split("=", 1)
        env[k] = v.strip().strip('"').strip("'")
    return env

cfg = json.load(open("/home/ubuntu/.geegoo/config.json"))
env = load_env("/home/ubuntu/.geegoo/agent.env")
runtime_key = env.get("GEEGOO_AGENT_RUNTIME_API_KEY", "").strip()
mcp = cfg.get("mcp_token", "")
body = json.dumps({
    "model": "geegoo-agent",
    "messages": [{"role": "user", "content": "只回复 ok"}],
}).encode()
headers = {"Content-Type": "application/json", "X-MCP-Token": mcp}
if runtime_key:
    headers["Authorization"] = f"Bearer {runtime_key}"
req = urllib.request.Request(
    "http://127.0.0.1:3400/v1/chat/completions",
    data=body,
    headers=headers,
    method="POST",
)
with urllib.request.urlopen(req, timeout=120) as resp:
    doc = json.loads(resp.read().decode())
choices = doc.get("choices") or []
text = (choices[0].get("message") or {}).get("content", "") if choices else ""
finish = choices[0].get("finish_reason") if choices else ""
print("finish:", finish, "text:", text[:200])
if finish == "error" or "Authentication Fails" in text:
    raise SystemExit(1)
print("OK")
"""
    print("\n=== chat/completions smoke ===")
    out = ssh("geegoo-agent", f"python3 <<'PY'\n{smoke}\nPY", timeout=180)
    print(out)
    if "OK" not in out:
        raise SystemExit("LLM smoke test still failing")
    print("OK: LLM smoke passed")


if __name__ == "__main__":
    main()
