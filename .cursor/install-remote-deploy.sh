#!/usr/bin/env bash
# Cloud Agent build: install remote-deploy skill from COS + wire credentials from env.
set -euo pipefail

COS_URL="${REMOTE_DEPLOY_COS_URL:-https://appsmith-1256042904.cos.ap-nanjing.myqcloud.com/skills/remote-deploy.zip}"
SKILLS_DIR="${HOME}/.cursor/skills"
CREDS_FILE="${HOME}/.cursor/credentials/remote-deploy.json"
TMP_ZIP="$(mktemp /tmp/remote-deploy.XXXXXX.zip)"

mkdir -p "${SKILLS_DIR}" "${HOME}/.cursor/credentials"
echo "==> [cloud] download remote-deploy skill"
curl -fsSL "${COS_URL}" -o "${TMP_ZIP}"
unzip -oq "${TMP_ZIP}" -d "${SKILLS_DIR}"
rm -f "${TMP_ZIP}"

python3 - <<'PY'
import json, os, re
from pathlib import Path

DEPLOY = Path.home() / ".cursor/skills/remote-deploy/deploy.json"
deploy = json.loads(DEPLOY.read_text(encoding="utf-8")) if DEPLOY.is_file() else {}
targets = deploy.get("targets") or {}

# deploy.json target 名 = skill 里的「服务器/target」ID（见 REFERENCE §1.2 / §2.1 / §3.1）
host_to_targets: dict[str, list[str]] = {}
for name, spec in targets.items():
    host = str((spec.get("ssh") or {}).get("host") or "")
    if host:
        host_to_targets.setdefault(host, []).append(name)

def is_ip_key(key: str) -> bool:
    return bool(re.fullmatch(r"\d{1,3}(\.\d{1,3}){3}", key or ""))


def fanout_same_host(ssh: dict[str, str]) -> dict[str, str]:
    """One target password on a host applies to all targets on that host."""
    out = dict(ssh)
    for _host, names in host_to_targets.items():
        pwd = ""
        for n in names:
            if out.get(n):
                pwd = out[n]
                break
        if pwd:
            for n in names:
                out.setdefault(n, pwd)
    return out


def strip_ip_keys(ssh: dict[str, str]) -> dict[str, str]:
    out = dict(ssh)
    for key in list(out):
        if is_ip_key(key):
            pwd = out.pop(key)
            for t in host_to_targets.get(key, ()):
                out.setdefault(t, pwd)
    return out

p = Path.home() / ".cursor/credentials/remote-deploy.json"
existing = {}
if p.is_file():
    try:
        existing = json.loads(p.read_text(encoding="utf-8"))
    except json.JSONDecodeError:
        existing = {}
ssh = dict(existing.get("ssh") or {}) if isinstance(existing.get("ssh"), dict) else {}
cos = dict(existing.get("cos") or {}) if isinstance(existing.get("cos"), dict) else {}

if os.environ.get("GEEGOO_SSH_PASSWORD", "").strip():
    ssh["default"] = os.environ["GEEGOO_SSH_PASSWORD"].strip()

for k, v in os.environ.items():
    if not k.startswith("GEEGOO_SSH_PASSWORD_") or not v.strip():
        continue
    suffix = k[len("GEEGOO_SSH_PASSWORD_") :].strip()
    if re.fullmatch(r"[\d_]+", suffix):
        # Legacy IP env → all deploy.json targets on that host (no IP keys in vault).
        ip = suffix.replace("_", ".")
        for t in host_to_targets.get(ip, ()):
            ssh.setdefault(t, v.strip())
        continue
    # GEEGOO_SSH_PASSWORD_GEEGOO_TRADINGBOT → target geegoo-tradingbot
    target_key = suffix.lower().replace("_", "-")
    if target_key not in targets and target_key != "default":
        print(f"WARN: unknown deploy target in env: {target_key}")
    ssh[target_key] = v.strip()

ssh = strip_ip_keys(ssh)
ssh = fanout_same_host(ssh)

sid = os.environ.get("TENCENT_COS_SECRET_ID", "").strip()
skey = os.environ.get("TENCENT_COS_SECRET_KEY", "").strip()
if sid and skey:
    cos = {"secret_id": sid, "secret_key": skey}

if ssh or cos:
    p.parent.mkdir(parents=True, exist_ok=True)
    p.write_text(
        json.dumps({"ssh": ssh, "cos": cos}, ensure_ascii=False, indent=2) + "\n",
        encoding="utf-8",
    )
    print("wrote", p, f"(ssh keys={list(ssh.keys())})")
elif not p.is_file():
    print("WARN: set GEEGOO_SSH_PASSWORD_<TARGET> in Cursor Cloud env secrets")
PY

echo "==> [cloud] remote-deploy ready at ${SKILLS_DIR}/remote-deploy"
