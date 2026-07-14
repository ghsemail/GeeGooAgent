#!/usr/bin/env python3
"""Patch GeeGooAgent server config to GeeGoo Go 3xxx endpoints."""
from __future__ import annotations

import json
from pathlib import Path

import paramiko

DEPLOY = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")


def ssh_run(host_cfg: dict, cmd: str) -> str:
    client = paramiko.SSHClient()
    client.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    client.connect(
        hostname=host_cfg["host"],
        port=int(host_cfg.get("port", 22)),
        username=host_cfg["user"],
        password=host_cfg.get("password"),
        timeout=30,
    )
    _, stdout, stderr = client.exec_command(cmd, timeout=120)
    out = stdout.read().decode("utf-8", errors="replace")
    err = stderr.read().decode("utf-8", errors="replace")
    client.close()
    if err.strip():
        raise RuntimeError(err.strip())
    return out


def read_key(host_cfg: dict, path: str, var: str) -> str:
    out = ssh_run(host_cfg, f"grep '^{var}=' {path} | head -1 | cut -d= -f2-")
    return out.strip().strip('"').strip("'")


def main() -> None:
    cfg = json.loads(DEPLOY.read_text(encoding="utf-8"))
    bot = cfg["targets"]["geegoo-bot"]["ssh"]
    sig = cfg["targets"]["geegoo-signal"]["ssh"]
    agent = cfg["targets"]["geegoo-agent"]["ssh"]

    bot_key = read_key(bot, "/home/ubuntu/apps/GeeGooBot/.env", "GEEGOO_BOT_MCP_API_KEY")
    sig_key = read_key(sig, "/root/apps/GeeGooSignal/.env", "GEEGOO_SIGNAL_SIGNAL_API_KEY")
    cat_key = read_key(sig, "/root/apps/GeeGooSignal/.env", "GEEGOO_SIGNAL_CATALOG_API_KEY")
    ana_key = read_key(sig, "/root/apps/GeeGooSignal/.env", "GEEGOO_SIGNAL_ANALYZE_API_KEY")

    patch = {
        "base_url": "http://118.195.135.97:3120",
        "geegoo_url": "http://118.195.135.97:3120",
        "api_key": bot_key,
        "geegoo_api_key": bot_key,
        "signal_base_url": "http://146.56.225.252:3210",
        "signal_api_url": "http://146.56.225.252:3200",
        "signal_api_key": sig_key,
        "signal_catalog_api_key": cat_key,
        "signal_analyze_api_url": "http://146.56.225.252:3230",
        "signal_analyze_api_key": ana_key,
        "data_base_url": "http://47.80.14.120:3300",
    }

    env_lines = [
        "export GEEGOO_BOT_MCP_URL=http://118.195.135.97:3120",
        f"export GEEGOO_BOT_MCP_API_KEY={bot_key}",
        "export GEEGOO_SIGNAL_CATALOG_API_URL=http://146.56.225.252:3210",
        "export GEEGOO_SIGNAL_SIGNAL_API_URL=http://146.56.225.252:3200",
        "export GEEGOO_SIGNAL_ANALYZE_API_URL=http://146.56.225.252:3230",
        f"export GEEGOO_SIGNAL_SIGNAL_API_KEY={sig_key}",
        f"export GEEGOO_SIGNAL_CATALOG_API_KEY={cat_key}",
        f"export GEEGOO_SIGNAL_ANALYZE_API_KEY={ana_key}",
        "export GEEGOO_DATA_HTTP_URL=http://47.80.14.120:3300",
        "export GEEGOO_CONFIG=/home/ubuntu/.geegoo/config.json",
        "export PATH=/home/ubuntu/.geegoo/bin:/usr/local/go/bin:$PATH",
    ]
    env_body = "\n".join(env_lines) + "\n"

    patch_json = json.dumps(patch, ensure_ascii=False)
    remote_py = f"""python3 - <<'PY'
import json
import urllib.request
from pathlib import Path

p = Path.home() / '.geegoo' / 'config.json'
cfg = json.loads(p.read_text(encoding='utf-8'))
patch = json.loads({patch_json!r})
cfg.update(patch)
hosts = set(cfg.get('sandbox', {{}}).get('allowed_hosts', []))
hosts.update(['118.195.135.97', '146.56.225.252', '47.80.14.120'])
cfg.setdefault('sandbox', {{}})['allowed_hosts'] = sorted(hosts)

cat_key = (cfg.get('signal_catalog_api_key') or '').strip()
cat_url = (cfg.get('signal_base_url') or 'http://146.56.225.252:3210').rstrip('/')
if cat_key:
    req = urllib.request.Request(
        cat_url + '/queryModel',
        data=json.dumps({{'type': 'configured'}}).encode(),
        headers={{
            'Content-Type': 'application/json',
            'Authorization': f'Bearer {{cat_key}}',
        }},
        method='POST',
    )
    doc = json.loads(urllib.request.urlopen(req, timeout=15).read().decode())
    token = (doc.get('token') or '').strip()
    base = (doc.get('base_url') or '').strip().rstrip('/')
    model = (doc.get('name') or doc.get('display_name') or 'gpt-5.5').strip()
    if token and base:
        llm = cfg.setdefault('llm', {{}})
        llm['token_key'] = token
        llm['base_url'] = base
        llm['model'] = model
        llm['provider'] = 'openai'
        llm['use_ops_model'] = True
        cfg['llm'] = llm
        print('llm synced from queryModel', 'model', model, 'token_suffix', token[-4:])
    else:
        print('warn: queryModel missing token/base_url; llm block unchanged')

p.write_text(json.dumps(cfg, ensure_ascii=False, indent=2) + '\\n', encoding='utf-8')
print('updated', p)
for k in ['base_url','geegoo_url','signal_base_url','signal_api_url','data_base_url']:
    print(k, '=', cfg.get(k))
PY"""
    print(ssh_run(agent, remote_py))

    env_remote = "cat > /home/ubuntu/.geegoo/agent.env <<'EOF'\n" + env_body + "EOF\n"
    print(ssh_run(agent, env_remote))
    print("updated /home/ubuntu/.geegoo/agent.env")

    print(ssh_run(agent, "export PATH=$HOME/.geegoo/bin:$PATH; geegoo doctor 2>&1 || true"))


if __name__ == "__main__":
    main()
