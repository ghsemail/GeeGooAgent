#!/usr/bin/env python3
"""Verify or rotate SSH passwords for all unique hosts in remote-deploy deploy.json.

Uses ~/.cursor/skills/remote-deploy/deploy.json + credentials.py (Secrets / credential vault).
Does not commit passwords to the GeeGooAgent git repo.

Examples:
  python scripts/ops/deploy/rotate_ssh_passwords.py --verify
  GEEGOO_SSH_NEW_PASSWORD='...' python scripts/ops/deploy/rotate_ssh_passwords.py --apply
  python scripts/ops/deploy/rotate_ssh_passwords.py --sync-deploy-json
"""
from __future__ import annotations

import argparse
import json
import os
import sys
from pathlib import Path

import paramiko

SKILL_DIR = Path.home() / ".cursor" / "skills" / "remote-deploy"
SCRIPTS_DIR = SKILL_DIR / "scripts"
DEPLOY_CFG = SKILL_DIR / "deploy.json"

UNIQUE_HOST_TARGETS = (
    "geegoo-tradingbot",
    "geegoo-tradingsignal",
    "geegoo-tradingserver",
    "geegoo-tradingdata",
    "geegoo-data-cn",
)


def _import_credentials():
    sys.path.insert(0, str(SCRIPTS_DIR))
    from credentials import resolve_ssh_cfg  # noqa: WPS433

    return resolve_ssh_cfg


def load_deploy() -> dict:
    if not DEPLOY_CFG.is_file():
        raise SystemExit(f"missing {DEPLOY_CFG}; run install-remote-deploy.sh first")
    return json.loads(DEPLOY_CFG.read_text(encoding="utf-8"))


def unique_hosts(cfg: dict) -> list[tuple[str, dict, str]]:
    """Return (host, resolved_ssh, representative_target) in stable order."""
    resolve_ssh_cfg = _import_credentials()
    seen: set[str] = set()
    out: list[tuple[str, dict, str]] = []
    for name in UNIQUE_HOST_TARGETS:
        target = cfg.get("targets", {}).get(name)
        if not target:
            continue
        raw = dict(target.get("ssh") or {})
        host = str(raw.get("host") or "")
        if not host or host in seen:
            continue
        seen.add(host)
        resolved = resolve_ssh_cfg(raw, name)
        out.append((host, resolved, name))
    return out


def connect(ssh_cfg: dict) -> paramiko.SSHClient:
    if not ssh_cfg.get("password") and not ssh_cfg.get("key_file"):
        raise RuntimeError(
            f"no SSH password for {ssh_cfg.get('host')}; "
            "set GEEGOO_SSH_PASSWORD_* in Cursor Secrets or credential vault"
        )
    client = paramiko.SSHClient()
    client.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    client.connect(
        ssh_cfg["host"],
        port=int(ssh_cfg.get("port") or 22),
        username=ssh_cfg["user"],
        password=ssh_cfg.get("password"),
        key_filename=ssh_cfg.get("key_file"),
        timeout=30,
    )
    return client


def run_chpasswd(client: paramiko.SSHClient, user: str, new_password: str) -> tuple[int, str]:
    cmd = "chpasswd" if user == "root" else "sudo -n chpasswd"
    stdin, stdout, stderr = client.exec_command(cmd, timeout=60)
    stdin.write(f"{user}:{new_password}\n")
    stdin.channel.shutdown_write()
    err = (stderr.read() + stdout.read()).decode("utf-8", "replace").strip()
    code = stdout.channel.recv_exit_status()
    return code, err


def resolve_new_password(host: str, target_name: str) -> str:
    host_key = f"GEEGOO_SSH_NEW_PASSWORD_{host.replace('.', '_')}"
    for key in (host_key, f"GEEGOO_SSH_NEW_PASSWORD_{target_name.upper().replace('-', '_')}"):
        if os.environ.get(key, "").strip():
            return os.environ[key].strip()
    if os.environ.get("GEEGOO_SSH_NEW_PASSWORD", "").strip():
        return os.environ["GEEGOO_SSH_NEW_PASSWORD"].strip()
    raise RuntimeError(
        f"set {host_key} or GEEGOO_SSH_NEW_PASSWORD for rotation on {host}"
    )


def cmd_verify(cfg: dict) -> int:
    ok = 0
    hosts = unique_hosts(cfg)
    for host, ssh_cfg, name in hosts:
        label = f"{host} ({ssh_cfg.get('user')}, via {name})"
        try:
            client = connect(ssh_cfg)
            _, stdout, _ = client.exec_command("hostname", timeout=20)
            hn = stdout.read().decode().strip()
            client.close()
            print(f"OK  {label} -> {hn}")
            ok += 1
        except Exception as exc:
            print(f"FAIL {label} -> {type(exc).__name__}: {exc}")
    total = len(hosts)
    print(f"verified {ok}/{total}")
    return 0 if ok == total else 1


def cmd_apply(cfg: dict) -> int:
    failures = 0
    for host, ssh_cfg, name in unique_hosts(cfg):
        user = str(ssh_cfg.get("user") or "")
        label = f"{host} ({user})"
        try:
            new_pwd = resolve_new_password(host, name)
            client = connect(ssh_cfg)
            code, err = run_chpasswd(client, user, new_pwd)
            client.close()
            if code != 0:
                print(f"FAIL {label} chpasswd exit {code}: {err}")
                failures += 1
                continue
            print(f"OK  {label} password rotated")
        except Exception as exc:
            print(f"FAIL {label} -> {type(exc).__name__}: {exc}")
            failures += 1
    return 1 if failures else 0


def cmd_sync_deploy_json(cfg: dict) -> int:
    resolve_ssh_cfg = _import_credentials()
    updated = 0
    for _name, target in cfg.get("targets", {}).items():
        ssh = target.get("ssh")
        if not isinstance(ssh, dict):
            continue
        filled = resolve_ssh_cfg(dict(ssh), _name)
        pwd = filled.get("password")
        if pwd:
            ssh["password"] = pwd
            updated += 1
    DEPLOY_CFG.write_text(
        json.dumps(cfg, ensure_ascii=False, indent=2) + "\n",
        encoding="utf-8",
    )
    print(f"wrote {DEPLOY_CFG} ({updated} targets with inline password)")
    print("optional: python ~/.cursor/skills/remote-deploy/scripts/upload_skill_to_cos.py")
    return 0


def main() -> int:
    parser = argparse.ArgumentParser(description="remote-deploy SSH verify / rotate")
    group = parser.add_mutually_exclusive_group(required=True)
    group.add_argument("--verify", action="store_true", help="test SSH login on 5 hosts")
    group.add_argument("--apply", action="store_true", help="chpasswd using GEEGOO_SSH_NEW_PASSWORD*")
    group.add_argument(
        "--sync-deploy-json",
        action="store_true",
        help="copy resolved passwords into deploy.json (for COS repack)",
    )
    args = parser.parse_args()
    cfg = load_deploy()
    if args.verify:
        return cmd_verify(cfg)
    if args.apply:
        return cmd_apply(cfg)
    return cmd_sync_deploy_json(cfg)


if __name__ == "__main__":
    raise SystemExit(main())
