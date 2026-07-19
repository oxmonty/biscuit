package spec

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/oxmonty/biscuit/internal/config"
)

func write(t *testing.T, dir, name, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestDiscoverRanksWellKnownFirst(t *testing.T) {
	// given: a dir with a conventional name and a sniffable spec
	dir := t.TempDir()
	write(t, dir, "openapi.yaml", "openapi: 3.1.0\n")
	write(t, dir, "api-def.yaml", "openapi: 3.0.3\n")
	write(t, dir, "values.yaml", "replicas: 3\n")

	// when: discovering
	got, err := DiscoverCandidates(dir)
	if err != nil {
		t.Fatal(err)
	}

	// then: the well-known name outranks the sniffed one; non-specs are absent
	want := []string{filepath.Join(dir, "openapi.yaml"), filepath.Join(dir, "api-def.yaml")}
	if len(got) != 2 || got[0] != want[0] || got[1] != want[1] {
		t.Errorf("candidates = %v, want %v", got, want)
	}
}

func TestDiscoverSniffsJSON(t *testing.T) {
	// given: only a JSON spec under an unconventional name
	dir := t.TempDir()
	write(t, dir, "svc.json", `{"openapi": "3.0.0"}`)

	got, err := DiscoverCandidates(dir)

	// then: the content sniff finds it
	if err != nil || len(got) != 1 || !strings.HasSuffix(got[0], "svc.json") {
		t.Errorf("got %v, %v", got, err)
	}
}

func TestDiscoverNothingFound(t *testing.T) {
	// given: a dir with no specs at all
	dir := t.TempDir()
	write(t, dir, "notes.txt", "openapi: not really\n")

	_, err := DiscoverCandidates(dir)

	// then: the sentinel maps to exit 3 at the CLI
	if !errors.Is(err, ErrNoSpecFound) {
		t.Errorf("err = %v, want ErrNoSpecFound", err)
	}
}

func TestCachedSpecPathRoundTrip(t *testing.T) {
	// given: a discovery result persisted into a fresh biscuit.yaml
	dir := t.TempDir()
	write(t, dir, "openapi.yaml", "openapi: 3.1.0\n")
	if err := PersistSpecPath(dir, filepath.Join(dir, "openapi.yaml")); err != nil {
		t.Fatal(err)
	}

	// then: the config loader returns it, so discovery runs once
	cfg, err := config.Load(dir)
	if err != nil || cfg.Spec.Path != "openapi.yaml" {
		t.Errorf("config = %+v, err = %v, want spec.path openapi.yaml", cfg, err)
	}
}

func TestPersistAppendsWithoutClobbering(t *testing.T) {
	// given: an existing biscuit.yaml with unrelated keys
	dir := t.TempDir()
	write(t, dir, "biscuit.yaml", "lint:\n  min_grade: 85\n")

	if err := PersistSpecPath(dir, filepath.Join(dir, "openapi.yaml")); err != nil {
		t.Fatal(err)
	}

	// then: the original keys survive and spec.path is readable
	data, _ := os.ReadFile(filepath.Join(dir, "biscuit.yaml"))
	if !strings.Contains(string(data), "min_grade: 85") {
		t.Error("existing config was clobbered")
	}
	cfg, err := config.Load(dir)
	if err != nil || cfg.Spec.Path != "openapi.yaml" || cfg.Lint.MinGrade != 85 {
		t.Errorf("config = %+v, err = %v after append", cfg, err)
	}
}
