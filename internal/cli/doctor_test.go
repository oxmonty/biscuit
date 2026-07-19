package cli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

const ladder = "../../testdata/specs/"

func TestDoctorTextReportFoldsCountsIntoImpact(t *testing.T) {
	// given: the doctor command against the easy ladder spec
	root := NewRootCommand()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetArgs([]string{"doctor", "--spec", ladder + "petstore.yaml"})

	// when: running it with output captured in a buffer (not a TTY)
	if err := root.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}

	// then: the impact line carries the count, and no separate "N×" prefix
	// or ANSI color escapes leak into piped output
	got := out.String()
	if !strings.Contains(got, "10 sites missing an example") {
		t.Errorf("output does not fold the finding count into the impact sentence:\n%s", got)
	}
	if strings.Contains(got, "10× oas3-missing-example") {
		t.Errorf("output still uses the old count-prefix style:\n%s", got)
	}
	if strings.Contains(got, "\x1b[") {
		t.Errorf("output has ANSI color codes when writing to a non-TTY buffer:\n%s", got)
	}
}

func TestDoctorHumanizesResolverDiagnostics(t *testing.T) {
	// given: the medium ladder spec, whose x-topics $ref points at a missing file
	root := NewRootCommand()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetArgs([]string{"doctor", "--spec", ladder + "train-travel.yaml"})

	// when: running doctor
	if err := root.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}

	// then: the resolver diagnostic reads as plain English, not a rolodex dump
	got := out.String()
	if !strings.Contains(got, "referenced but missing from the spec") && !strings.Contains(got, "file not found") {
		t.Errorf("resolver diagnostic was not humanized:\n%s", got)
	}
	if strings.Contains(got, "unable to open the rolodex file") {
		t.Errorf("raw rolodex diagnostic leaked into output:\n%s", got)
	}
}

func TestDoctorFormatJSON(t *testing.T) {
	// given: the doctor command with --format json
	root := NewRootCommand()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetArgs([]string{"doctor", "--spec", ladder + "petstore.yaml", "--format", "json"})

	// when: running it
	if err := root.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}

	// then: stdout is a single valid JSON report with the documented shape
	var report jsonReport
	if err := json.Unmarshal(out.Bytes(), &report); err != nil {
		t.Fatalf("output is not valid JSON: %v\n%s", err, out.String())
	}
	if report.Grade <= 0 {
		t.Errorf("Grade = %d, want > 0", report.Grade)
	}
	if report.Blocking {
		t.Error("Blocking = true, want false without --strict or lint.min_grade")
	}
	found := false
	for _, f := range report.Findings {
		if f.Rule == "oas3-missing-example" {
			found = true
			if f.Count != 10 {
				t.Errorf("Count = %d, want 10", f.Count)
			}
			if f.Impact == "" || f.Remediation == "" {
				t.Errorf("finding %q missing Impact/Remediation: %+v", f.Rule, f)
			}
		}
	}
	if !found {
		t.Error("json findings missing oas3-missing-example")
	}
}

func TestDoctorFormatJSONReflectsStrictGate(t *testing.T) {
	// given: --strict against a spec with advisory findings, in JSON format
	root := NewRootCommand()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetArgs([]string{"doctor", "--spec", ladder + "petstore.yaml", "--strict", "--format", "json"})

	// when: running it
	err := root.Execute()

	// then: the JSON reports blocking, and the exit-code contract is unaffected
	var report jsonReport
	if jsonErr := json.Unmarshal(out.Bytes(), &report); jsonErr != nil {
		t.Fatalf("output is not valid JSON: %v\n%s", jsonErr, out.String())
	}
	if !report.Blocking {
		t.Error("Blocking = false, want true under --strict with findings present")
	}
	if got := ExitCode(err); got != ExitQualityGate {
		t.Errorf("ExitCode = %d, want %d", got, ExitQualityGate)
	}
}

func TestDoctorRejectsUnknownFormat(t *testing.T) {
	// given: an unsupported --format value
	root := NewRootCommand()
	root.SetOut(&bytes.Buffer{})
	root.SetArgs([]string{"doctor", "--spec", ladder + "petstore.yaml", "--format", "xml"})

	// when: running it
	err := root.Execute()

	// then: it fails as a usage error, exit code 2
	if got := ExitCode(err); got != ExitUsage {
		t.Errorf("ExitCode = %d, want %d", got, ExitUsage)
	}
}

func TestHumanizeDiagnostic(t *testing.T) {
	cases := []struct {
		name string
		raw  string
		want string
	}{
		{
			name: "circular reference arrow",
			raw:  "circular reference: A -> B -> A",
			want: "circular reference: A → B → A",
		},
		{
			name: "missing component ref",
			raw:  "component `./docs/getting-started.md` does not exist in the specification",
			want: "docs/getting-started.md: referenced but missing from the spec",
		},
		{
			name: "unmatched line passes through",
			raw:  "some other diagnostic",
			want: "some other diagnostic",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given/when: humanizing the raw diagnostic
			got := humanizeDiagnostic(tc.raw)
			// then: it matches the expected plain-English line
			if got != tc.want {
				t.Errorf("humanizeDiagnostic(%q) = %q, want %q", tc.raw, got, tc.want)
			}
		})
	}
}
