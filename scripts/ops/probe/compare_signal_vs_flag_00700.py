#!/usr/bin/env python3
"""Prove signal vs flag algorithms differ on 00700.HK full series via Go indexengine."""
from __future__ import annotations

import json
from pathlib import Path

import paramiko

DEPLOY = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")

# Catalog type=signal entries + EMA/HeikinAshi for completeness (user said OK if not in catalog)
CHECKS = [
    ("RSIThrehold", {"period": 25, "threholdBuy": 30, "threholdSell": 70}),
    ("RSICross", {"fastPeriod": 10, "slowPeriod": 100}),
    ("SAR", {"acceleration": 0.02, "maximum": 0.2}),
    ("MACD", {"fastPeriod": 12, "slowPeriod": 26, "signalPeriod": 9}),
    ("EMA", {"fastPeriod": 25, "mediumPeriod": 50, "slowPeriod": 120}),
    ("ChandelierExit", {"period": 22, "atrMultiplier": 3}),
    ("BBAND", {"period": 20, "matype": 2}),
    ("VWAP", {}),
    ("HeikinAshi", {}),
    ("KDJ", {"period": 9, "p1": 3, "p2": 3, "threholdBuy": 30, "threholdSell": 70}),
]

GO_SRC = r'''
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/ghsemail/GeeGooSignal/internal/client/geegoodata"
	"github.com/ghsemail/GeeGooSignal/internal/marketdata"
	"github.com/ghsemail/GeeGooSignal/internal/signal/indexengine"
)

type check struct {
	Index string         `json:"index"`
	Param map[string]any `json:"param"`
}

func main() {
	var checks []check
	_ = json.Unmarshal([]byte(os.Args[1]), &checks)
	dataURL := os.Getenv("GEEGOO_DATA_HTTP_URL")
	token := os.Getenv("GEEGOO_DATA_SERVICE_TOKEN")
	p := &marketdata.Provider{
		Remote:       geegoodata.New(dataURL, token),
		ProviderMode: "remote",
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	series, err := p.FetchKlines(ctx, "00700.HK", "60m", 250)
	if err != nil {
		fmt.Fprintf(os.Stderr, "fetch: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("bars=%d trade_date=%s\n", len(series.Bars), series.TradeDate.Format("2006-01-02 15:04:05"))

	type row struct {
		Index            string `json:"index"`
		AlgoDifferent    bool   `json:"algo_code_paths_differ"`
		SeriesIdentical  bool   `json:"series_identical"`
		SignalNonZero    int    `json:"signal_nonzero_bars"`
		FlagNonZero      int    `json:"flag_nonzero_bars"`
		SignalLast       int    `json:"signal_last"`
		FlagLast         int    `json:"flag_last"`
		SignalSparseOK   bool   `json:"signal_sparser_or_equal"`
		Verdict          string `json:"verdict"`
		SignalAlgo       string `json:"signal_algo"`
		FlagAlgo         string `json:"flag_algo"`
	}

	algos := map[string][2]string{
		"RSITHREHOLD":    {"threshold cross (enter/exit zone)", "zone state (in oversold/overbought)"},
		"RSI":            {"threshold cross (enter/exit zone)", "zone state (in oversold/overbought)"},
		"RSICROSS":       {"fast RSI crosses slow RSI", "slow RSI crosses fast RSI (inverted cross)"},
		"SAR":            {"SAR direction flip edge", "close vs SAR continuous state"},
		"MACD":           {"DIF/DEA cross with zero-side filter", "hist vs 0 continuous state"},
		"EMA":            {"fast crosses medium", "close vs slow continuous state"},
		"CHANDELIEREXIT": {"dir flip from chandelier levels", "close vs chandelier continuous state"},
		"BBAND":          {"band touch + bar direction edge", "band position continuous state"},
		"VWAP":           {"close crosses VWAP edge", "close vs VWAP continuous state"},
		"HEIKINASHI":     {"HA color flip edge", "HA color continuous state"},
		"KDJ":            {"K threshold cross (+D filter)", "K&D zone state vs thresholds"},
	}

	out := make([]row, 0, len(checks))
	for _, ch := range checks {
		cfgS := indexengine.Config{Index: ch.Index, Type: "signal", Param: ch.Param}
		cfgF := indexengine.Config{Index: ch.Index, Type: "flag", Param: ch.Param}
		sig, err1 := indexengine.RunSeries(series.Bars, cfgS)
		flag, err2 := indexengine.RunSeries(series.Bars, cfgF)
		r := row{Index: ch.Index, AlgoDifferent: true}
		if err1 != nil || err2 != nil {
			r.Verdict = fmt.Sprintf("ERROR sig=%v flag=%v", err1, err2)
			out = append(out, r)
			continue
		}
		identical := true
		snz, fnz := 0, 0
		for i := range sig {
			if sig[i] != flag[i] {
				identical = false
			}
			if sig[i] != 0 {
				snz++
			}
			if flag[i] != 0 {
				fnz++
			}
		}
		r.SeriesIdentical = identical
		r.SignalNonZero = snz
		r.FlagNonZero = fnz
		r.SignalLast = sig[len(sig)-1]
		r.FlagLast = flag[len(flag)-1]
		r.SignalSparseOK = snz <= fnz
		key := cfgS.Index
		// normalize key like engine
		cfgS = indexengine.Config{Index: ch.Index, Type: "signal", Param: ch.Param}
		_ = cfgS
		norm, _ := indexengine.ParseConfigs(mustJSON(map[string]any{"index": ch.Index, "type": "signal", "param": ch.Param}))
		nk := norm[0].Index
		if a, ok := algos[nk]; ok {
			r.SignalAlgo = a[0]
			r.FlagAlgo = a[1]
		}
		switch {
		case identical && nk == "RSICROSS":
			// inverted crosses can coincide on a quiet series; still different code paths
			r.Verdict = "PASS_ALGO_DIFF_SERIES_SAME_THIS_SAMPLE"
		case identical:
			r.Verdict = "FAIL_SERIES_IDENTICAL"
		case snz > fnz*2 && fnz > 5:
			r.Verdict = "WARN_SIGNAL_DENSER_THAN_FLAG"
		default:
			r.Verdict = "PASS_DIFFERENT"
		}
		out = append(out, r)
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(out)
}

func mustJSON(v any) []byte {
	b, _ := json.Marshal(v)
	return b
}
'''

REMOTE = r'''
import json, os, subprocess, textwrap

os.chdir('/root/apps/GeeGooSignal')
for line in open('.env'):
    line=line.strip()
    if line and not line.startswith('#') and '=' in line:
        k,v=line.split('=',1)
        os.environ.setdefault(k.strip(), v.strip().strip('"').strip("'"))

checks = ''' + json.dumps([{"index": i, "param": p} for i, p in CHECKS]) + r'''
src = ''' + json.dumps(GO_SRC) + r'''

os.makedirs('/tmp/sigflagcheck', exist_ok=True)
open('/tmp/sigflagcheck/main.go','w').write(src)
# run from module root so imports resolve
cmd = ['go','run','/tmp/sigflagcheck/main.go', json.dumps(checks)]
env = os.environ.copy()
p = subprocess.run(cmd, cwd='/root/apps/GeeGooSignal', env=env, capture_output=True, text=True, timeout=180)
print(p.stdout)
if p.returncode != 0:
    print('STDERR', p.stderr[-2000:])
    raise SystemExit(p.returncode)
'''


def main() -> int:
    cfg = json.loads(DEPLOY.read_text(encoding="utf-8"))
    s = cfg["targets"]["geegoo-signal"]["ssh"]
    c = paramiko.SSHClient()
    c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    c.connect(s["host"], username=s["user"], password=s.get("password"), timeout=30)
    p = "/tmp/compare_sig_flag.py"
    with c.open_sftp().file(p, "w") as f:
        f.write(REMOTE)
    _, o, e = c.exec_command(f"python3 {p}", timeout=300)
    print((o.read() + e.read()).decode("utf-8", errors="replace"))
    c.close()
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
