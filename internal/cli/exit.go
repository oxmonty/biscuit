package cli

import (
	"errors"
	"io/fs"

	"github.com/oxmonty/biscuit/internal/spec"
)

// Biscuit's exit-code contract (PRD "Project structure"): scripts and the
// update pipeline branch on these, so they only ever grow — never renumber.
const (
	ExitOK          = 0
	ExitInternal    = 1
	ExitUsage       = 2
	ExitNoSpec      = 3
	ExitSpecInvalid = 4
	ExitQualityGate = 5
)

// usageError marks errors from flag/arg parsing so they map to ExitUsage.
type usageError struct{ err error }

func (u *usageError) Error() string { return u.err.Error() }
func (u *usageError) Unwrap() error { return u.err }

// ExitCode maps an error from Execute to the contract above.
func ExitCode(err error) int {
	var invalid *spec.InvalidError
	var usage *usageError
	var gate *qualityGateError
	switch {
	case err == nil:
		return ExitOK
	case errors.As(err, &usage):
		return ExitUsage
	case errors.Is(err, fs.ErrNotExist):
		return ExitNoSpec
	case errors.As(err, &invalid):
		return ExitSpecInvalid
	case errors.As(err, &gate):
		return ExitQualityGate
	default:
		return ExitInternal
	}
}
