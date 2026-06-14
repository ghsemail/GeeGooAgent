#!/usr/bin/env python3
import paramiko

c = paramiko.SSHClient()
c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
c.connect("119.45.16.112", username="ubuntu", password="Ghs@2024", timeout=25)

def run(cmd, t=60):
    _, o, e = c.exec_command(cmd, timeout=t)
    return o.read().decode("utf-8", "replace"), e.read().decode("utf-8", "replace"), o.channel.recv_exit_status()

checks = [
    "ls -la ~/.geegoo/bin/ 2>/dev/null || echo NO_BIN",
    "test -x ~/.geegoo/bin/geegoo && echo GEegoo_BIN_OK || echo NO_GEegoo",
    "grep -n 'GEEGOO' ~/.bashrc ~/.profile 2>/dev/null || echo NO_BASHRC_LINES",
    "echo PATH=$PATH",
    "bash -lc 'which geegoo; geegoo --help | head -3'",
]
for cmd in checks:
    print("===", cmd[:70])
    out, err, _ = run(cmd)
    print(out.strip() or err.strip())
    print()

c.close()
