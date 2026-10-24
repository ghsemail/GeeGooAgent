package strategydev

import "fmt"

// LabDetailRefMismatchHint detects a common FE mistake: using episode list index as the ref suffix
// (e.g. sell:7) while lab_episode_summaries keys use start_idx (e.g. sell:190).
func LabDetailRefMismatchHint(summaries LabCardEpisodeSummaries, ref string) string {
	if len(summaries) == 0 {
		return ""
	}
	if _, ok := summaries[ref]; ok {
		return ""
	}
	side, idx, ok := ParseLabDetailRef(ref)
	if !ok || side == "flow" {
		return ""
	}
	for _, sum := range summaries {
		if sum.EpisodeIndex != idx {
			continue
		}
		sumSide, _, sumOK := ParseLabDetailRef(sum.DetailRef)
		if !sumOK || sumSide != side {
			continue
		}
		if sum.DetailRef == ref {
			continue
		}
		return fmt.Sprintf(
			"detail_ref %q 指向图表 bar %d，但本 run 该 side 的 episode 列表序号 %d 对应 %q（start_idx=%d）；弹窗应使用摘要里的 detail_ref",
			ref, idx, idx, sum.DetailRef, sum.StartIdx,
		)
	}
	return ""
}
