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

func TestBareInvocationShowsHelp(t *testing.T) {
	// given: the root command with no subcommand
	root := NewRootCommand()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetArgs([]string{})

	// when: running bare `biscuit`
	if err := root.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}

	// then: cobra's default help fires, including the quickstart
	if !strings.Contains(out.String(), "Usage:") {
		t.Errorf("expected cobra help output, got: %q", out.String())
	}
	if !strings.Contains(out.String(), "biscuit doctor") {
		t.Error("bare invocation help is missing the quickstart")
	}
}

func TestHelpCommandShowsQuickstart(t *testing.T) {
	// given: the root command
	root := NewRootCommand()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetArgs([]string{"help"})

	// when: running `biscuit help`
	if err := root.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}

	// then: the quickstart appears alongside the usual command listing
	if !strings.Contains(out.String(), "cd <project>") {
		t.Error("biscuit help is missing the quickstart")
	}
	if !strings.Contains(out.String(), "Available Commands:") {
		t.Error("biscuit help lost cobra's command listing")
	}
}
