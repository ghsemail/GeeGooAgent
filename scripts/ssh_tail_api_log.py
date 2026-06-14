#!/usr/bin/env python3
import os
import paramiko

PASS = os.environ.get("SSH_PASS", "")
client = paramiko.SSHClient()
client.set_missing_host_key_policy(paramiko.AutoAddPolicy())
client.connect("118.195.135.97", username="ubuntu", password=PASS, timeout=25)
for cmd in [
    "grep -i 'getCurrentPrice\\|getPrice\\|Error\\|Traceback' /home/ubuntu/apps/TradingBot/api.out | tail -15",
    "grep -i searchCode /home/ubuntu/apps/TradingBot/mcpapi.out | tail -5",
]:
    print("---", cmd)
    _i, o, _e = client.exec_command(cmd, timeout=20)
    print(o.read().decode()[:2000])
client.close()
