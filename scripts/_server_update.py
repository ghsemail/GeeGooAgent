#!/usr/bin/env python3
import paramiko

c = paramiko.SSHClient()
c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
c.connect("119.45.16.112", username="ubuntu", password="Ghs@2024", timeout=25)

def run(cmd, t=300):
    _, o, e = c.exec_command(cmd, timeout=t)
    code = o.channel.recv_exit_status()
    return o.read().decode("utf-8", "replace"), e.read().decode("utf-8", "replace"), code

cmds = [
    "export PATH=$HOME/.geegoo/bin:$PATH; geegoo update 2>&1",
    "export PATH=$HOME/.geegoo/bin:$PATH; geegoo doctor 2>&1",
    "git -C ~/.geegoo/geegoo-agent remote -v; git -C ~/.geegoo/geegoo-agent log -1 --oneline",
]
for cmd in cmds:
    print("===", cmd.split(";")[-1].strip()[:60])
    out, err, code = run(cmd)
    print(out.strip() or err.strip())
    print()
c.close()
