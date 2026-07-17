package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/oxmonty/biscuit/internal/spec"
)

// resolveSpecPath returns the spec to operate on: the --spec flag, then
// biscuit.yaml's spec.path (the cache), then cwd discovery. A discovered
// choice is persisted so discovery runs once.
func resolveSpecPath(cmd *cobra.Command, flag string) (string, error) {
	if flag != "" {
		return flag, nil
	}
	if cached := spec.CachedSpecPath("."); cached != "" {
		return cached, nil
	}

	candidates, err := spec.DiscoverCandidates(".")
	if err != nil {
		return "", err
	}

	choice := candidates[0]
	if len(candidates) > 1 {
		choice, err = pickCandidate(cmd, candidates)
		if err != nil {
			return "", err
		}
	}
	fmt.Fprintf(cmd.ErrOrStderr(), "using spec %s (recorded in biscuit.yaml)\n", choice)
	if err := spec.PersistSpecPath(".", choice); err != nil {
		return "", err
	}
	return choice, nil
}

// pickCandidate prompts on a TTY (plain numbered stderr list — the Bubble Tea
// selector arrives with the chat TUI epic); otherwise it takes the best-ranked
// candidate and says so.
func pickCandidate(cmd *cobra.Command, candidates []string) (string, error) {
	stat, err := os.Stdin.Stat()
	interactive := err == nil && stat.Mode()&os.ModeCharDevice != 0
	if !interactive {
		return candidates[0], nil
	}

	fmt.Fprintln(cmd.ErrOrStderr(), "multiple specs found:")
	for i, c := range candidates {
		fmt.Fprintf(cmd.ErrOrStderr(), "  %d) %s\n", i+1, c)
	}
	fmt.Fprintf(cmd.ErrOrStderr(), "which one? [1]: ")

	var answer int
	if _, err := fmt.Fscanln(cmd.InOrStdin(), &answer); err != nil || answer < 1 || answer > len(candidates) {
		answer = 1
	}
	return candidates[answer-1], nil
}
