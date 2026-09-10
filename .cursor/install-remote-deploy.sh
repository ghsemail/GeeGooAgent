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

if [[ ! -f "${CREDS_FILE}" ]]; then
  python3 - <<'PY'
import json, os
from pathlib import Path
p = Path.home() / ".cursor/credentials/remote-deploy.json"
ssh = {}
if os.environ.get("GEEGOO_SSH_PASSWORD", "").strip():
    ssh["default"] = os.environ["GEEGOO_SSH_PASSWORD"].strip()
for k, v in os.environ.items():
    if k.startswith("GEEGOO_SSH_PASSWORD_") and v.strip():
        ssh[k[len("GEEGOO_SSH_PASSWORD_"):].replace("_", ".")] = v.strip()
cos = {}
sid = os.environ.get("TENCENT_COS_SECRET_ID", "").strip()
skey = os.environ.get("TENCENT_COS_SECRET_KEY", "").strip()
if sid and skey:
    cos = {"secret_id": sid, "secret_key": skey}
if ssh or cos:
    p.write_text(json.dumps({"ssh": ssh, "cos": cos}, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
    print("wrote", p)
else:
    print("WARN: set GEEGOO_SSH_PASSWORD in Cursor Cloud env secrets")
PY
fi

echo "==> [cloud] remote-deploy ready at ${SKILLS_DIR}/remote-deploy"
