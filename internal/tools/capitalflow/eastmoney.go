package capitalflow

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Snapshot is a minimal capital-flow view for A-shares via EastMoney public API.
type Snapshot struct {
	MainInFlow float64
	Source     string
}

// Distribution is T-1 style capital distribution buckets.
type Distribution struct {
	SuperIn float64
	BigIn   float64
	MidIn   float64
	SmallIn float64
	Source  string
}

// SupportsAShare reports whether the code can use the EastMoney fallback.
func SupportsAShare(code string) bool {
	return strings.HasSuffix(code, ".SH") || strings.HasSuffix(code, ".SZ")
}

// FetchAShareSnapshot loads latest main capital inflow for A-shares.
func FetchAShareSnapshot(ctx context.Context, code string) (*Snapshot, error) {
	secid, err := toSecID(code)
	if err != nil {
		return nil, err
	}
	u := fmt.Sprintf(
		"https://push2.eastmoney.com/api/qt/stock/fflow/kline/get?lmt=1&klt=101&secid=%s&fields1=f1,f2,f3,f7&fields2=f51,f52,f53,f54,f55,f56,f57,f58,f59,f60,f61,f62,f63,f64,f65",
		secid,
	)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0")
	resp, err := (&http.Client{Timeout: 15 * time.Second}).Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	var parsed struct {
		Data struct {
			Klines []string `json:"klines"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, err
	}
	if len(parsed.Data.Klines) == 0 {
		return nil, fmt.Errorf("eastmoney: no capital flow data")
	}
	// kline format: date,main_in,small_in,mid_in,big_in,super_in,...
	parts := strings.Split(parsed.Data.Klines[len(parsed.Data.Klines)-1], ",")
	if len(parts) < 2 {
		return nil, fmt.Errorf("eastmoney: unexpected kline")
	}
	var mainIn float64
	fmt.Sscanf(parts[1], "%f", &mainIn)
	return &Snapshot{MainInFlow: mainIn, Source: "eastmoney"}, nil
}

// FetchAShareDistribution loads capital distribution buckets for A-shares.
func FetchAShareDistribution(ctx context.Context, code string) (*Distribution, error) {
	secid, err := toSecID(code)
	if err != nil {
		return nil, err
	}
	u := fmt.Sprintf(
		"https://push2.eastmoney.com/api/qt/stock/get?secid=%s&fields=f62,f184,f66,f69,f72,f75,f78,f81,f84,f87,f204,f205,f124",
		secid,
	)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0")
	resp, err := (&http.Client{Timeout: 15 * time.Second}).Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	var parsed struct {
		Data map[string]any `json:"data"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, err
	}
	if parsed.Data == nil {
		return nil, fmt.Errorf("eastmoney: no distribution data")
	}
	return &Distribution{
		SuperIn: toFloat(parsed.Data["f66"]),
		BigIn:   toFloat(parsed.Data["f72"]),
		MidIn:   toFloat(parsed.Data["f78"]),
		SmallIn: toFloat(parsed.Data["f84"]),
		Source:  "eastmoney",
	}, nil
}

func toSecID(code string) (string, error) {
	code = strings.TrimSpace(code)
	var sym string
	switch {
	case strings.HasSuffix(code, ".SH"):
		sym = strings.TrimSuffix(code, ".SH")
		return "1." + sym, nil
	case strings.HasSuffix(code, ".SZ"):
		sym = strings.TrimSuffix(code, ".SZ")
		return "0." + sym, nil
	default:
		return "", fmt.Errorf("unsupported A-share code %s", code)
	}
}

func toFloat(v any) float64 {
	switch n := v.(type) {
	case float64:
		return n
	case json.Number:
		f, _ := n.Float64()
		return f
	default:
		return 0
	}
}
