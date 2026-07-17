package spec

import (
	"errors"
	"io/fs"
	"strings"
	"testing"
)

const ladder = "../../testdata/specs/"

func TestLoadCleanSpec(t *testing.T) {
	// given: the easy ladder spec (petstore, OpenAPI 3.0)
	// when: loading it
	doc, err := Load(ladder + "petstore.yaml")

	// then: it loads with no problems and the model is populated
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got := doc.Operations(); got != 3 {
		t.Errorf("Operations() = %d, want 3", got)
	}
	if len(doc.Diagnostics) != 0 {
		t.Errorf("Diagnostics = %v, want none", doc.Diagnostics)
	}
}

func TestLoadSpecWithExtensionRef(t *testing.T) {
	// given: the medium spec (train-travel, 3.1) whose x-topics $refs a missing markdown file
	doc, err := Load(ladder + "train-travel.yaml")

	// then: the unresolvable extension ref is advisory, never blocking
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got := doc.Operations(); got != 7 {
		t.Errorf("Operations() = %d, want 7 (webhooks counted separately)", got)
	}
	if len(doc.Diagnostics) == 0 {
		t.Error("want a diagnostic for the unresolvable x-topics ref, got none")
	}
}

func TestLoadHardSpec(t *testing.T) {
	// given: the hard ladder spec (openai.yaml, 2.8 MB)
	doc, err := Load(ladder + "openai.yaml")

	// then: it loads and the full operation surface is present
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got := doc.Operations(); got != 281 {
		t.Errorf("Operations() = %d, want 281", got)
	}
}

func TestLoadCyclicRefsIsSafe(t *testing.T) {
	// given: a spec with A→B→A and self-referencing schemas
	doc, err := Load(ladder + "pathological/cyclic-refs.yaml")

	// then: loading terminates, succeeds, and reports the cycles as diagnostics
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	found := false
	for _, d := range doc.Diagnostics {
		if strings.Contains(d, "circular") {
			found = true
		}
	}
	if !found {
		t.Errorf("Diagnostics = %v, want a circular-reference entry", doc.Diagnostics)
	}
}

func TestLoadUnresolvableRefIsBlocking(t *testing.T) {
	// given: a spec whose operations $ref schemas that do not exist
	_, err := Load(ladder + "pathological/unresolvable-ref.yaml")

	// then: Load fails with InvalidError naming the missing references
	var invalid *InvalidError
	if !errors.As(err, &invalid) {
		t.Fatalf("Load err = %v, want *InvalidError", err)
	}
	if len(invalid.Problems) == 0 {
		t.Error("InvalidError.Problems is empty")
	}
}

func TestLoadDuplicateOperationIDsIsBlocking(t *testing.T) {
	// given: two operations sharing one operationId
	_, err := Load(ladder + "pathological/duplicate-operation-ids.yaml")

	// then: Load fails with InvalidError citing the duplicate
	var invalid *InvalidError
	if !errors.As(err, &invalid) {
		t.Fatalf("Load err = %v, want *InvalidError", err)
	}
	if !strings.Contains(invalid.Error(), "duplicate operationId") {
		t.Errorf("error %q does not cite the duplicate operationId", invalid.Error())
	}
}

func TestLoadMissingFile(t *testing.T) {
	// given: a path that does not exist
	_, err := Load(ladder + "nope.yaml")

	// then: the fs error type survives so the CLI can map it to exit 3
	if !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("err = %v, want fs.ErrNotExist", err)
	}
}
