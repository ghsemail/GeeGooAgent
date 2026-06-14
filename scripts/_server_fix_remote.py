#!/usr/bin/env python3
import paramiko

c = paramiko.SSHClient()
c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
c.connect("119.45.16.112", username="ubuntu", password="Ghs@2024", timeout=25)

def run(cmd, t=300):
    _, o, e = c.exec_command(cmd, timeout=t)
    code = o.channel.recv_exit_status()
    return o.read().decode("utf-8", "replace"), e.read().decode("utf-8", "replace"), code

fix = r"""
set -e
cd ~/.geegoo/geegoo-agent
git remote set-url origin git@github.com:ghsemail/GeeGooAgent.git
git fetch origin main
git reset --hard origin/main
git log -1 --oneline
"""
print("==> fix remote + pull")
out, err, code = run(fix, 120)
print(out.strip())
if code:
    print(err.strip())

out, _, _ = run("export PATH=$HOME/.geegoo/bin:$PATH; geegoo doctor 2>&1 | head -12", 60)
print("\n==> doctor\n", out.strip())
c.close()
