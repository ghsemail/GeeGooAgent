package context

import (
	"fmt"
	"sync"
	"time"
)

var (
	shanghaiOnce sync.Once
	shanghaiLoc  *time.Location
)

var weekdayCN = [...]string{"星期日", "星期一", "星期二", "星期三", "星期四", "星期五", "星期六"}

func shanghai() *time.Location {
	shanghaiOnce.Do(func() {
		loc, err := time.LoadLocation("Asia/Shanghai")
		if err != nil {
			loc = time.FixedZone("CST", 8*3600)
		}
		shanghaiLoc = loc
	})
	return shanghaiLoc
}

// ClockFragment injects the current wall clock (Asia/Shanghai) for relative dates.
// Priority 1 so it survives fragment compression.
func ClockFragment(now time.Time) Fragment {
	t := now.In(shanghai())
	text := fmt.Sprintf(
		"当前时间：%s %s（Asia/Shanghai）\n「今天 / 今早 / 本周 / 昨天」一律按此时钟解释。不要用新闻日期、K 线日期或记忆片段推断今天。工具返回的日期若不是今天，必须标明是历史数据。",
		t.Format("2006-01-02 15:04"),
		weekdayCN[t.Weekday()],
	)
	return StaticFragment{K: KindClock, Text: text, Prio: 1}
}
