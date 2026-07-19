package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/oxmonty/biscuit/internal/version"
)

func NewRootCommand() *cobra.Command {
	root := &cobra.Command{
		Use:     "biscuit",
		Short:   "Generate a production-ready CLI repository from an OpenAPI 3.x spec",
		Version: version.Version,
	}
	// bare version for --version (script/agent-friendly); `biscuit version` has the detail
	root.SetVersionTemplate("{{.Version}}\n")
	root.SetFlagErrorFunc(func(_ *cobra.Command, err error) error {
		return &usageError{err}
	})
	root.AddCommand(newVersionCommand())
	root.AddCommand(newDoctorCommand())
	root.AddCommand(newInitCommand())
	root.SilenceUsage = true // usage on errors drowns the actual failure; exit 2 already marks misuse
	return root
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
