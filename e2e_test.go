package biscuit

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// The feature end to end, through its real surfaces: the biscuit CLI binary
// generates a repo, the repo builds, and the generated binary makes a correct
// HTTP request against a live server. Everything else in the suite pins a
// layer; this pins the seams between them.
func TestEndToEndGeneratedCLIMakesRequests(t *testing.T) {
	if testing.Short() {
		t.Skip("e2e skipped in -short")
	}

	// given: a real biscuit binary and an echo server
	work := t.TempDir()
	biscuitBin := filepath.Join(work, "biscuit")
	build := exec.Command("go", "build", "-o", biscuitBin, "./cmd/biscuit")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("building biscuit: %v\n%s", err, out)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"method": r.Method, "path": r.URL.Path, "query": r.URL.RawQuery,
		})
	}))
	defer srv.Close()

	// when: generating petstore-cli through the actual CLI
	spec, err := filepath.Abs("testdata/specs/petstore.yaml")
	if err != nil {
		t.Fatal(err)
	}
	outDir := filepath.Join(work, "petstore-cli")
	gen := exec.Command(biscuitBin, "generate", "--spec", spec, "--out", outDir, "--quiet")
	gen.Dir = work
	if out, err := gen.CombinedOutput(); err != nil {
		t.Fatalf("biscuit generate: %v\n%s", err, out)
	}

	// then: the generated repo builds with no tidy step
	cliBin := filepath.Join(work, "petstore")
	if runtime.GOOS == "windows" {
		cliBin += ".exe"
	}
	buildCLI := exec.Command("go", "build", "-o", cliBin, "./cmd/swagger-petstore")
	buildCLI.Dir = outDir
	if out, err := buildCLI.CombinedOutput(); err != nil {
		t.Fatalf("building generated CLI: %v\n%s", err, out)
	}

	// then: a generated command sends the right request on the wire —
	// query param from a typed flag, response body relayed to stdout
	list := exec.Command(cliBin, "pets", "list", "--limit", "5", "--base-url", srv.URL)
	out, err := list.CombinedOutput()
	if err != nil {
		t.Fatalf("pets list: %v\n%s", err, out)
	}
	var echoed struct{ Method, Path, Query string }
	if err := json.Unmarshal(out, &echoed); err != nil {
		t.Fatalf("stdout is not the relayed response body: %q", out)
	}
	if echoed.Method != "GET" || echoed.Path != "/pets" || echoed.Query != "limit=5" {
		t.Errorf("wire request = %+v, want GET /pets limit=5", echoed)
	}

	// then: a path parameter substitutes into the URL
	show := exec.Command(cliBin, "pets", "show", "--pet-id", "42", "--base-url", srv.URL)
	out, err = show.CombinedOutput()
	if err != nil {
		t.Fatalf("pets show: %v\n%s", err, out)
	}
	if err := json.Unmarshal(out, &echoed); err != nil {
		t.Fatalf("stdout not JSON: %q", out)
	}
	if echoed.Path != "/pets/42" {
		t.Errorf("path substitution = %q, want /pets/42", echoed.Path)
	}

	// then: a missing required flag fails without touching the network
	missing := exec.Command(cliBin, "pets", "show", "--base-url", srv.URL)
	out, err = missing.CombinedOutput()
	if err == nil {
		t.Errorf("missing required flag succeeded:\n%s", out)
	}
	if !strings.Contains(string(out), "pet-id") {
		t.Errorf("error does not name the missing flag:\n%s", out)
	}

	// then: help surfaces show the quickstart on bare invocation
	bare := exec.Command(cliBin)
	out, _ = bare.CombinedOutput()
	if !strings.Contains(string(out), "Quickstart:") {
		t.Errorf("bare invocation missing quickstart:\n%s", out)
	}
}

// install.sh's offline suite rides go test so CI's single command runs it —
// the shell script owns the assertions.
func TestInstallScriptOfflineSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("skipped in -short")
	}
	if runtime.GOOS == "windows" {
		t.Skip("bash-based installer test")
	}
	cmd := exec.Command("sh", "scripts/test_install.sh")
	cmd.Env = append(os.Environ(), "HOME="+t.TempDir())
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Errorf("install.sh suite failed: %v\n%s", err, out)
	}
}
