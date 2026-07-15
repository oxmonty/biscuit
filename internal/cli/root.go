package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/monthy-app/biscuit/internal/version"
)

func NewRootCommand() *cobra.Command {
	root := &cobra.Command{
		Use:     "biscuit",
		Short:   "Generate a production-ready CLI repository from an OpenAPI 3.x spec",
		Version: version.Version,
	}
	root.AddCommand(newVersionCommand())
	return root
}

func newVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the biscuit version",
		Run: func(cmd *cobra.Command, _ []string) {
			fmt.Fprintf(cmd.OutOrStdout(), "biscuit %s (commit %s, built %s)\n",
				version.Version, version.Commit, version.Date)
		},
	}
}
