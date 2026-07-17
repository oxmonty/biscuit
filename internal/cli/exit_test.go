package cli

import (
	"errors"
	"io/fs"
	"testing"

	"github.com/oxmonty/biscuit/internal/spec"
)

func TestExitCode(t *testing.T) {
	// given: one error of each contract category
	cases := []struct {
		name string
		err  error
		want int
	}{
		{"nil", nil, ExitOK},
		{"usage", &usageError{errors.New("unknown flag")}, ExitUsage},
		{"no spec", fs.ErrNotExist, ExitNoSpec},
		{"invalid spec", &spec.InvalidError{Path: "x", Problems: []string{"p"}}, ExitSpecInvalid},
		{"anything else", errors.New("boom"), ExitInternal},
	}

	for _, tc := range cases {
		// when/then: the mapping matches the contract
		if got := ExitCode(tc.err); got != tc.want {
			t.Errorf("%s: ExitCode = %d, want %d", tc.name, got, tc.want)
		}
	}
}
