package cli

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/oxmonty/biscuit/internal/config"
)

const idlessSpec = `openapi: 3.0.3
info: {title: Idless, version: 1.0.0}
paths:
  /users:
    get:
      responses:
        '200': {description: OK}
    post:
      responses:
        '201': {description: Created}
`

func runInit(t *testing.T, args ...string) (string, error) {
	t.Helper()
	root := NewRootCommand()
	var out, errOut bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&errOut)
	root.SetArgs(append([]string{"init"}, args...))
	err := root.Execute()
	return errOut.String(), err
}

func TestInitScaffoldsFromGapAnalysis(t *testing.T) {
	// given: a cwd with a spec whose operations have no operationIds
	dir := t.TempDir()
	t.Chdir(dir)
	if err := os.WriteFile("openapi.yaml", []byte(idlessSpec), 0o644); err != nil {
		t.Fatal(err)
	}

	// when: running biscuit init
	stderr, err := runInit(t, "--spec", "openapi.yaml")
	if err != nil {
		t.Fatalf("init: %v", err)
	}

	// then: the file exists, loads through the strict loader, and carries
	// commented override stubs keyed "METHOD /path" with the derived names
	cfg, err := config.Load(".")
	if err != nil || cfg.Spec.Path != "openapi.yaml" {
		t.Fatalf("config = %+v, err = %v", cfg, err)
	}
	data, _ := os.ReadFile(config.FileName)
	for _, want := range []string{`#   "GET /users":`, "#     name: list", `#   "POST /users":`, "#     name: create"} {
		if !strings.Contains(string(data), want) {
			t.Errorf("generated config missing %q:\n%s", want, data)
		}
	}
	if !strings.Contains(stderr, "2 override stubs") {
		t.Errorf("stderr = %q, want stub count", stderr)
	}
}

func TestInitRefusesToClobber(t *testing.T) {
	// given: a cwd that already has a biscuit.yaml
	dir := t.TempDir()
	t.Chdir(dir)
	if err := os.WriteFile(config.FileName, []byte("version: 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// when/then: init refuses with a usage error (exit 2)
	_, err := runInit(t)
	if err == nil || ExitCode(err) != ExitUsage {
		t.Errorf("err = %v (exit %d), want usage error", err, ExitCode(err))
	}
}
