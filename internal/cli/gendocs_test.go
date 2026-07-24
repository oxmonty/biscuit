package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra/doc"
	"go.yaml.in/yaml/v4"
)

// goreleaserConfig mirrors only the bits of .goreleaser.yml this test checks.
type goreleaserConfig struct {
	HomebrewCasks []struct {
		Manpages []string `yaml:"manpages"`
	} `yaml:"homebrew_casks"`
}

func TestGenManTreeMatchesCommands(t *testing.T) {
	// given: the man tree generated for the real root command
	root := NewRootCommand()
	dir := t.TempDir()
	if err := doc.GenManTree(root, &doc.GenManHeader{Title: "BISCUIT", Section: "1"}, dir); err != nil {
		t.Fatalf("GenManTree: %v", err)
	}

	// then: biscuit.1 exists for the root command itself
	if _, err := os.Stat(filepath.Join(dir, "biscuit.1")); err != nil {
		t.Errorf("missing biscuit.1: %v", err)
	}

	// then: every visible top-level subcommand has a matching man page
	// (same filter cobra's own doc.GenManTree applies internally)
	for _, sub := range root.Commands() {
		if !sub.IsAvailableCommand() || sub.IsAdditionalHelpTopicCommand() {
			continue
		}
		want := "biscuit-" + sub.Name() + ".1"
		if _, err := os.Stat(filepath.Join(dir, want)); err != nil {
			t.Errorf("missing man page for %q: %v", sub.Name(), err)
		}
	}

	// when: parsing .goreleaser.yml's cask manpage lists
	repoRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}
	raw, err := os.ReadFile(filepath.Join(repoRoot, ".goreleaser.yml"))
	if err != nil {
		t.Fatalf("read .goreleaser.yml: %v", err)
	}
	var cfg goreleaserConfig
	if err := yaml.Unmarshal(raw, &cfg); err != nil {
		t.Fatalf("parse .goreleaser.yml: %v", err)
	}
	if len(cfg.HomebrewCasks) == 0 {
		t.Fatal(".goreleaser.yml: no homebrew_casks entries found")
	}

	// then: the cask lists exactly the top-level man pages — a removed
	// command left dangling and a new command missing from the list both fail
	wanted := map[string]bool{"biscuit.1": true}
	for _, sub := range root.Commands() {
		if !sub.IsAvailableCommand() || sub.IsAdditionalHelpTopicCommand() {
			continue
		}
		wanted["biscuit-"+sub.Name()+".1"] = true
	}
	for _, cask := range cfg.HomebrewCasks {
		listed := map[string]bool{}
		for _, entry := range cask.Manpages {
			name := filepath.Base(entry)
			listed[name] = true
			if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
				t.Errorf("cask references man page %q but it was not generated: %v", entry, err)
			}
		}
		for name := range wanted {
			if !listed[name] {
				t.Errorf("cask manpages list is missing %q — update .goreleaser.yml", name)
			}
		}
	}

	// then: every generated man page is declared in every cask (catches a new
	// command added without updating the cask manpage list)
	generated, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read generated man dir: %v", err)
	}
	for _, cask := range cfg.HomebrewCasks {
		declared := make(map[string]bool, len(cask.Manpages))
		for _, entry := range cask.Manpages {
			declared[filepath.Base(entry)] = true
		}
		for _, f := range generated {
			if !declared[f.Name()] {
				t.Errorf("generated man page %q is not declared in a homebrew_casks manpages list", f.Name())
			}
		}
	}
}
