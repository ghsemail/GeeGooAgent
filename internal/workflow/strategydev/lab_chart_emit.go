package strategydev

import kp "github.com/ghsemail/GeeGooAgent/internal/lab/klinepattern"

// PrepareLabEpisodeDetailForEmit strips non-anchor chart pattern markers from lab-detail JSON.
// Call after PA-JEV and jev_unified so decision/display synthesis keeps full internal inputs.
func PrepareLabEpisodeDetailForEmit(out *LabEpisodeDetail) {
	if out == nil {
		return
	}
	out.KlinePatternThreeBar = kp.ThreeBarWindowForChartEmit(out.KlinePatternThreeBar)
	out.KlinePatternMTF = kp.MTFContextForChartEmit(out.KlinePatternMTF)
	if out.KlinePattern != nil {
		out.KlinePattern.ThreeBar = kp.ThreeBarWindowForChartEmit(out.KlinePattern.ThreeBar)
		out.KlinePattern.MTF = kp.MTFContextForChartEmit(out.KlinePattern.MTF)
	}
}

// EpisodeTriggerRecordForChartEmit copies one trigger with anchor-only three-bar markers.
func EpisodeTriggerRecordForChartEmit(rec EpisodeTriggerRecord) EpisodeTriggerRecord {
	out := rec
	out.KlinePatternThreeBar = kp.ThreeBarWindowForChartEmit(rec.KlinePatternThreeBar)
	out.KlinePatternMTF = kp.MTFContextForChartEmit(rec.KlinePatternMTF)
	out.HigherTFContext = kp.MTFContextForChartEmit(rec.HigherTFContext)
	return out
}

// EpisodeTriggerRecordsForChartEmit copies triggers with anchor-only three-bar chart markers.
func EpisodeTriggerRecordsForChartEmit(list []EpisodeTriggerRecord) []EpisodeTriggerRecord {
	if len(list) == 0 {
		return nil
	}
	out := make([]EpisodeTriggerRecord, len(list))
	for i, rec := range list {
		out[i] = EpisodeTriggerRecordForChartEmit(rec)
	}
	return out
}
