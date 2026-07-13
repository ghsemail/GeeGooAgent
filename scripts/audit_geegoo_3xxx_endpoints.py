#!/usr/bin/env python3
"""Verify GeeGooAgent outbound URLs are GeeGoo 3xxx stack only (no Trading 5xxx/7xxx)."""
from __future__ import annotations

import json
import re
from pathlib import Path

import paramiko

DEPLOY = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")

GEEGOO_PORTS = {
    3100: "GeeGooBot app-api",
    3110: "GeeGooBot agent-api",
    3120: "GeeGooBot mcp-api",
    3140: "GeeGooBot service-api",
    3200: "GeeGooSignal signal-api",
    3210: "GeeGooSignal catalog-api",
    3230: "GeeGooSignal analyze-api",
    3240: "GeeGooSignal decision-api",
    3300: "GeeGooData data-api",
    3400: "GeeGooAgent agent-runtime",
}

LEGACY_PORTS = {
    5500: "TradingBot service-api (legacy)",
    5600: "TradingBot app-api (legacy)",
    5700: "TradingBot mcp-api (legacy)",
    5800: "TradingSignal (legacy)",
    5900: "TradingSignal reports (legacy)",
    6100: "TradingData (legacy)",
    6200: "TradingSignal analyze (legacy)",
    7000: "TradingServer (legacy)",
}


def ssh_run(ssh_cfg: dict, cmd: str, timeout: int = 90) -> tuple[int, str, str]:
    client = paramiko.SSHClient()
    client.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    client.connect(
        hostname=ssh_cfg["host"],
        port=int(ssh_cfg.get("port", 22)),
        username=ssh_cfg["user"],
        password=ssh_cfg.get("password"),
        timeout=30,
    )
    _, stdout, stderr = client.exec_command(cmd, timeout=timeout)
    out = stdout.read().decode("utf-8", errors="replace")
    err = stderr.read().decode("utf-8", errors="replace")
    code = stdout.channel.recv_exit_status()
    client.close()
    return code, out, err


def port_of(url: str) -> int | None:
    m = re.search(r":(\d+)(?:/|$)", url or "")
    return int(m.group(1)) if m else None


def classify_url(label: str, url: str) -> dict:
    p = port_of(url)
    if not url:
        return {"label": label, "url": url, "status": "MISSING", "port": None}
    if p in GEEGOO_PORTS:
        return {"label": label, "url": url, "status": "OK", "port": p, "service": GEEGOO_PORTS[p]}
    if p in LEGACY_PORTS:
        return {"label": label, "url": url, "status": "LEGACY", "port": p, "service": LEGACY_PORTS[p]}
    if p and 3000 <= p < 4000:
        return {"label": label, "url": url, "status": "OK?", "port": p, "service": "GeeGoo-range (unlisted)"}
    return {"label": label, "url": url, "status": "OTHER", "port": p, "service": "non-3xxx or external"}


def main() -> int:
    cfg = json.loads(DEPLOY.read_text(encoding="utf-8"))
    agent = cfg["targets"]["geegoo-agent"]["ssh"]
    bot = cfg["targets"]["geegoo-bot"]["ssh"]

    py = r'''
import json, os
home = os.path.expanduser("~/.geegoo")
cfg = json.load(open(home + "/config.json", encoding="utf-8"))
env = {}
for line in open(home + "/agent.env", encoding="utf-8"):
    line = line.strip()
    if line and not line.startswith("#") and "=" in line:
        k, v = line.split("=", 1)
        env[k.strip()] = v.strip().strip('"').strip("'")
print(json.dumps({"config": cfg, "env": env}, ensure_ascii=False))
'''
    import base64

    b64 = base64.b64encode(py.encode()).decode()
    _, out, _ = ssh_run(agent, f"python3 -c \"import base64; exec(base64.b64decode('{b64}').decode())\"")
    payload = json.loads(out)
    c = payload["config"]
    env = payload["env"]

    rows = []
    for label, key in [
        ("MCP (geegoo_url)", c.get("geegoo_url") or c.get("base_url")),
        ("Signal catalog", c.get("signal_base_url")),
        ("Signal signal-api", c.get("signal_api_url")),
        ("Signal analyze-api", c.get("signal_analyze_api_url")),
        ("GeeGooData", c.get("data_base_url")),
        ("LLM base_url", (c.get("llm") or {}).get("base_url")),
    ]:
        rows.append(classify_url(label, key or ""))

    for label, key in [
        ("env GEEGOO_BOT_MCP_URL", env.get("GEEGOO_BOT_MCP_URL")),
        ("env GEEGOO_SIGNAL_CATALOG_API_URL", env.get("GEEGOO_SIGNAL_CATALOG_API_URL")),
        ("env GEEGOO_SIGNAL_ANALYZE_API_URL", env.get("GEEGOO_SIGNAL_ANALYZE_API_URL")),
        ("env GEEGOO_DATA_HTTP_URL", env.get("GEEGOO_DATA_HTTP_URL")),
    ]:
        if key:
            rows.append(classify_url(label, key))

    _, bot_env, _ = ssh_run(bot, "grep -hE 'GEEGOO_DATA|GEEGOO_SIGNAL|PORT' /home/ubuntu/apps/GeeGooBot/.env 2>/dev/null | head -20")
    for line in bot_env.splitlines():
        if "URL" in line and "http" in line:
            k, _, v = line.partition("=")
            rows.append(classify_url(f"Bot .env {k.strip()}", v.strip()))

    print("=" * 72)
    print("GeeGooAgent 出站 URL 审计（线上 config + env）")
    print("=" * 72)
    legacy = []
    ok = []
    for r in rows:
        port_s = str(r["port"]) if r["port"] else "-"
        svc = r.get("service", "")
        print(f"  [{r['status']:6}] {r['label']:32} :{port_s:4}  {r['url']}")
        if svc:
            print(f"         -> {svc}")
        if r["status"] == "LEGACY":
            legacy.append(r)
        elif r["status"] == "OK":
            ok.append(r)

    print("\n" + "=" * 72)
    print("geegoo doctor")
    print("=" * 72)
    _, doc, _ = ssh_run(agent, "export PATH=$HOME/.geegoo/bin:$PATH; geegoo doctor 2>&1")
    print(doc.strip())

    print("\n" + "=" * 72)
    print("数据链路抽样（Bot mcp-api -> GeeGooData）")
    print("=" * 72)
    _, data_health, _ = ssh_run(bot, "curl -sf -m 5 http://47.80.14.120:3300/health && echo")
    print("  GeeGooData :3300/health ->", data_health.strip() or "FAIL")

    print("\n" + "=" * 72)
    print("SUMMARY")
    print("=" * 72)
    if legacy:
        print(f"  LEGACY 端口命中: {len(legacy)} — 需迁移到 3xxx")
        for r in legacy:
            print(f"    - {r['label']}: {r['url']}")
    else:
        print("  配置中未发现 Trading 5xxx/7xxx 端口")
    print(f"  GeeGoo 3xxx 端点: {len(ok)} 项已对齐")
    print("  非 GeeGoo HTTP: LLM 供应商（OpenAI 兼容）为外部，属预期")

    return 1 if legacy else 0


if __name__ == "__main__":
    raise SystemExit(main())
