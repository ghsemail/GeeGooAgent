//go:build ignore

// Usage: go run scripts/dev/gen_turnplan_eval_json.go
package main

import (
	"encoding/json"
	"fmt"

	"github.com/ghsemail/GeeGooAgent/internal/eval"
)

func main() {
	b, err := json.Marshal(eval.DefaultTurnPlanSuite())
	if err != nil {
		panic(err)
	}
	fmt.Println(string(b))
}
