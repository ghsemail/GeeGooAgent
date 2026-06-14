#!/usr/bin/env python3
import paramiko
c = paramiko.SSHClient()
c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
c.connect("119.45.16.112", username="ubuntu", password="Ghs@2024", timeout=25)
cmds = [
    "test -f ~/.ssh/id_ed25519.pub && echo 'server_ssh_pubkey: yes' || echo 'server_ssh_pubkey: no'",
    "test -f ~/.geegoo/github_token && echo 'server_github_token: yes (len='$(wc -c < ~/.geegoo/github_token | tr -d ' ')')' || echo 'server_github_token: no'",
    "ssh -o BatchMode=yes -T git@github.com 2>&1 | head -1 || true",
]
for cmd in cmds:
    _, o, e = c.exec_command(cmd, timeout=20)
    print(o.read().decode().strip() or e.read().decode().strip())
c.close()
