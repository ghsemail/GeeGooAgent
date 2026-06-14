#!/usr/bin/env python3
import paramiko

HOST, USER, PASS = "119.45.16.112", "ubuntu", "Ghs@2024"
cmds = [
    "ls -la ~/.geegoo 2>/dev/null; ls -la ~/.geegoo/geegoo-agent/.git 2>/dev/null | head -3",
    "ls -la ~/.ssh 2>/dev/null; cat ~/.ssh/id_*.pub 2>/dev/null || echo no_ssh_pubkey",
    "test -x ~/.geegoo/bin/geegoo && ~/.geegoo/bin/geegoo --help | head -3 || echo no_geegoo",
    "curl -fsSL -I https://raw.githubusercontent.com/ghsemail/GeeGooAgent/main/scripts/install.sh | head -5",
    "curl -fsSL https://raw.githubusercontent.com/ghsemail/GeeGooAgent/main/scripts/install.sh | grep GEEGOO_REPO",
]
c = paramiko.SSHClient()
c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
c.connect(HOST, username=USER, password=PASS, timeout=25)
for cmd in cmds:
    print("===", cmd[:80])
    _, o, e = c.exec_command(cmd, timeout=60)
    print(o.read().decode("utf-8", "replace").strip() or e.read().decode("utf-8", "replace").strip())
c.close()
