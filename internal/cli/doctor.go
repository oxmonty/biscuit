package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/spf13/cobra"
	"go.yaml.in/yaml/v4"

	"github.com/oxmonty/biscuit/internal/lint"
	"github.com/oxmonty/biscuit/internal/spec"
)

// qualityGateError marks a failed --strict / lint.min_grade gate; exit code 5.
type qualityGateError struct{ msg string }

func (e *qualityGateError) Error() string { return e.msg }

func newDoctorCommand() *cobra.Command {
	var specPath string
	var strict bool
	var format string

	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Grade a spec and report what its gaps do to the generated CLI",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if format != "text" && format != "json" {
				return &usageError{fmt.Errorf("--format must be text or json, got %q", format)}
			}

			path, err := resolveSpecPath(cmd, specPath)
			if err != nil {
				return err
			}
			doc, err := spec.Load(path)
			if err != nil {
				return err // InvalidError formats the blocking report itself
			}

			report := lint.Run(doc)
			groups := groupFindings(report.Findings)
			minGrade := loadMinGrade()
			blocking := (strict && len(report.Findings) > 0) || (minGrade > 0 && report.Grade < minGrade)

			if format == "json" {
				writeJSONReport(cmd.OutOrStdout(), doc.Path, report.Grade, groups, doc.Diagnostics, blocking)
			} else {
				writeTextReport(cmd.OutOrStdout(), doc.Path, report, groups, doc.Diagnostics)
			}

			switch {
			case strict && len(report.Findings) > 0:
				return &qualityGateError{fmt.Sprintf("--strict: %d advisory findings", len(report.Findings))}
			case minGrade > 0 && report.Grade < minGrade:
				return &qualityGateError{fmt.Sprintf("grade %d below lint.min_grade %d", report.Grade, minGrade)}
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&specPath, "spec", "", "path to the OpenAPI spec (default: discover it)")
	cmd.Flags().BoolVar(&strict, "strict", false, "fail on any advisory finding (exit 5)")
	cmd.Flags().StringVar(&format, "format", "text", "output format: text or json")
	return cmd
}

// findingGroup collapses same-rule findings into one row: doctor reports per
// rule, not per occurrence, with the occurrence count folded into Impact
// once it's known, rather than printed as a separate column.
type findingGroup struct {
	rule        string
	severity    string
	message     string
	count       int
	impact      string // rendered with the count folded in; empty when no rule carries generation impact
	remediation string
}

func groupFindings(findings []lint.Finding) []findingGroup {
	index := map[string]int{}
	var groups []findingGroup
	for _, f := range findings {
		i, seen := index[f.Rule]
		if !seen {
			i = len(groups)
			index[f.Rule] = i
			groups = append(groups, findingGroup{
				rule: f.Rule, severity: f.Severity, message: f.Message,
				impact: f.Impact, remediation: f.Remediation,
			})
		}
		groups[i].count++
	}
	for i := range groups {
		if groups[i].impact != "" {
			groups[i].impact = fmt.Sprintf(groups[i].impact, groups[i].count, plural(groups[i].count))
		}
	}
	return groups
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

// writeTextReport renders the humane doctor report: grouped findings with
// counts folded into the impact sentence, resolver diagnostics in plain
// English, and severity colors when out is a real terminal.
func writeTextReport(out io.Writer, path string, report *lint.Report, groups []findingGroup, diagnostics []string) {
	color := isTTY(out)
	_, _ = fmt.Fprintf(out, "%s — grade %d/100\n", path, report.Grade)
	for _, g := range groups {
		label := severityLabel(g.severity, color)
		if g.impact != "" {
			_, _ = fmt.Fprintf(out, "  %s %s — e.g. %s\n", label, g.rule, g.message)
			_, _ = fmt.Fprintf(out, "        → %s; %s\n", g.impact, g.remediation)
		} else {
			_, _ = fmt.Fprintf(out, "  %s %d× %s — e.g. %s\n", label, g.count, g.rule, g.message)
		}
	}
	for _, d := range diagnostics {
		_, _ = fmt.Fprintf(out, "  %s %s\n", severityLabel("info", color), humanizeDiagnostic(d))
	}
	if len(report.Findings) > 0 {
		printSummary(out, report.Findings)
	}
}

// printSummary prints one footer line with per-severity counts and the gate
// hint, so a run with no --strict / lint.min_grade set doesn't read as clean.
func printSummary(out io.Writer, findings []lint.Finding) {
	var errors, warnings, info int
	for _, f := range findings {
		switch f.Severity {
		case "error":
			errors++
		case "warn":
			warnings++
		default:
			info++
		}
	}
	_, _ = fmt.Fprintf(out, "%d errors, %d warnings, %d info — advisory only, generation not blocked (gate with --strict or lint.min_grade)\n", errors, warnings, info)
}

// severityLabel renders "[error]" etc., wrapped in ANSI color when color is true.
func severityLabel(severity string, color bool) string {
	label := "[" + severity + "]"
	if !color {
		return label
	}
	code := "36" // cyan for info
	switch severity {
	case "error":
		code = "31"
	case "warn":
		code = "33"
	}
	return "\x1b[" + code + "m" + label + "\x1b[0m"
}

// isTTY reports whether w is a real terminal, not a pipe/file/buffer — text
// output only colors when writing to one.
func isTTY(w io.Writer) bool {
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	stat, err := f.Stat()
	return err == nil && stat.Mode()&os.ModeCharDevice != 0
}

type jsonReport struct {
	Spec        string        `json:"spec"`
	Grade       int           `json:"grade"`
	Findings    []jsonFinding `json:"findings"`
	Diagnostics []string      `json:"diagnostics"`
	Blocking    bool          `json:"blocking"`
}

type jsonFinding struct {
	Rule        string `json:"rule"`
	Severity    string `json:"severity"`
	Count       int    `json:"count"`
	Impact      string `json:"impact,omitempty"`
	Remediation string `json:"remediation,omitempty"`
}

// writeJSONReport emits the same report as writeTextReport, machine-readable
// for CI pipelines; the exit-code contract is decided by the caller either way.
func writeJSONReport(out io.Writer, path string, grade int, groups []findingGroup, diagnostics []string, blocking bool) {
	jr := jsonReport{
		Spec:        path,
		Grade:       grade,
		Findings:    make([]jsonFinding, 0, len(groups)),
		Diagnostics: make([]string, len(diagnostics)),
		Blocking:    blocking,
	}
	for _, g := range groups {
		jr.Findings = append(jr.Findings, jsonFinding{
			Rule: g.rule, Severity: g.severity, Count: g.count,
			Impact: g.impact, Remediation: g.remediation,
		})
	}
	for i, d := range diagnostics {
		jr.Diagnostics[i] = humanizeDiagnostic(d)
	}
	enc := json.NewEncoder(out)
	enc.SetIndent("", "  ")
	_ = enc.Encode(jr)
}

var diagFileRe = regexp.MustCompile(`\(file: (.+?)\)`)

// humanizeDiagnostic turns a raw libopenapi/resolver diagnostic — an
// absolute rolodex path wrapped in Go error text, or a bare ref-lookup
// error — into one plain-English line with a relative path.
func humanizeDiagnostic(raw string) string {
	switch {
	case strings.HasPrefix(raw, "circular reference: "):
		return strings.ReplaceAll(raw, " -> ", " → ")
	case strings.HasPrefix(raw, "component `") && strings.HasSuffix(raw, "` does not exist in the specification"):
		ref := strings.TrimSuffix(strings.TrimPrefix(raw, "component `"), "` does not exist in the specification")
		return fmt.Sprintf("%s: referenced but missing from the spec", relPath(ref))
	case strings.Contains(raw, "rolodex file") || strings.Contains(raw, "locate file in the rolodex"):
		if m := diagFileRe.FindStringSubmatch(raw); m != nil {
			return fmt.Sprintf("%s: file not found", relPath(m[1]))
		}
	}
	return raw
}

// relPath shortens an absolute path to one relative to the working
// directory, falling back to the base name — doctor output never dumps a
// full rolodex path.
func relPath(p string) string {
	p = strings.TrimPrefix(p, "./")
	if !filepath.IsAbs(p) {
		return p
	}
	if wd, err := os.Getwd(); err == nil {
		if rel, err := filepath.Rel(wd, p); err == nil && !strings.HasPrefix(rel, "..") {
			return rel
		}
	}
	return filepath.Base(p)
}

// loadMinGrade reads lint.min_grade from ./biscuit.yaml if present.
// ponytail: E3 replaces this with the schema-validated config loader;
// until then only this one key is read, and silently.
func loadMinGrade() int {
	data, err := os.ReadFile("biscuit.yaml")
	if err != nil {
		return 0
	}
	var cfg struct {
		Lint struct {
			MinGrade int `yaml:"min_grade"`
		} `yaml:"lint"`
	}
	if yaml.Unmarshal(data, &cfg) != nil {
		return 0
	}
	return cfg.Lint.MinGrade
}
