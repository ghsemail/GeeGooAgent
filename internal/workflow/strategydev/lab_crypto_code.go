package strategydev

import "strings"

func labCodeLooksCrypto(code string) bool {
	u := strings.ToUpper(strings.TrimSpace(code))
	if strings.Contains(u, ".CRYPTO") || strings.HasPrefix(u, "CRYPTO.") {
		return true
	}
	compact := strings.NewReplacer("-", "", "_", "", "/", "").Replace(u)
	for _, quote := range []string{"USDT", "USDC", "BUSD", "FDUSD", "TUSD"} {
		if strings.HasSuffix(compact, quote) && len(compact) > len(quote) {
			return true
		}
	}
	return false
}
