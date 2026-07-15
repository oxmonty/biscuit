package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestVersionCommand(t *testing.T) {
	// given: the root command with output captured
	root := NewRootCommand()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetArgs([]string{"version"})

	// when: running `biscuit version`
	if err := root.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}

	// then: the output reports the version
	if !strings.HasPrefix(out.String(), "biscuit dev") {
		t.Errorf("unexpected version output: %q", out.String())
	}
}
