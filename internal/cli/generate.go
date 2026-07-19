package cli

import (
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"

	"github.com/oxmonty/biscuit/internal/config"
	"github.com/oxmonty/biscuit/internal/ir"
	"github.com/oxmonty/biscuit/internal/lint"
	"github.com/oxmonty/biscuit/internal/mapping"
	"github.com/oxmonty/biscuit/internal/spec"
)

func newGenerateCommand() *cobra.Command {
	var specPath string
	var dryRun, showFlags, quiet, strict bool

	cmd := &cobra.Command{
		Use:   "generate",
		Short: "Generate the CLI repository for a spec",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := config.Load(".")
			if err != nil {
				return &usageError{err}
			}
			path, err := resolveSpecPath(cmd, specPath, cfg)
			if err != nil {
				return err
			}
			doc, err := spec.Load(path)
			if err != nil {
				return err
			}

			// generate runs doctor implicitly: blocking problems already
			// failed in Load; advisories surface as one line unless --quiet
			report := lint.Run(doc)
			if !quiet && len(report.Findings) > 0 {
				_, _ = fmt.Fprintf(cmd.ErrOrStderr(),
					"spec grade %d/100 — %d advisory findings (biscuit doctor for detail)\n",
					report.Grade, len(report.Findings))
			}
			switch {
			case strict && len(report.Findings) > 0:
				return &qualityGateError{fmt.Sprintf("--strict: %d advisory findings", len(report.Findings))}
			case cfg.Lint.MinGrade > 0 && report.Grade < cfg.Lint.MinGrade:
				return &qualityGateError{fmt.Sprintf("grade %d below lint.min_grade %d", report.Grade, cfg.Lint.MinGrade)}
			}

			if !dryRun {
				return &usageError{fmt.Errorf("repository rendering ships with the next milestone; preview the command surface with --dry-run")}
			}

			api := mapping.Map(doc, mapping.OverridesFromConfig(cfg))
			printDryRun(cmd.OutOrStdout(), api, showFlags)
			return nil
		},
	}
	cmd.Flags().StringVar(&specPath, "spec", "", "path to the OpenAPI spec (default: discover it)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "print the derived command surface and file plan without writing")
	cmd.Flags().BoolVar(&showFlags, "flags", false, "with --dry-run: list every derived flag")
	cmd.Flags().BoolVar(&quiet, "quiet", false, "suppress the advisory-findings summary")
	cmd.Flags().BoolVar(&strict, "strict", false, "fail on any advisory finding (exit 5)")
	return cmd
}

func printDryRun(out io.Writer, api *ir.API, showFlags bool) {
	resources, verbs := countTree(api.Commands)
	verbs += len(api.RootVerbs)
	title := api.Title
	if title == "" {
		title = "(untitled spec)"
	}
	_, _ = fmt.Fprintf(out, "%s — %d operations → %d resources, %d commands\n\n",
		title, len(api.Operations), resources, verbs)

	printCommands(out, api.Commands, "", showFlags)
	for i := range api.RootVerbs {
		printVerb(out, &api.RootVerbs[i], "", showFlags)
	}

	if len(api.Diagnostics) > 0 {
		_, _ = fmt.Fprintln(out, "\ndiagnostics:")
		for _, d := range api.Diagnostics {
			_, _ = fmt.Fprintf(out, "  - %s\n", d)
		}
	}
	_, _ = fmt.Fprintln(out, "\nfile plan: 0 files (repository rendering ships with the next milestone)")
}

func printCommands(out io.Writer, cmds []ir.Command, indent string, showFlags bool) {
	for i := range cmds {
		c := &cmds[i]
		_, _ = fmt.Fprintf(out, "%s%s\n", indent, c.Name)
		for j := range c.Verbs {
			printVerb(out, &c.Verbs[j], indent+"  ", showFlags)
		}
		printCommands(out, c.Children, indent+"  ", showFlags)
	}
}

func printVerb(out io.Writer, v *ir.Verb, indent string, showFlags bool) {
	suffix := fmt.Sprintf("(%d flag%s)", len(v.Flags), plural(len(v.Flags)))
	if v.Deprecated {
		suffix += " deprecated"
	}
	_, _ = fmt.Fprintf(out, "%s%-28s %s %s  %s\n", indent, v.Name, v.Method, v.Path, suffix)
	if !showFlags {
		return
	}
	for _, f := range v.Flags {
		var notes []string
		if f.Required {
			notes = append(notes, "required")
		}
		if f.Repeated {
			notes = append(notes, "repeated")
		}
		if len(f.Enum) > 0 {
			notes = append(notes, "enum")
		}
		if f.Union != nil {
			notes = append(notes, "oneOf via "+f.Union.Kind)
		}
		note := ""
		if len(notes) > 0 {
			note = "  (" + strings.Join(notes, ", ") + ")"
		}
		_, _ = fmt.Fprintf(out, "%s    --%s %s%s\n", indent, f.Name, f.Type, note)
	}
}

func countTree(cmds []ir.Command) (resources, verbs int) {
	for _, c := range cmds {
		resources++
		verbs += len(c.Verbs)
		r, v := countTree(c.Children)
		resources += r
		verbs += v
	}
	return resources, verbs
}
