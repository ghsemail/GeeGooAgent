#!/usr/bin/env python3
"""Find ghsemail user_id from GeeGooBot Mongo on bot host."""
from __future__ import annotations

import json
from pathlib import Path

import paramiko

DEPLOY = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")
cfg = json.loads(DEPLOY.read_text(encoding="utf-8"))
ssh = cfg["targets"]["geegoo-bot"]["ssh"]
c = paramiko.SSHClient()
c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
c.connect(ssh["host"], username=ssh["user"], password=ssh["password"], timeout=60)
cmd = r'''python3 - <<'PY'
from pymongo import MongoClient
import os, re
# load mongo uri from .env
uri = None
dbn = "QT_DB"
for line in open("/home/ubuntu/apps/GeeGooBot/.env"):
    line=line.strip()
    if line.startswith("GEEGOO_BOT_MONGO_URI="):
        uri = line.split("=",1)[1]
    if line.startswith("GEEGOO_BOT_MONGO_DB="):
        dbn = line.split("=",1)[1]
if not uri:
    uri = "mongodb://127.0.0.1:27017"
client = MongoClient(uri, serverSelectionTimeoutMS=5000)
db = client[dbn]
for coll in ["user", "users"]:
    if coll not in db.list_collection_names():
        continue
    for doc in db[coll].find({"$or": [{"username": "ghsemail"}, {"name": "ghsemail"}, {"email": re.compile("ghsemail", re.I)}]}).limit(3):
        print("coll", coll, "id", str(doc.get("_id")), "username", doc.get("username") or doc.get("name"), "keys", list(doc.keys())[:12])
PY'''
_, o, e = c.exec_command(cmd, timeout=60)
print((o.read() + e.read()).decode("utf-8", errors="replace"))
c.close()
