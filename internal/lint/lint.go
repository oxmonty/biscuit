// Package lint grades a spec with vacuum and translates rule hits into
// generation impact: what each finding does to the generated CLI and how to
// fix it — in the spec if you own it, in biscuit.yaml if you don't.
package lint

import (
	"log/slog"
	"sort"

	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/motor"
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/daveshanley/vacuum/statistics"

	"github.com/oxmonty/biscuit/internal/spec"
)

type Report struct {
	Grade    int // vacuum quality score, 0-100
	Findings []Finding
}

type Finding struct {
	Rule     string
	Severity string
	Message  string
	Line     int
	Impact   string // generation impact + remediation; empty when the rule is generic quality
}

// impact maps vacuum rule ids to what the finding means for the generated CLI.
// Advisory only — blocking correctness lives in spec.Load. Tuning this set on
// the test-ladder specs is a tracked open question in the PRD.
var impact = map[string]string{
	"operation-operationId": "commands will be path-derived (e.g. post-v1-users-id-activate); fix in the spec, or map names in biscuit.yaml",
	"operation-tags":        "untagged operations flatten the command tree; tag them to group commands by resource",
	"operation-description": "generated --help will be empty for these commands, which guts agent usability; add descriptions",
	"operation-summary":     "command short help will be empty; add summaries",
	"oas3-missing-example":  "no schema examples weakens mock-server responses and the synthesized bench corpus",
	"component-description": "generated flag help for these types will be empty; add descriptions",
}

// biscuitRuleSet is the thin generation-relevant selection from vacuum's
// built-in rules: spec style (casing, duplication) doesn't move generated-CLI
// quality, so it stays out and keeps the grade meaningful for min_grade.
// Which rules belong here is a tracked open question, tuned on the ladder.
func biscuitRuleSet() *rulesets.RuleSet {
	relevant := []string{
		"operation-operationId", "operation-operationId-unique", "operation-tags",
		"operation-tag-defined", "operation-description", "operation-summary",
		"oas3-missing-example", "component-description", "oas3-parameter-description",
		"oas-schema-check", "path-params", "no-ambiguous-paths",
		"duplicated-entry-in-enum", "oas3-schema", "oas3-valid-schema-example",
	}
	all := rulesets.GetAllBuiltInRules()
	picked := map[string]*model.Rule{}
	for _, id := range relevant {
		if r, ok := all[id]; ok {
			picked[id] = r
		}
	}
	return rulesets.CreateRuleSetFromRuleMap(picked)
}

// Run grades the loaded spec against the biscuit ruleset.
func Run(doc *spec.Document) *Report {
	execution := motor.ApplyRulesToRuleSet(&motor.RuleSetExecution{
		RuleSet:      biscuitRuleSet(),
		Spec:         doc.Bytes,
		SpecFileName: doc.Path,
		SilenceLogs:  true,
		Logger:       slog.New(slog.DiscardHandler),
	})

	report := &Report{
		Grade: statistics.CalculateQualityScore(model.NewRuleResultSet(execution.Results)),
	}
	for _, r := range execution.Results {
		report.Findings = append(report.Findings, Finding{
			Rule:     r.RuleId,
			Severity: r.RuleSeverity,
			Message:  r.Message,
			Line:     r.Range.Start.Line,
			Impact:   impact[r.RuleId],
		})
	}
	sort.Slice(report.Findings, func(i, j int) bool {
		if report.Findings[i].Rule != report.Findings[j].Rule {
			return report.Findings[i].Rule < report.Findings[j].Rule
		}
		return report.Findings[i].Line < report.Findings[j].Line
	})
	return report
}
