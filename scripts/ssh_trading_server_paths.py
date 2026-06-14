#!/usr/bin/env python3
import os
import paramiko

PASS = os.environ.get("SSH_PASS", "")
client = paramiko.SSHClient()
client.set_missing_host_key_policy(paramiko.AutoAddPolicy())
client.connect("43.134.94.87", username="ubuntu", password=PASS, timeout=25)
cmd = "curl -s http://127.0.0.1:7000/openapi.json | python3 -c \"import sys,json; p=json.load(sys.stdin)['paths']; print('\\n'.join(sorted(p.keys())[:50])); print('TOTAL',len(p))\""
_i, o, _e = client.exec_command(cmd, timeout=30)
print(o.read().decode())
client.close()
