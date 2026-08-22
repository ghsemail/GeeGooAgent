#!/usr/bin/env python3
"""Seed WeKnora admin, models, and default KB. Run on the Agent host."""
from __future__ import annotations

import json
import os
import sys
import time
import urllib.error
import urllib.request
from pathlib import Path

BASE = os.environ.get("WEKNORA_API", "http://127.0.0.1:3481/api/v1").rstrip("/")
ADMIN_FILE = Path("/home/ubuntu/apps/WeKnora/.geegoo-admin")
CONFIG_JSON = Path("/home/ubuntu/.geegoo/config.json")
AGENT_ENV = Path("/home/ubuntu/.geegoo/agent.env")


def mask(v: str) -> str:
    s = (v or "").strip()
    if not s:
        return ""
    if len(s) <= 8:
        return "*" * len(s)
    return s[:4] + "***" + s[-4:]


def load_env_file(path: Path) -> dict[str, str]:
    out: dict[str, str] = {}
    if not path.is_file():
        return out
    for raw in path.read_text(encoding="utf-8", errors="replace").splitlines():
        line = raw.strip()
        if not line or line.startswith("#"):
            continue
        if line.startswith("export "):
            line = line[7:].strip()
        if "=" not in line:
            continue
        k, v = line.split("=", 1)
        out[k.strip()] = v.strip().strip('"').strip("'")
    return out


def http_json(method: str, url: str, body: dict | None = None, headers: dict | None = None, timeout: int = 60):
    data = None if body is None else json.dumps(body).encode()
    req = urllib.request.Request(url, data=data, method=method)
    req.add_header("Content-Type", "application/json")
    for k, v in (headers or {}).items():
        req.add_header(k, v)
    try:
        with urllib.request.urlopen(req, timeout=timeout) as resp:
            raw = resp.read().decode()
            return resp.status, json.loads(raw) if raw else {}
    except urllib.error.HTTPError as e:
        raw = e.read().decode(errors="replace")
        try:
            parsed = json.loads(raw) if raw else {}
        except json.JSONDecodeError:
            parsed = {"raw": raw[:500]}
        return e.code, parsed


def dig(obj, *keys, default=None):
    cur = obj
    for k in keys:
        if not isinstance(cur, dict) or k not in cur:
            return default
        cur = cur[k]
    return cur


def wait_health(retries: int = 60) -> None:
    url = BASE.rsplit("/api/v1", 1)[0] + "/health"
    for i in range(retries):
        try:
            with urllib.request.urlopen(url, timeout=5) as resp:
                if resp.status == 200:
                    print("health ok")
                    return
        except Exception:
            pass
        print(f"waiting health {i+1}/{retries}")
        time.sleep(5)
    raise SystemExit("WeKnora /health not ready")


def catalog_models() -> tuple[dict, dict | None, dict]:
    cfg = {}
    if CONFIG_JSON.is_file():
        cfg = json.loads(CONFIG_JSON.read_text(encoding="utf-8"))
    env = load_env_file(AGENT_ENV)
    catalog = (
        cfg.get("signal_catalog_api_url")
        or cfg.get("signal_base_url")
        or env.get("GEEGOO_SIGNAL_CATALOG_API_URL")
        or "http://146.56.225.252:3210"
    ).rstrip("/")
    key = (
        cfg.get("signal_catalog_api_key")
        or env.get("GEEGOO_SIGNAL_CATALOG_API_KEY")
        or ""
    )
    headers = {"Authorization": f"Bearer {key}"} if key else {}
    code, runtime = http_json("POST", catalog + "/getModelRuntimeConfig", {}, headers)
    if code != 200 or runtime.get("code") not in (100, None):
        raise SystemExit(f"getModelRuntimeConfig failed: {code} {runtime}")
    data = runtime.get("data") or {}
    primary = data.get("configured_model") or {}
    fallbacks = data.get("fallback_models") or []
    fallback = fallbacks[0] if fallbacks else None
    if not primary.get("name") or not primary.get("base_url") or not primary.get("token"):
        # queryModel type=configured as fallback
        qcode, qdoc = http_json("POST", catalog + "/queryModel", {"type": "configured"}, headers)
        if qcode == 200 and isinstance(qdoc, dict):
            primary = qdoc
    if not primary.get("name") or not primary.get("token"):
        raise SystemExit("ops primary model missing name/token")
    emb_cfg = cfg.get("embedding") or {}
    embedding = {
        "name": emb_cfg.get("model") or "kinfra-text-embedding-4b",
        "base_url": (emb_cfg.get("base_url") or "https://tokenhub.tencentmaas.com/v1").rstrip("/"),
        "token": emb_cfg.get("token_key") or env.get("OPENAI_API_KEY") or env.get("GEEGOO_OPENAI_API_KEY") or "",
        "dimensions": int(emb_cfg.get("dimensions") or 2560),
        "provider": emb_cfg.get("provider") or "tencent-maas",
    }
    if not embedding["token"]:
        raise SystemExit("embedding token missing in config.json")
    print(
        "primary",
        primary.get("display_name") or primary.get("name"),
        "base",
        primary.get("base_url"),
        "token",
        mask(primary.get("token") or ""),
    )
    if fallback:
        print(
            "fallback",
            fallback.get("display_name") or fallback.get("name"),
            "base",
            fallback.get("base_url"),
            "token",
            mask(fallback.get("token") or ""),
        )
    print("embedding", embedding["name"], "base", embedding["base_url"], "dim", embedding["dimensions"])
    return primary, fallback, embedding


def auth_headers(token: str) -> dict:
    return {"Authorization": f"Bearer {token}"}


def register_or_login(username: str, email: str, password: str) -> str:
    code, body = http_json(
        "POST",
        BASE + "/auth/register",
        {"username": username, "email": email, "password": password},
    )
    token = body.get("token") or dig(body, "data", "token")
    if token:
        print("registered admin")
        return token
    code, body = http_json(
        "POST",
        BASE + "/auth/login",
        {"email": email, "password": password},
    )
    token = body.get("token") or dig(body, "data", "token")
    if not token:
        raise SystemExit(f"login failed: {code} keys={list(body)[:8]}")
    print("logged in existing admin")
    return token


def normalize_base_url(url: str) -> str:
    u = (url or "").strip().rstrip("/")
    if not u:
        return u
    lower = u.lower()
    if lower.endswith(("/v1", "/v2", "/v3", "/v4", "/v1beta", "/compatible-mode/v1")):
        return u
    return u + "/v1"


def source_for(provider: str, base_url: str = "", model_name: str = "") -> str:
    blob = " ".join([provider or "", base_url or "", model_name or ""]).lower()
    if "minimax" in blob or "minimaxi" in blob:
        return "minimax"
    if "deepseek" in blob:
        return "deepseek"
    if "tokenhub.tencentmaas" in blob or "kinfra" in blob:
        return "generic"
    return "generic"


def test_llm(token: str, model: dict) -> None:
    base = normalize_base_url(model["base_url"])
    src = source_for(model.get("provider") or "", base, model.get("name") or "")
    payload = {
        "source": src,
        "provider": src,
        "modelName": model["name"],
        "baseUrl": base,
        "apiKey": model["token"],
    }
    code, body = http_json("POST", BASE + "/initialization/remote/check", payload, auth_headers(token), timeout=90)
    available = dig(body, "data", "available", default=body.get("available"))
    msg = dig(body, "data", "message", default=body.get("message") or body)
    print("llm check", src, base, code, available, msg)
    if not available:
        for alt in ("minimax", "generic", "remote"):
            if alt == src:
                continue
            payload["source"] = alt
            payload["provider"] = alt
            code, body = http_json("POST", BASE + "/initialization/remote/check", payload, auth_headers(token), timeout=90)
            available = dig(body, "data", "available", default=body.get("available"))
            print("llm check retry", alt, code, available, body.get("message") or dig(body, "data", "message"))
            if available:
                model["_source"] = alt
                model["_base_url"] = base
                return
        raise SystemExit("primary LLM connectivity failed")
    model["_source"] = src
    model["_base_url"] = base


def test_embedding(token: str, emb: dict) -> None:
    payload = {
        "source": "generic",
        "provider": "generic",
        "modelName": emb["name"],
        "baseUrl": normalize_base_url(emb["base_url"]),
        "apiKey": emb["token"],
        "dimension": emb["dimensions"],
    }
    code, body = http_json("POST", BASE + "/initialization/embedding/test", payload, auth_headers(token), timeout=90)
    available = dig(body, "data", "available", default=body.get("available"))
    print("embedding check", code, available, dig(body, "data", "message") or body.get("message"))
    if not available:
        for alt in ("openai", "remote"):
            payload["source"] = alt
            payload["provider"] = alt
            code, body = http_json("POST", BASE + "/initialization/embedding/test", payload, auth_headers(token), timeout=90)
            available = dig(body, "data", "available", default=body.get("available"))
            print("embedding check retry", alt, code, available, dig(body, "data", "message") or body.get("message"))
            if available:
                emb["_source"] = alt
                emb["_base_url"] = payload["baseUrl"]
                return
        raise SystemExit("embedding connectivity failed")
    emb["_source"] = payload["source"]
    emb["_base_url"] = payload["baseUrl"]


def ensure_kb(token: str) -> str:
    code, body = http_json("GET", BASE + "/knowledge-bases", None, auth_headers(token))
    items = body.get("data") if isinstance(body.get("data"), list) else body if isinstance(body, list) else []
    if isinstance(body.get("data"), dict) and "items" in body["data"]:
        items = body["data"]["items"]
    for item in items or []:
        if isinstance(item, dict) and item.get("name") == "GeeGoo":
            kb_id = item.get("id") or item.get("knowledge_base_id")
            print("kb exists", kb_id)
            return str(kb_id)
    code, body = http_json(
        "POST",
        BASE + "/knowledge-bases",
        {"name": "GeeGoo", "description": "GeeGoo default knowledge base", "type": "document"},
        auth_headers(token),
    )
    kb_id = dig(body, "data", "id") or body.get("id")
    if not kb_id:
        raise SystemExit(f"create kb failed: {code} {body}")
    print("kb created", kb_id)
    return str(kb_id)


def initialize_kb(token: str, kb_id: str, primary: dict, emb: dict) -> None:
    payload = {
        "llm": {
            "source": primary.get("_source") or source_for(primary.get("provider") or "", primary.get("base_url") or "", primary.get("name") or ""),
            "modelName": primary["name"],
            "baseUrl": primary.get("_base_url") or normalize_base_url(primary["base_url"]),
            "apiKey": primary["token"],
        },
        "embedding": {
            "source": emb.get("_source") or "generic",
            "modelName": emb["name"],
            "baseUrl": emb.get("_base_url") or normalize_base_url(emb["base_url"]),
            "apiKey": emb["token"],
            "dimension": emb["dimensions"],
        },
        "rerank": {"enabled": False},
        "multimodal": {"enabled": False},
        "documentSplitting": {
            "chunkSize": 512,
            "chunkOverlap": 50,
            "separators": ["\n\n", "\n", "。"],
        },
        "nodeExtract": {"enabled": False},
        "questionGeneration": {"enabled": False},
    }
    code, body = http_json(
        "POST",
        f"{BASE}/initialization/initialize/{kb_id}",
        payload,
        auth_headers(token),
        timeout=120,
    )
    ok = body.get("success") is True or code in (200, 201)
    print("initialize", code, body.get("message") or body.get("msg") or ok)
    if not ok:
        payload["llm"]["source"] = "remote"
        payload["embedding"]["source"] = "remote"
        code, body = http_json(
            "POST",
            f"{BASE}/initialization/initialize/{kb_id}",
            payload,
            auth_headers(token),
            timeout=120,
        )
        print("initialize retry", code, body.get("message") or ok)
        if body.get("success") is not True and code not in (200, 201):
            raise SystemExit(f"initialize failed: {body}")


def register_fallback(token: str, fallback: dict | None) -> None:
    if not fallback or not fallback.get("name"):
        print("no fallback model")
        return
    code, body = http_json(
        "POST",
        BASE + "/models",
        {
            "name": fallback["name"],
            "display_name": fallback.get("display_name") or fallback["name"],
            "type": "chat",
            "source": source_for(fallback.get("provider") or "", fallback.get("base_url") or "", fallback.get("name") or ""),
            "parameters": {"base_url": normalize_base_url(fallback.get("base_url") or "")},
        },
        auth_headers(token),
    )
    model_id = dig(body, "data", "id") or body.get("id")
    print("fallback model", code, model_id)
    if model_id and fallback.get("token"):
        http_json(
            "PUT",
            f"{BASE}/models/{model_id}/credentials",
            {"api_key": fallback["token"]},
            auth_headers(token),
        )


def psql(sql: str) -> str:
    import subprocess

    cmd = [
        "docker",
        "exec",
        "WeKnora-postgres",
        "psql",
        "-U",
        "weknora",
        "-d",
        "WeKnora",
        "-t",
        "-A",
        "-c",
        sql,
    ]
    proc = subprocess.run(cmd, capture_output=True, text=True)
    out = (proc.stdout or "") + (proc.stderr or "")
    if proc.returncode != 0:
        raise SystemExit(f"psql failed: {out.strip()[:400]}")
    return proc.stdout.strip()


def ensure_builtin_models() -> None:
    out = psql("UPDATE models SET is_builtin=true WHERE tenant_id=0 AND deleted_at IS NULL AND is_builtin IS DISTINCT FROM true RETURNING id;")
    print("is_builtin updated", out or "none")


def ensure_bff_key(tenant_id: int = 10001) -> str:
    import base64
    import hashlib

    key_file = Path("/home/ubuntu/apps/WeKnora/.geegoo-bff-key")
    existing = psql(
        "SELECT api_key FROM tenant_api_keys WHERE name='geegoo-agent-bff' AND revoked_at IS NULL ORDER BY id DESC LIMIT 1;"
    )
    if existing:
        key_file.write_text(existing.strip() + "\n", encoding="utf-8")
        key_file.chmod(0o600)
        print("bff key exists", mask(existing.strip()))
        return existing.strip()
    token = "sk-" + base64.urlsafe_b64encode(os.urandom(32)).decode().rstrip("=")
    digest = hashlib.sha256(token.encode()).hexdigest()
    psql(
        "INSERT INTO tenant_api_keys "
        "(tenant_id, name, key_hash, api_key, full_access, knowledge_base_ids, capabilities, scope_type) "
        f"VALUES ({int(tenant_id)}, 'geegoo-agent-bff', '{digest}', '{token}', true, '[]'::jsonb, '[]'::jsonb, 'tenant');"
    )
    key_file.write_text(token + "\n", encoding="utf-8")
    key_file.chmod(0o600)
    print("bff key created", mask(token))
    return token


def patch_agent_config(kb_id: str) -> None:
    cfg: dict = {}
    if CONFIG_JSON.is_file():
        cfg = json.loads(CONFIG_JSON.read_text(encoding="utf-8"))
    wk = cfg.get("weknora") if isinstance(cfg.get("weknora"), dict) else {}
    wk.setdefault("api_url", "http://127.0.0.1:3481")
    wk.setdefault("web_url", "http://82.157.97.76:3480")
    wk["kb_id"] = kb_id
    wk.setdefault("api_key_file", "/home/ubuntu/apps/WeKnora/.geegoo-bff-key")
    cfg["weknora"] = wk
    CONFIG_JSON.write_text(json.dumps(cfg, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
    print("config weknora kb", kb_id)


def main() -> int:
    wait_health()
    creds = json.loads(ADMIN_FILE.read_text(encoding="utf-8"))
    token = register_or_login(creds["username"], creds["email"], creds["password"])
    primary, fallback, embedding = catalog_models()
    test_llm(token, primary)
    test_embedding(token, embedding)
    kb_id = ensure_kb(token)
    initialize_kb(token, kb_id, primary, embedding)
    register_fallback(token, fallback)
    ensure_builtin_models()
    ensure_bff_key()
    patch_agent_config(kb_id)
    print("bootstrap done kb", kb_id)
    return 0


if __name__ == "__main__":
    sys.exit(main())
