#!/usr/bin/env python3
"""Migrate GeeGooAgent user_llm_settings/*.json into QT_DB.user.llm_settings.

Reads JSON files from the agent workspace (default: ~/.geegoo/geegoo-agent/user_llm_settings)
and upserts llm_settings on matching user documents.

Environment:
  GEEGOO_BOT_MONGO_URI  (required)
  GEEGOO_BOT_MONGO_DB   (default QT_DB)
  AGENT_SETTINGS_DIR    (optional override for JSON source directory)

Usage:
  python scripts/migrate_user_llm_settings_to_mongo.py --dry-run
  python scripts/migrate_user_llm_settings_to_mongo.py
"""
from __future__ import annotations

import argparse
import json
import os
import re
from datetime import datetime, timezone
from pathlib import Path

from bson import ObjectId
from pymongo import MongoClient

SAFE_ID = re.compile(r"[^a-zA-Z0-9._-]+")


def settings_dir() -> Path:
    override = os.environ.get("AGENT_SETTINGS_DIR", "").strip()
    if override:
        return Path(override)
    home = Path.home()
    return home / ".geegoo" / "geegoo-agent" / "user_llm_settings"


def filename_to_user_id(name: str) -> str | None:
    stem = Path(name).stem
    if not stem or stem == "anonymous":
        return None
    # Files use safeUserID: non-alnum → underscore; cannot recover original slashes.
    return stem


def load_json(path: Path) -> dict | None:
    try:
        return json.loads(path.read_text(encoding="utf-8"))
    except (OSError, json.JSONDecodeError) as exc:
        print(f"skip {path}: {exc}")
        return None


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--dry-run", action="store_true")
    args = parser.parse_args()

    uri = os.environ.get("GEEGOO_BOT_MONGO_URI", "").strip()
    if not uri:
        print("GEEGOO_BOT_MONGO_URI is required")
        return 1
    db_name = os.environ.get("GEEGOO_BOT_MONGO_DB", "QT_DB").strip() or "QT_DB"
    src = settings_dir()
    if not src.is_dir():
        print(f"no settings dir: {src}")
        return 0

    client = MongoClient(uri, serverSelectionTimeoutMS=8000)
    db = client[db_name]
    coll = db["user"]

    migrated = skipped = failed = 0
    for path in sorted(src.glob("*.json")):
        user_id = filename_to_user_id(path.name)
        if not user_id:
            skipped += 1
            continue
        try:
            oid = ObjectId(user_id)
        except Exception:
            print(f"skip {path.name}: not a valid ObjectId user id")
            skipped += 1
            continue
        doc = load_json(path)
        if not doc:
            failed += 1
            continue
        doc.setdefault("updated_at", datetime.now(timezone.utc).isoformat())
        existing = coll.find_one({"_id": oid}, {"llm_settings": 1})
        if not existing:
            print(f"skip {path.name}: user {user_id} not in {db_name}.user")
            skipped += 1
            continue
        if existing.get("llm_settings"):
            print(f"skip {path.name}: user {user_id} already has llm_settings")
            skipped += 1
            continue
        if args.dry_run:
            print(f"would migrate {path.name} -> user {user_id}: keys={list(doc.keys())}")
            migrated += 1
            continue
        res = coll.update_one({"_id": oid}, {"$set": {"llm_settings": doc}})
        if res.modified_count:
            print(f"migrated {path.name} -> user {user_id}")
            migrated += 1
        else:
            print(f"no change {path.name}")
            skipped += 1

    print(f"done: migrated={migrated} skipped={skipped} failed={failed}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
