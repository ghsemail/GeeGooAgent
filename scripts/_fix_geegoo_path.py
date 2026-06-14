#!/usr/bin/env python3
import paramiko

c = paramiko.SSHClient()
c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
c.connect("119.45.16.112", username="ubuntu", password="Ghs@2024", timeout=25)

fix = r"""
set -e
BIN="$HOME/.geegoo/bin"
LINE='export PATH="$HOME/.geegoo/bin:$PATH"'
for rc in "$HOME/.bashrc" "$HOME/.profile"; do
  if [ -f "$rc" ]; then
    grep -qF '.geegoo/bin' "$rc" || echo "$LINE" >> "$rc"
  fi
done
echo "added PATH to bashrc/profile"
grep -n 'geegoo/bin' "$HOME/.bashrc" "$HOME/.profile" || true
"""
_, o, e = c.exec_command(fix, timeout=30)
print(o.read().decode())
print(e.read().decode())
_, o, _, = c.exec_command("bash -lc 'which geegoo'", timeout=15)
print("which geegoo:", o.read().decode().strip())
c.close()
