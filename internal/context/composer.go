package context

import "strings"

// Kind identifies a dynamic context fragment source.
type Kind string

const (
	KindToolResult     Kind = "tool_result"
	KindRecall         Kind = "recall"
	KindProcedural     Kind = "procedural"
	KindWorkingState   Kind = "working_state"
	KindHookInject     Kind = "hook_inject"
	KindSystemRules    Kind = "system_rules"
	KindBudgetReminder Kind = "budget_reminder"
	KindClock          Kind = "clock"
)

// Fragment is one injectable user-side context block with compression priority.
type Fragment interface {
	Kind() Kind
	Render() string
	TokenEstimate() int
	Priority() int // lower = kept longer when compressing
}

// StaticFragment is a simple text fragment.
type StaticFragment struct {
	K          Kind
	Text       string
	Prio       int
	TokenEst   int
}

func (f StaticFragment) Kind() Kind { return f.K }
func (f StaticFragment) Render() string {
	return strings.TrimSpace(f.Text)
}
func (f StaticFragment) TokenEstimate() int {
	if f.TokenEst > 0 {
		return f.TokenEst
	}
	return len([]byte(f.Render())) / 4
}
func (f StaticFragment) Priority() int {
	if f.Prio > 0 {
		return f.Prio
	}
	return 50
}

// Composer merges fragments by priority under a byte budget.
type Composer struct {
	MaxBytes int
}

// Compose renders fragments; drops lowest priority first when over budget.
func (c Composer) Compose(fragments []Fragment) (text string, applied []Kind, dropped []Kind) {
	max := c.MaxBytes
	if max <= 0 {
		max = 32 * 1024
	}
	sorted := append([]Fragment(nil), fragments...)
	// Stable sort by priority ascending (keep low priority number = high importance... 
	// spec says drop LOW priority first = HIGH priority number first to drop)
	// Priority(): lower number = higher importance = keep longer
	// So when over budget, drop from highest Priority() value first.
	type item struct {
		f Fragment
	}
	items := make([]item, len(sorted))
	for i, f := range sorted {
		items[i] = item{f: f}
	}
	// Drop from end after sorting by priority desc (drop least important)
	for len(items) > 0 {
		var b strings.Builder
		applied = applied[:0]
		for _, it := range items {
			part := strings.TrimSpace(it.f.Render())
			if part == "" {
				continue
			}
			if b.Len() > 0 {
				b.WriteString("\n\n")
			}
			b.WriteString(part)
			applied = append(applied, it.f.Kind())
		}
		if len([]byte(b.String())) <= max || len(items) == 1 {
			return b.String(), applied, dropped
		}
		// drop the fragment with max priority value (least important)
		worstIdx := 0
		worstPrio := items[0].f.Priority()
		for i := 1; i < len(items); i++ {
			if items[i].f.Priority() > worstPrio {
				worstPrio = items[i].f.Priority()
				worstIdx = i
			}
		}
		dropped = append(dropped, items[worstIdx].f.Kind())
		items = append(items[:worstIdx], items[worstIdx+1:]...)
	}
	return "", nil, dropped
}
