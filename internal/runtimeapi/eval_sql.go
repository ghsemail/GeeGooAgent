package runtimeapi

import (
	"strconv"
	"strings"
)

func (h *Handler) evalSQL(sql string) string {
	if h.usesPostgresEval() {
		return rebindQuestionSQL(sql)
	}
	return sql
}

func (h *Handler) usesPostgresEval() bool {
	return h.App != nil && h.App.PG != nil
}

func rebindQuestionSQL(sql string) string {
	var b strings.Builder
	b.Grow(len(sql) + 8)
	n := 1
	for i := 0; i < len(sql); i++ {
		if sql[i] == '?' {
			b.WriteByte('$')
			b.WriteString(strconv.Itoa(n))
			n++
			continue
		}
		b.WriteByte(sql[i])
	}
	return b.String()
}
