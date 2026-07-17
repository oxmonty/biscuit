package cli

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/oxmonty/biscuit/internal/version"
)

const (
	muted  = "\033[0;2m"
	accent = "\033[38;5;214m"
	reset  = "\033[0m"
)

// logo is figlet's "big" font — chosen over a hand-drawn block font because
// it's a fixed, testable render rather than bespoke glyphs guessed blind.
const logo = ` _     _                _ _
| |   (_)              (_) |
| |__  _ ___  ___ _   _ _| |_
| '_ \| / __|/ __| | | | | __|
| |_) | \__ \ (__| |_| | | |_
|_.__/|_|___/\___|\__,_|_|\__|`

func NewRootCommand() *cobra.Command {
	root := &cobra.Command{
		Use:     "biscuit",
		Short:   "Generate a production-ready CLI repository from an OpenAPI 3.x spec",
		Version: version.Version,
		// bare invocation only — cobra falls through to "unknown command" for typos
		Run: func(cmd *cobra.Command, _ []string) {
			if isTTY(cmd.OutOrStdout()) {
				printWelcome(cmd)
				return
			}
			_ = cmd.Help() // non-TTY keeps cobra's normal help output; scripts stay unsurprised
		},
	}
	// bare version for --version (script/agent-friendly); `biscuit version` has the detail
	root.SetVersionTemplate("{{.Version}}\n")
	root.SetFlagErrorFunc(func(_ *cobra.Command, err error) error {
		return &usageError{err}
	})
	root.AddCommand(newVersionCommand())
	root.AddCommand(newDoctorCommand())
	root.SilenceUsage = true // usage on errors drowns the actual failure; exit 2 already marks misuse
	return root
}

func printWelcome(cmd *cobra.Command) {
	out := cmd.OutOrStdout()
	_, _ = fmt.Fprintf(out, "\n%s%s%s\n\n", accent, logo, reset)
	_, _ = fmt.Fprintf(out, "%s%s%s\n\n", muted, cmd.Short, reset)
	_, _ = fmt.Fprintf(out, "%-16s %s# a dir with an OpenAPI spec%s\n", "cd <project>", muted, reset)
	_, _ = fmt.Fprintf(out, "%-16s %s# grade the spec%s\n\n", "biscuit doctor", muted, reset)
	_, _ = fmt.Fprintf(out, "%sFor more information visit %shttps://github.com/oxmonty/biscuit\n\n", muted, reset)
}

func isTTY(w io.Writer) bool {
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	stat, err := f.Stat()
	return err == nil && stat.Mode()&os.ModeCharDevice != 0
}

func newVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the biscuit version",
		Run: func(cmd *cobra.Command, _ []string) {
			// not cmd.Printf: cobra's Print helpers default to stderr
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "biscuit %s (commit %s, built %s)\n",
				version.Version, version.Commit, version.Date)
		},
	}
}
