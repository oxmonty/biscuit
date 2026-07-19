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
	// Impact is a %d/%s template ("%d command%s ...") — the CLI fills the
	// count in once findings are grouped by rule. Empty when the rule is
	// generic quality with no generation-specific story.
	Impact      string
	Remediation string // fix advice paired with Impact; empty when Impact is empty
}

// impact and remediation map vacuum rule ids to what the finding means for
// the generated CLI and how to fix it. Advisory only — blocking correctness
// lives in spec.Load. Tuning this set on the test-ladder specs is a tracked
// open question in the PRD.
var impact = map[string]string{
	"operation-operationId": "%d command%s path-derived instead of named (e.g. post-v1-users-id-activate)",
	"operation-tags":        "%d operation%s untagged: command tree flattens",
	"operation-description": "%d command%s missing a description: generated --help empty, which guts agent usability",
	"operation-summary":     "%d command%s missing a summary: short help empty",
	"oas3-missing-example":  "%d site%s missing an example: mock-server responses and bench corpus weakened",
	"component-description": "%d type%s missing a description: generated flag help empty",
}

var remediation = map[string]string{
	"operation-operationId": "fix in the spec, or map names in biscuit.yaml",
	"operation-tags":        "tag them to group commands by resource",
	"operation-description": "add descriptions",
	"operation-summary":     "add summaries",
	"oas3-missing-example":  "add examples",
	"component-description": "add descriptions",
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
		severity := r.RuleSeverity
		if severity == "" {
			severity = "info" // vacuum's circular-references rule leaves this blank
		}
		report.Findings = append(report.Findings, Finding{
			Rule:        r.RuleId,
			Severity:    severity,
			Message:     r.Message,
			Line:        r.Range.Start.Line,
			Impact:      impact[r.RuleId],
			Remediation: remediation[r.RuleId],
		})
	}
	sort.Slice(report.Findings, func(i, j int) bool {
		a, b := report.Findings[i], report.Findings[j]
		if ra, rb := severityRank(a.Severity), severityRank(b.Severity); ra != rb {
			return ra < rb
		}
		if a.Rule != b.Rule {
			return a.Rule < b.Rule
		}
		return a.Line < b.Line
	})
	return report
}

func severityRank(severity string) int {
	switch severity {
	case "error":
		return 0
	case "warn":
		return 1
	default:
		return 2
	}
}
