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
import json, os
from pathlib import Path

p = Path.home() / ".cursor/credentials/remote-deploy.json"
existing = {}
if p.is_file():
    try:
        existing = json.loads(p.read_text(encoding="utf-8"))
    except json.JSONDecodeError:
        existing = {}

ssh = dict(existing.get("ssh") or {})
if os.environ.get("GEEGOO_SSH_PASSWORD", "").strip():
    ssh["default"] = os.environ["GEEGOO_SSH_PASSWORD"].strip()
for k, v in os.environ.items():
    if k.startswith("GEEGOO_SSH_PASSWORD_") and not k.startswith("GEEGOO_SSH_PASSWORD_NEW") and v.strip():
        ssh[k[len("GEEGOO_SSH_PASSWORD_"):].replace("_", ".")] = v.strip()

cos = dict(existing.get("cos") or {})
sid = os.environ.get("TENCENT_COS_SECRET_ID", "").strip()
skey = os.environ.get("TENCENT_COS_SECRET_KEY", "").strip()
if sid and skey:
    cos = {"secret_id": sid, "secret_key": skey}

if ssh or cos:
    p.parent.mkdir(parents=True, exist_ok=True)
    payload = {"ssh": ssh, "cos": cos}
    if existing.get("_comment"):
        payload["_comment"] = existing["_comment"]
    p.write_text(json.dumps(payload, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
    print("merged credentials ->", p, f"({len(ssh)} ssh entries)")
elif not p.is_file():
    print("WARN: set GEEGOO_SSH_PASSWORD_* in Cursor Cloud env secrets")
PY

echo "==> [cloud] remote-deploy ready at ${SKILLS_DIR}/remote-deploy"
