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

func TestBareInvocationNonTTYFallsThroughToHelp(t *testing.T) {
	// given: the root command with a non-TTY output (a bytes.Buffer, not *os.File)
	root := NewRootCommand()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetArgs([]string{})

	// when: running bare `biscuit`
	if err := root.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}

	// then: it's cobra's normal help, not the welcome splash
	if strings.Contains(out.String(), logo) {
		t.Error("non-TTY invocation printed the splash logo")
	}
	if !strings.Contains(out.String(), "Usage:") {
		t.Errorf("expected cobra help output, got: %q", out.String())
	}
}
