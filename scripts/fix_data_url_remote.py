#!/usr/bin/env python3
"""Fix GeeGooData URL: rebuild geegoo, config, env wrapper."""
import json
from pathlib import Path
import paramiko

DEPLOY = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")
INSTALL = "/home/ubuntu/.geegoo/geegoo-agent"
BIN = "/home/ubuntu/.geegoo/bin"
CONFIG = "/home/ubuntu/.geegoo/config.json"
ENV_FILE = "/home/ubuntu/.geegoo/agent.env"
DATA_URL = "http://47.80.14.120:3300"


def run(c, cmd, timeout=300):
    _, o, e = c.exec_command(cmd, timeout=timeout)
    return o.channel.recv_exit_status(), (o.read() + e.read()).decode()


def main():
    cfg = json.loads(DEPLOY.read_text(encoding="utf-8-sig"))
    s = cfg["targets"]["geegoo-agent"]["ssh"]
    c = paramiko.SSHClient()
    c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    c.connect(s["host"], username=s["user"], password=s.get("password"), timeout=60)

    sftp = c.open_sftp()
    with sftp.open(CONFIG, "r") as f:
        raw = json.loads(f.read().decode())
    raw["data_base_url"] = DATA_URL
    with sftp.open(CONFIG, "w") as f:
        f.write(json.dumps(raw, indent=2, ensure_ascii=False).encode() + b"\n")

    env_body = f"""export GEEGOO_BOT_MCP_URL=http://118.195.135.97:3120
export GEEGOO_SIGNAL_CATALOG_API_URL=http://146.56.225.252:3210
export GEEGOO_DATA_HTTP_URL={DATA_URL}
export GEEGOO_CONFIG={CONFIG}
export PATH={BIN}:/usr/local/go/bin:$PATH
"""
    with sftp.open(ENV_FILE, "w") as f:
        f.write(env_body.encode())
    sftp.close()

    for cmd in [
        f"cd {INSTALL} && git fetch origin main && git reset --hard origin/main",
        f"cd {INSTALL} && bash start.sh build",
        f"mv -f {BIN}/geegoo {BIN}/geegoo.bin",
    ]:
        print(f">>> {cmd}")
        code, out = run(c, cmd, 600)
        print(out[-1500:] if out else f"exit {code}")
        if code != 0:
            c.close()
            return code

    wrapper = f"""#!/usr/bin/env bash
set -a
# shellcheck disable=SC1091
source {ENV_FILE}
set +a
exec {BIN}/geegoo.bin "$@"
"""
    sftp = c.open_sftp()
    with sftp.open(f"{BIN}/geegoo", "w") as f:
        f.write(wrapper.encode())
    sftp.close()
    run(c, f"chmod +x {BIN}/geegoo {BIN}/geegoo.bin")

    grep_cmd = f"grep -q 'source {ENV_FILE}' ~/.bashrc || echo 'source {ENV_FILE}' >> ~/.bashrc"
    run(c, grep_cmd)

    _, out = run(c, "geegoo doctor 2>&1", 120)
    print("\n=== doctor ===\n", out)
    c.close()


if __name__ == "__main__":
    main()
