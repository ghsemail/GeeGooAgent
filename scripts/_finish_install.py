#!/usr/bin/env python3
import paramiko
from pathlib import Path

HOST, USER, PASS = "119.45.16.112", "ubuntu", "Ghs@2024"
INSTALL_SH = Path(__file__).resolve().parents[1] / "scripts" / "install.sh"

def run(c, cmd, timeout=600):
    _, o, e = c.exec_command(cmd, timeout=timeout)
    code = o.channel.recv_exit_status()
    return o.read().decode("utf-8", "replace"), e.read().decode("utf-8", "replace"), code

c = paramiko.SSHClient()
c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
c.connect(HOST, username=USER, password=PASS, timeout=25)

# status
for cmd in [
    "test -d ~/.geegoo/geegoo-agent/.git && echo GIT_OK || echo NO_GIT",
    "test -x ~/.geegoo/bin/geegoo && ~/.geegoo/bin/geegoo --version 2>/dev/null || ~/.geegoo/bin/geegoo --help | head -1",
    "test -d ~/.gigo && echo GIGO_EXISTS || echo GIGO_GONE",
]:
    out, _, _ = run(c, cmd, 30)
    print(out.strip())

# install if needed
out, _, _ = run(c, "test -d ~/.geegoo/geegoo-agent/.git && echo skip || echo install", 10)
if "install" in out:
    sh = INSTALL_SH.read_text(encoding="utf-8").replace("\r\n", "\n")
    sftp = c.open_sftp()
    sftp.open("/tmp/geegoo-install.sh", "w").write(sh)
    sftp.chmod("/tmp/geegoo-install.sh", 0o755)
    sftp.close()
    if run(c, "test -d ~/.geegoo/geegoo-agent && [ ! -d ~/.geegoo/geegoo-agent/.git ]", 10)[2] == 0:
        run(c, "rm -rf ~/.geegoo/geegoo-agent", 60)
    out, err, code = run(c, "export GEEGOO_SKIP_SETUP=1; bash /tmp/geegoo-install.sh", 900)
    print("INSTALL:", out[-800:] if len(out) > 800 else out)
    if code: print("ERR:", err[:300])

out, _, _ = run(c, "export PATH=$HOME/.geegoo/bin:$PATH; geegoo doctor 2>&1 | head -20", 120)
print("DOCTOR:", out.strip())
c.close()
