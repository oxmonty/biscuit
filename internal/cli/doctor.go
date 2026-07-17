package cli

import (
	"fmt"
	"io"
	"os"

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

	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Grade a spec and report what its gaps do to the generated CLI",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			path, err := resolveSpecPath(cmd, specPath)
			if err != nil {
				return err
			}
			doc, err := spec.Load(path)
			if err != nil {
				return err // InvalidError formats the blocking report itself
			}

			report := lint.Run(doc)
			out := cmd.OutOrStdout()
			_, _ = fmt.Fprintf(out, "%s — grade %d/100\n", doc.Path, report.Grade)
			printGrouped(out, report.Findings)
			for _, d := range doc.Diagnostics {
				_, _ = fmt.Fprintf(out, "  [info] %s\n", d)
			}
			if len(report.Findings) > 0 {
				printSummary(out, report.Findings)
			}

			minGrade := loadMinGrade()
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
	return cmd
}

// printGrouped renders findings as one line per rule — "N × rule: sample" —
// with the generation impact once beneath it, per the PRD's doctor UX.
func printGrouped(out io.Writer, findings []lint.Finding) {
	type group struct {
		count    int
		first    lint.Finding
		severity string
	}
	groups := map[string]*group{}
	var order []string
	for _, f := range findings {
		g, seen := groups[f.Rule]
		if !seen {
			g = &group{first: f, severity: f.Severity}
			groups[f.Rule] = g
			order = append(order, f.Rule)
		}
		g.count++
	}
	for _, rule := range order {
		g := groups[rule]
		_, _ = fmt.Fprintf(out, "  [%s] %d× %s — e.g. %s\n", g.severity, g.count, rule, g.first.Message)
		if g.first.Impact != "" {
			_, _ = fmt.Fprintf(out, "        → %s\n", g.first.Impact)
		}
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
