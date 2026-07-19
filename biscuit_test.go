package biscuit

import (
	"context"
	"os"
	"path/filepath"
	"testing"
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

	// when: generating and writing the plan
	plan, err := Generate(context.Background(), doc, cfg)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	dir := t.TempDir()
	if err := plan.Write(dir); err != nil {
		t.Fatalf("Write: %v", err)
	}

	// then: the plan writes exactly its files — none yet, so an empty dir
	entries, _ := os.ReadDir(dir)
	if len(entries) != len(plan.Files) {
		t.Errorf("wrote %d entries for %d planned files", len(entries), len(plan.Files))
	}
}

func TestFilePlanWriteCreatesDirectories(t *testing.T) {
	// given: a plan with a nested file
	plan := &FilePlan{Files: []PlannedFile{{Path: "cmd/app/main.go", Contents: []byte("package main\n")}}}
	dir := t.TempDir()

	// when: writing it
	if err := plan.Write(dir); err != nil {
		t.Fatalf("Write: %v", err)
	}

	// then: the nested path exists with the contents
	data, err := os.ReadFile(filepath.Join(dir, "cmd", "app", "main.go"))
	if err != nil || string(data) != "package main\n" {
		t.Errorf("read = %q, %v", data, err)
	}
}
