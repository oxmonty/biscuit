package biscuit

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/oxmonty/biscuit/internal/render"
)

func TestGenerateIsPureAndWriteIsSeparate(t *testing.T) {
	// given: a loaded ladder spec and a config with an override
	doc, err := Load("testdata/specs/petstore.yaml")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	cfg := &Config{Operations: map[string]Operation{
		"listPets": {Name: "ls"},
	}}

	// when: generating twice and writing the plan once
	plan, err := Generate(context.Background(), doc, cfg)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	again, err := Generate(context.Background(), doc, cfg)
	if err != nil {
		t.Fatalf("Generate again: %v", err)
	}
	dir := t.TempDir()
	res, err := plan.Write(dir)
	if err != nil {
		t.Fatalf("Write: %v", err)
	}

	// then: the plan is non-empty, byte-deterministic, and fully written
	if len(plan.Files) == 0 {
		t.Fatal("empty file plan")
	}
	if len(again.Files) != len(plan.Files) {
		t.Fatalf("second Generate planned %d files, first %d", len(again.Files), len(plan.Files))
	}
	for i := range plan.Files {
		if plan.Files[i].Path != again.Files[i].Path || !bytes.Equal(plan.Files[i].Contents, again.Files[i].Contents) {
			t.Errorf("nondeterministic output at %s", plan.Files[i].Path)
		}
	}
	if len(res.Written) != len(plan.Files) {
		t.Errorf("wrote %d of %d planned files", len(res.Written), len(plan.Files))
	}
	if _, err := os.Stat(filepath.Join(dir, "go.mod")); err != nil {
		t.Errorf("go.mod not written: %v", err)
	}
}

func TestFilePlanWriteCreatesDirectories(t *testing.T) {
	// given: a plan with a nested file
	plan := &FilePlan{Files: []PlannedFile{{Path: "cmd/app/main.go", Contents: []byte("package main\n")}}}
	dir := t.TempDir()

	// when: writing it
	if _, err := plan.Write(dir); err != nil {
		t.Fatalf("Write: %v", err)
	}

	// then: the nested path exists with the contents
	data, err := os.ReadFile(filepath.Join(dir, "cmd", "app", "main.go"))
	if err != nil || string(data) != "package main\n" {
		t.Errorf("read = %q, %v", data, err)
	}
}

func TestRegenerationSafety(t *testing.T) {
	// given: a written petstore plan with a user-edited custom file and a
	// generated file whose marker the user stripped (taking ownership)
	doc, err := Load("testdata/specs/petstore.yaml")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	plan, err := Generate(context.Background(), doc, &Config{})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	dir := t.TempDir()
	if _, err := plan.Write(dir); err != nil {
		t.Fatalf("first Write: %v", err)
	}
	customPath := filepath.Join(dir, "internal", "custom", "custom.go")
	if err := os.WriteFile(customPath, []byte("package custom // mine\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	ownedPath := filepath.Join(dir, "internal", "iostreams", "iostreams.go")
	if err := os.WriteFile(ownedPath, []byte("package iostreams // marker stripped\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// when: regenerating over the same directory
	res, err := plan.Write(dir)
	if err != nil {
		t.Fatalf("second Write: %v", err)
	}

	// then: custom/ and the marker-stripped file survive byte-for-byte, and
	// the release-please manifest (bot-owned after the first release) is
	// emit-once too
	if got, _ := os.ReadFile(customPath); string(got) != "package custom // mine\n" {
		t.Errorf("custom file overwritten: %q", got)
	}
	if got, _ := os.ReadFile(ownedPath); string(got) != "package iostreams // marker stripped\n" {
		t.Errorf("user-owned file overwritten: %q", got)
	}
	wantOnce := []string{".release-please-manifest.json", "internal/custom/custom.go"}
	if len(res.SkippedCustom) != len(wantOnce) || res.SkippedCustom[0] != wantOnce[0] || res.SkippedCustom[1] != wantOnce[1] {
		t.Errorf("emit-once skips = %v, want %v", res.SkippedCustom, wantOnce)
	}
	if len(res.SkippedUnmarked) != 1 {
		t.Errorf("unmarked skips = %v", res.SkippedUnmarked)
	}
}

func TestMachineOwnedFilesAlwaysOverwrite(t *testing.T) {
	// given: a written plan whose go.sum (no comment syntax, so no marker)
	// was modified on disk
	doc, err := Load("testdata/specs/petstore.yaml")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	plan, err := Generate(context.Background(), doc, &Config{})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	dir := t.TempDir()
	if _, err := plan.Write(dir); err != nil {
		t.Fatalf("first Write: %v", err)
	}
	sumPath := filepath.Join(dir, "go.sum")
	if err := os.WriteFile(sumPath, []byte("stale entries\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// when: regenerating
	if _, err := plan.Write(dir); err != nil {
		t.Fatalf("second Write: %v", err)
	}

	// then: go.sum is machine-owned — both sides unmarked — so it is
	// restored, never treated as user-owned
	got, err := os.ReadFile(sumPath)
	if err != nil || string(got) == "stale entries\n" {
		t.Errorf("go.sum not restored on regeneration: %q, %v", firstBytes(got), err)
	}
}

func firstBytes(b []byte) string {
	if len(b) > 40 {
		b = b[:40]
	}
	return string(b)
}

func TestEveryGeneratedFileCarriesTheMarker(t *testing.T) {
	// given: a generated petstore plan
	doc, err := Load("testdata/specs/petstore.yaml")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	plan, err := Generate(context.Background(), doc, &Config{})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	// then: every file carries the marker except emit-once files (user-owned
	// after first emission) and files with no comment syntax to carry it
	// (go.sum, and JSON has none at all)
	noMarkerSyntax := map[string]bool{
		"go.sum":                        true,
		"release-please-config.json":    true,
		".release-please-manifest.json": true,
	}
	for _, f := range plan.Files {
		marked := bytes.Contains(f.Contents, []byte(render.Marker))
		switch {
		case f.EmitOnce && marked:
			t.Errorf("%s: emit-once file must not claim DO NOT EDIT", f.Path)
		case !f.EmitOnce && !noMarkerSyntax[f.Path] && !marked:
			t.Errorf("%s: missing generated-file marker", f.Path)
		}
	}
}
