#!/usr/bin/env python3
"""Install GeeGooAgent on remote server per README (install.sh | bash)."""

from __future__ import annotations

import sys
from pathlib import Path

import paramiko

HOST, USER, PASS = "119.45.16.112", "ubuntu", "Ghs@2024"
PROJECT = Path(__file__).resolve().parents[1]
INSTALL_SH = PROJECT / "scripts" / "install.sh"


def run(client: paramiko.SSHClient, cmd: str, timeout: int = 600) -> tuple[str, str, int]:
    _stdin, stdout, stderr = client.exec_command(cmd, timeout=timeout)
    code = stdout.channel.recv_exit_status()
    return (
        stdout.read().decode("utf-8", errors="replace"),
        stderr.read().decode("utf-8", errors="replace"),
        code,
    )


def main() -> int:
    client = paramiko.SSHClient()
    client.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    print(f"==> connect {USER}@{HOST}")
    client.connect(HOST, username=USER, password=PASS, timeout=25)

    setup_ssh = r"""
set -e
mkdir -p ~/.ssh && chmod 700 ~/.ssh
if [ ! -f ~/.ssh/id_ed25519 ]; then
  ssh-keygen -t ed25519 -N "" -f ~/.ssh/id_ed25519 -C "ubuntu@119.45.16.112"
fi
ssh-keyscan -H github.com >> ~/.ssh/known_hosts 2>/dev/null || true
chmod 600 ~/.ssh/id_ed25519 ~/.ssh/known_hosts 2>/dev/null || true
cat ~/.ssh/id_ed25519.pub
"""
    print("==> GitHub SSH key")
    out, err, code = run(client, setup_ssh, timeout=120)
    if code != 0:
        print(err or out, file=sys.stderr)
        return code
    pubkey = [ln for ln in out.splitlines() if ln.startswith("ssh-")][-1]
    print(f"    {pubkey[:72]}...")

    register = f"""
python3 - <<'PY'
import json, urllib.request, pathlib
pub = {pubkey!r}
token = pathlib.Path.home().joinpath(".geegoo/github_token").read_text().strip()
if not token:
    print("no_token_skip")
    raise SystemExit(0)
req = urllib.request.Request(
    "https://api.github.com/repos/ghsemail/GeeGooAgent/keys",
    headers={{"Authorization": f"Bearer {{token}}", "Accept": "application/vnd.github+json"}},
)
with urllib.request.urlopen(req, timeout=30) as r:
    keys = json.load(r)
fp = pub.split()[1]
if any(fp in k.get("key", "") for k in keys):
    print("deploy_key_exists")
    raise SystemExit(0)
body = json.dumps({{"title": "119.45.16.112", "key": pub, "read_only": True}}).encode()
req = urllib.request.Request(
    "https://api.github.com/repos/ghsemail/GeeGooAgent/keys",
    data=body,
    headers={{
        "Authorization": f"Bearer {{token}}",
        "Accept": "application/vnd.github+json",
        "Content-Type": "application/json",
    }},
    method="POST",
)
try:
    with urllib.request.urlopen(req, timeout=30) as r:
        print("deploy_key_added", r.status)
except urllib.error.HTTPError as e:
    print("deploy_key_error", e.code, e.read().decode()[:300])
PY
"""
    print("==> register deploy key")
    out, err, _ = run(client, register, timeout=120)
    print(out.strip() or err.strip())

    out, _, _ = run(client, "ssh -o BatchMode=yes -T git@github.com 2>&1 || true", timeout=30)
    print("==> github ssh:", out.strip()[:180])

    print("==> upload install.sh")
    sh_text = INSTALL_SH.read_text(encoding="utf-8").replace("\r\n", "\n")
    sftp = client.open_sftp()
    remote_sh = "/tmp/geegoo-install.sh"
    with sftp.open(remote_sh, "w") as f:
        f.write(sh_text)
    sftp.chmod(remote_sh, 0o755)
    sftp.close()

    prep = r"""
set -e
if [ -d "$HOME/.geegoo/geegoo-agent" ] && [ ! -d "$HOME/.geegoo/geegoo-agent/.git" ]; then
  rm -rf "$HOME/.geegoo/geegoo-agent"
  echo removed_tarball_install
fi
"""
    out, _, _ = run(client, prep, timeout=60)
    if out.strip():
        print(out.strip())

    print("==> bash install.sh (README)")
    install_cmd = f"export GEEGOO_SKIP_SETUP=1 DEBIAN_FRONTEND=noninteractive; bash {remote_sh}"
    out, err, code = run(client, install_cmd, timeout=900)
    print(out)
    if err.strip():
        print(err, file=sys.stderr)
    if code != 0:
        client.close()
        return code

    post = r"""
set -e
export PATH="$HOME/.geegoo/bin:$PATH"
export GEEGOO_HOME="$HOME/.geegoo"
export GEEGOO_CONFIG="$HOME/.geegoo/config.json"
echo '--- verify ---'
test -d ~/.geegoo/geegoo-agent/.git && echo git_clone_ok || echo no_git
~/.geegoo/bin/geegoo --help | head -4
echo '--- geegoo setup --skip-install ---'
~/.geegoo/bin/geegoo setup --config ~/.geegoo/config.json --skip-install 2>&1 || true
echo '--- geegoo doctor ---'
~/.geegoo/bin/geegoo doctor 2>&1 || true
"""
    print("==> setup + doctor")
    out, err, _ = run(client, post, timeout=300)
    print(out)
    if err.strip():
        print(err, file=sys.stderr)

    client.close()
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
