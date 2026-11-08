package strategydev

import "strings"

func labDetailDailyMacro(in LabDetailInput) bool {
	return strings.TrimSpace(in.KlineJEVProfile) == LabKlineJEVProfileDailyMacro
}

// labDetailJEVSignalFreq is the TF label passed into JEV state for macro (daily only).
func labDetailJEVSignalFreq(in LabDetailInput) string {
	if labDetailDailyMacro(in) {
		return "daily"
	}
	return strings.TrimSpace(in.SignalFreq)
}

// LabDetailCacheRef keys persisted lab-detail (episode ref + optional kline JEV profile).
func LabDetailCacheRef(detailRef, klineJEVProfile string) string {
	detailRef = strings.TrimSpace(detailRef)
	profile := strings.TrimSpace(klineJEVProfile)
	if profile == "" {
		return detailRef
	}
	return detailRef + "|" + profile
}

func LabDetailProfileMatches(cached LabEpisodeDetail, wantProfile string) bool {
	return strings.TrimSpace(cached.KlineJEVProfile) == strings.TrimSpace(wantProfile)
}
