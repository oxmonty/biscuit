package cli

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/oxmonty/biscuit/internal/config"
)

func runGenerate(t *testing.T, args ...string) (stdout, stderr string, err error) {
	t.Helper()
	root := NewRootCommand()
	var out, errOut bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&errOut)
	root.SetArgs(append([]string{"generate"}, args...))
	err = root.Execute()
	return out.String(), errOut.String(), err
}

func TestGenerateDryRunPrintsTreeAndPlan(t *testing.T) {
	// given: the easy ladder spec
	stdout, stderr, err := runGenerate(t, "--spec", ladder+"petstore.yaml", "--dry-run")
	if err != nil {
		t.Fatalf("generate: %v", err)
	}

	// then: the tree, the counts header, and the empty file plan print
	for _, want := range []string{"3 operations", "pets", "list", "show", "file plan: 0 files"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("stdout missing %q:\n%s", want, stdout)
		}
	}
	// then: the implicit doctor summary lands on stderr
	if !strings.Contains(stderr, "advisory findings") {
		t.Errorf("stderr = %q, want advisory summary", stderr)
	}
}

func TestGenerateDryRunShowsFlags(t *testing.T) {
	// given: --flags on the medium ladder spec
	stdout, _, err := runGenerate(t, "--spec", ladder+"train-travel.yaml", "--dry-run", "--flags")
	if err != nil {
		t.Fatalf("generate: %v", err)
	}

	// then: derived flags print with their types
	if !strings.Contains(stdout, "--") || !strings.Contains(stdout, "string") {
		t.Errorf("stdout has no flag lines:\n%s", stdout)
	}
}

func TestGenerateWithoutDryRunIsUsageError(t *testing.T) {
	// given: generate without --dry-run before rendering exists
	_, _, err := runGenerate(t, "--spec", ladder+"petstore.yaml")

	// then: a usage error points at --dry-run
	if err == nil || ExitCode(err) != ExitUsage {
		t.Errorf("err = %v (exit %d), want usage", err, ExitCode(err))
	}
}

func TestGenerateMinGradeGate(t *testing.T) {
	// given: a config gating on a grade petstore can't reach
	spec, err := os.ReadFile(ladder + "petstore.yaml")
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	t.Chdir(dir)
	if err := os.WriteFile("openapi.yaml", spec, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(config.FileName, []byte("spec:\n  path: openapi.yaml\nlint:\n  min_grade: 100\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// when/then: the quality gate fails with exit 5
	_, _, err = runGenerate(t, "--dry-run")
	if err == nil || ExitCode(err) != ExitQualityGate {
		t.Errorf("err = %v (exit %d), want quality gate", err, ExitCode(err))
	}
}

func TestGenerateAppliesConfigOverrides(t *testing.T) {
	// given: a config renaming and hiding operations (the rescue demo shape)
	dir := t.TempDir()
	t.Chdir(dir)
	specYAML := `openapi: 3.0.3
info: {title: Rescue, version: 1.0.0}
paths:
  /users:
    get:
      responses:
        '200': {description: OK}
  /internal/dump:
    get:
      operationId: debugDump
      responses:
        '200': {description: OK}
`
	if err := os.WriteFile("openapi.yaml", []byte(specYAML), 0o644); err != nil {
		t.Fatal(err)
	}
	cfgYAML := `version: 1
spec:
  path: openapi.yaml
operations:
  "GET /users":
    name: everyone
  debugDump:
    ignore: true
`
	if err := os.WriteFile(config.FileName, []byte(cfgYAML), 0o644); err != nil {
		t.Fatal(err)
	}

	// when: dry-running
	stdout, _, err := runGenerate(t, "--dry-run")
	if err != nil {
		t.Fatalf("generate: %v", err)
	}

	// then: the rename shows and the ignored operation is gone
	if !strings.Contains(stdout, "everyone") {
		t.Errorf("rename not applied:\n%s", stdout)
	}
	if strings.Contains(stdout, "dump") {
		t.Errorf("ignored operation still present:\n%s", stdout)
	}
}
