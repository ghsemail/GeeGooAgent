package agent

import (
	"sync"
	"time"
)

var (
	clockNowMu sync.Mutex
	clockNowFn func() time.Time
)

func clockNow() time.Time {
	clockNowMu.Lock()
	fn := clockNowFn
	clockNowMu.Unlock()
	if fn != nil {
		return fn()
	}
	return time.Now()
}

// SetClockNowForTest overrides the wall clock used for per-turn date injection.
// Pass nil to restore time.Now.
func SetClockNowForTest(fn func() time.Time) {
	clockNowMu.Lock()
	clockNowFn = fn
	clockNowMu.Unlock()
}
