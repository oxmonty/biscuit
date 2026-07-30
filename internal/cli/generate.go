package cli

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"

	"github.com/oxmonty/biscuit/internal/config"
	"github.com/oxmonty/biscuit/internal/ir"
	"github.com/oxmonty/biscuit/internal/lint"
	"github.com/oxmonty/biscuit/internal/mapping"
	"github.com/oxmonty/biscuit/internal/render"
	"github.com/oxmonty/biscuit/internal/spec"
)

func newGenerateCommand() *cobra.Command {
	var specPath, outDir string
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
			if err := mapping.ValidatePagination(cfg); err != nil {
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

			api := mapping.Map(doc, cfg)
			sum := sha256.Sum256(doc.Bytes)
			files, err := render.Render(api, cfg, render.Provenance{
				SpecPath:   doc.Path,
				SpecSHA256: hex.EncodeToString(sum[:]),
			})
			if err != nil {
				return err
			}

			if dryRun {
				printDryRun(cmd.OutOrStdout(), api, files, showFlags)
				return nil
			}

			dir := outDir
			if dir == "" {
				dir = render.OutputDir(api, cfg)
			}
			res, err := render.WriteFiles(dir, files)
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			_, _ = fmt.Fprintf(out, "wrote %d files to %s\n", len(res.Written), dir)
			for _, p := range res.SkippedCustom {
				_, _ = fmt.Fprintf(out, "kept %s (yours, never regenerated)\n", p)
			}
			for _, p := range res.SkippedUnmarked {
				_, _ = fmt.Fprintf(out, "kept %s (no generated-file marker; user-owned)\n", p)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&specPath, "spec", "", "path to the OpenAPI spec (default: discover it)")
	cmd.Flags().StringVar(&outDir, "out", "", "output directory (default: output.dir or ./{binary}-cli)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "print the derived command surface and file plan without writing")
	cmd.Flags().BoolVar(&showFlags, "flags", false, "with --dry-run: list every derived flag")
	cmd.Flags().BoolVar(&quiet, "quiet", false, "suppress the advisory-findings summary")
	cmd.Flags().BoolVar(&strict, "strict", false, "fail on any advisory finding (exit 5)")
	return cmd
}

func printDryRun(out io.Writer, api *ir.API, files []render.File, showFlags bool) {
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
	_, _ = fmt.Fprintf(out, "\nfile plan: %d files\n", len(files))
	for _, f := range files {
		note := ""
		if f.EmitOnce {
			note = "  (emitted once, never overwritten)"
		}
		_, _ = fmt.Fprintf(out, "  %s%s\n", f.Path, note)
	}
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
	if v.Pagination != nil {
		suffix += " [paginated: " + v.Pagination.Scheme + "]"
	}
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
