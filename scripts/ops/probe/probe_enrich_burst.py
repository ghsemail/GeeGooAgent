#!/usr/bin/env python3
"""Simulate news-worker burst of enrichStockNews calls."""
import json
import time
import urllib.request
from concurrent.futures import ThreadPoolExecutor, as_completed
from pathlib import Path

import paramiko

DEPLOY = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")


def get_key() -> str:
    cfg = json.loads(DEPLOY.read_text(encoding="utf-8"))
    ssh = cfg["targets"]["geegoo-signal"]["ssh"]
    client = paramiko.SSHClient()
    client.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    client.connect(
        hostname=ssh["host"],
        username=ssh["user"],
        password=ssh.get("password"),
        timeout=60,
    )
    _, stdout, _ = client.exec_command(
        "grep '^GEEGOO_SIGNAL_ANALYZE_API_KEY=' /root/apps/GeeGooSignal/.env | cut -d= -f2-"
    )
    key = stdout.read().decode().strip()
    client.close()
    return key


def enrich(key: str, title: str) -> tuple[str, bool, int, str]:
    url = "http://146.56.225.252:3230/enrichStockNews"
    body = json.dumps({"title": title, "snippet": ""}).encode()
    req = urllib.request.Request(
        url,
        data=body,
        headers={
            "Content-Type": "application/json",
            "Authorization": "Bearer " + key,
        },
    )
    t0 = time.time()
    resp = urllib.request.urlopen(req, timeout=180)
    data = json.loads(resp.read())
    elapsed = time.time() - t0
    return (
        title[:50],
        bool(data.get("used_fallback")),
        len(data.get("title_cn") or ""),
        f"{elapsed:.1f}s",
    )


def main() -> int:
    key = get_key()
    titles = [
        "How to Deploy $1000 Across Tesla and Ford for Growth and Optionality",
        "Elon Musk's Boring Company Is Raising Money at a $20 Billion Valuation.",
        "Elon Musk Owns 20% of Tesla, a Stake Worth Roughly $200 Billion.",
        "Tesla UK sales drop as drivers opt for Chinese rivals",
        "Prediction for Tesla Stock in 3 Years: The Bear Case",
        "Tencent (SEHK:700) Stock May Be Undervalued Despite Fresh AI Spending Questions",
        "Tencent Brings Together AI and Games to Help Preserve Cultural Heritage",
        "Starlink Unbelievable Juggernaut Cash Machine Says VC David Friedberg",
    ] * 2  # 16 calls like a busy refresh

    print("=== sequential burst ===")
    fb = 0
    for title in titles:
        try:
            short, fallback, cn_len, elapsed = enrich(key, title)
            if fallback or cn_len == 0:
                fb += 1
                print("FAIL", short, "fallback", fallback, "cn_len", cn_len, elapsed)
        except Exception as exc:
            fb += 1
            print("ERR", title[:50], exc)
    print("sequential failures", fb, "/", len(titles))

    print("\n=== parallel burst (workers=4) ===")
    fb2 = 0
    with ThreadPoolExecutor(max_workers=4) as pool:
        futs = [pool.submit(enrich, key, t) for t in titles]
        for fut in as_completed(futs):
            try:
                short, fallback, cn_len, elapsed = fut.result()
                if fallback or cn_len == 0:
                    fb2 += 1
                    print("FAIL", short, "fallback", fallback, "cn_len", cn_len, elapsed)
            except Exception as exc:
                fb2 += 1
                print("ERR", exc)
    print("parallel failures", fb2, "/", len(titles))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
