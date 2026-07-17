package lint

import (
	"testing"

	"github.com/oxmonty/biscuit/internal/spec"
)

func TestRunGradesAndMapsImpact(t *testing.T) {
	// given: the easy ladder spec, which lacks examples and descriptions
	doc, err := spec.Load("../../testdata/specs/petstore.yaml")
	if err != nil {
		t.Fatal(err)
	}

	// when: grading it with the biscuit ruleset
	report := Run(doc)

	// then: it gets a real grade and generation-impact notes on mapped rules
	if report.Grade <= 0 || report.Grade >= 100 {
		t.Errorf("Grade = %d, want a mid-range score for petstore", report.Grade)
	}
	withImpact := 0
	for _, f := range report.Findings {
		if f.Impact != "" {
			withImpact++
		}
	}
	if withImpact == 0 {
		t.Error("no finding carries a generation-impact note")
	}
}
