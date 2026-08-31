package tools

import (
	"fmt"
	"sort"

	"github.com/ghsemail/GeeGooAgent/internal/tools/catalog"
)

// DriftIssue describes catalog ↔ registry mismatch.
type DriftIssue struct {
	Kind   string
	Detail string
}

// CheckCatalogDrift compares HTTP catalog specs with the live registry.
func CheckCatalogDrift(r *Registry) []DriftIssue {
	if r == nil {
		return []DriftIssue{{Kind: "registry", Detail: "nil registry"}}
	}
	var issues []DriftIssue
	registered := map[string]struct{}{}
	for _, name := range r.ListNames() {
		registered[name] = struct{}{}
	}
	for _, spec := range catalog.AllHTTP() {
		if catalog.BespokeNames[spec.Name] {
			continue
		}
		if _, ok := registered[spec.Name]; !ok {
			issues = append(issues, DriftIssue{
				Kind:   "missing_registry",
				Detail: spec.Name,
			})
		}
	}
	for name := range registered {
		if catalog.BespokeNames[name] {
			continue
		}
		found := false
		for _, spec := range catalog.AllHTTP() {
			if spec.Name == name {
				found = true
				break
			}
		}
		if !found && toolDomain(name) == DomainMeta {
			issues = append(issues, DriftIssue{
				Kind:   "extra_registry",
				Detail: name,
			})
		}
	}
	sort.Slice(issues, func(i, j int) bool {
		if issues[i].Kind != issues[j].Kind {
			return issues[i].Kind < issues[j].Kind
		}
		return issues[i].Detail < issues[j].Detail
	})
	return issues
}

// FormatDriftIssues renders drift issues for doctor / inspect.
func FormatDriftIssues(issues []DriftIssue) string {
	if len(issues) == 0 {
		return "no drift"
	}
	parts := make([]string, 0, len(issues))
	for _, issue := range issues {
		parts = append(parts, fmt.Sprintf("%s:%s", issue.Kind, issue.Detail))
	}
	return fmt.Sprintf("%d issue(s): %v", len(issues), parts)
}

// PolicyRuleCount returns loaded declarative policy rules.
func PolicyRuleCount() int {
	return len(policyRulesSnapshot())
}

func policyRulesSnapshot() []policyRule {
	policyMu.RLock()
	defer policyMu.RUnlock()
	out := make([]policyRule, len(policyRules))
	copy(out, policyRules)
	return out
}
