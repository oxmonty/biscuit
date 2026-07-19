package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func write(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, FileName), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestLoadFullConfig(t *testing.T) {
	// given: a config using every schema section
	dir := write(t, `version: 1
spec:
  path: openapi.yaml
lint:
  min_grade: 85
operations:
  listUsers:
    name: ls
    group: admin users
    aliases: [list-all]
    pagination: cursor
  debugDump:
    ignore: true
`)

	// when: loading it
	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	// then: every field round-trips
	if cfg.Spec.Path != "openapi.yaml" || cfg.Lint.MinGrade != 85 {
		t.Errorf("cfg = %+v", cfg)
	}
	if op := cfg.Operations["listUsers"]; op.Name != "ls" || op.Group != "admin users" || op.Pagination != "cursor" {
		t.Errorf("listUsers = %+v", op)
	}
	if !cfg.Operations["debugDump"].Ignore {
		t.Error("debugDump.Ignore not set")
	}
}

func TestLoadMissingFileIsEmptyConfig(t *testing.T) {
	// given: a directory with no biscuit.yaml
	cfg, err := Load(t.TempDir())

	// then: an empty config, no error — the pre-discovery state
	if err != nil || cfg == nil {
		t.Fatalf("Load = %+v, %v", cfg, err)
	}
}

func TestLoadRejectsUnknownKeys(t *testing.T) {
	// given: a typo'd key
	dir := write(t, "lint:\n  min_grde: 85\n")

	// when/then: loading fails naming the unknown field
	_, err := Load(dir)
	if err == nil || !strings.Contains(err.Error(), "min_grde") {
		t.Errorf("err = %v, want unknown-field error naming min_grde", err)
	}
}

func TestLoadRejectsNewerVersion(t *testing.T) {
	// given: a config from a future biscuit
	dir := write(t, "version: 2\n")

	// when/then: loading refuses with an upgrade hint
	_, err := Load(dir)
	if err == nil || !strings.Contains(err.Error(), "upgrade biscuit") {
		t.Errorf("err = %v", err)
	}
}

func TestLoadAcceptsDiscoveryWrittenFile(t *testing.T) {
	// given: the minimal file spec discovery persists (no version key)
	dir := write(t, "spec:\n  path: openapi.yaml\n")

	// when/then: it loads as version 1
	cfg, err := Load(dir)
	if err != nil || cfg.Spec.Path != "openapi.yaml" {
		t.Errorf("cfg = %+v, err = %v", cfg, err)
	}
}
