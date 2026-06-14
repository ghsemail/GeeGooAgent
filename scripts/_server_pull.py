#!/usr/bin/env python3
import paramiko
c = paramiko.SSHClient()
c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
c.connect("119.45.16.112", username="ubuntu", password="Ghs@2024", timeout=25)
cmd = r"""
export PATH="$HOME/.geegoo/bin:$PATH"
cd ~/.geegoo/geegoo-agent && git pull --ff-only origin main
source venv/bin/activate && pip install -e .[dev] -q
geegoo doctor 2>&1 | head -15
"""
_, o, e = c.exec_command(cmd, timeout=180)
print(o.read().decode())
if e.read().decode().strip():
    print(e.read().decode())
c.close()
