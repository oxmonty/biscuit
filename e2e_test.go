package biscuit

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
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
		body, _ := io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"method": r.Method, "path": r.URL.Path, "query": r.URL.RawQuery, "body": string(body),
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

	// then: the fresh repo passes its own test suite — the generated smoke
	// cases driving every command against the spec-derived mock it ships,
	// which is what a user gets from `go test ./...` on day one
	suite := exec.Command("go", "test", "./...")
	suite.Dir = outDir
	if out, err := suite.CombinedOutput(); err != nil {
		t.Fatalf("generated repo's own go test ./... failed: %v\n%s", err, out)
	}

	// then: a generated command sends the right request on the wire —
	// query param from a typed flag, response body relayed to stdout
	list := exec.Command(cliBin, "pets", "list", "--limit", "5", "--base-url", srv.URL)
	out, err := list.CombinedOutput()
	if err != nil {
		t.Fatalf("pets list: %v\n%s", err, out)
	}
	var echoed struct{ Method, Path, Query, Body string }
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

	// then: --format jsonl on a non-array JSON body (the echo server always
	// returns a JSON object) prints a single compact line, not one line per
	// top-level key
	jsonl := exec.Command(cliBin, "pets", "list", "--format", "jsonl", "--base-url", srv.URL)
	out, err = jsonl.CombinedOutput()
	if err != nil {
		t.Fatalf("pets list --format jsonl: %v\n%s", err, out)
	}
	if lines := strings.Split(strings.TrimRight(string(out), "\n"), "\n"); len(lines) != 1 {
		t.Errorf("--format jsonl on a JSON object = %d lines, want 1:\n%s", len(lines), out)
	}

	// then: --transform extracts one field via a GJSON path, still printed
	// through the format pipeline
	transform := exec.Command(cliBin, "pets", "list", "--transform", "path", "--base-url", srv.URL)
	out, err = transform.CombinedOutput()
	if err != nil {
		t.Fatalf("pets list --transform path: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "/pets") {
		t.Errorf("--transform path = %q, want it to contain /pets", out)
	}

	// then: --include-headers prints the response status/headers before the
	// body, on every format
	headers := exec.Command(cliBin, "pets", "list", "--include-headers", "--base-url", srv.URL)
	out, err = headers.CombinedOutput()
	if err != nil {
		t.Fatalf("pets list --include-headers: %v\n%s", err, out)
	}
	headerIdx := strings.Index(string(out), "Content-Type:")
	bodyIdx := strings.Index(string(out), "{")
	if headerIdx < 0 {
		t.Errorf("--include-headers missing a header line:\n%s", out)
	} else if bodyIdx < 0 || headerIdx > bodyIdx {
		t.Errorf("--include-headers header must precede the body:\n%s", out)
	}

	// then: a missing required flag fails without touching the network,
	// exiting with the documented usage-error code
	missing := exec.Command(cliBin, "pets", "show", "--base-url", srv.URL)
	out, err = missing.CombinedOutput()
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("missing required flag: want *exec.ExitError, got %v\n%s", err, out)
	}
	if exitErr.ExitCode() != 2 {
		t.Errorf("missing required flag exit code = %d, want 2 (usage)\n%s", exitErr.ExitCode(), out)
	}
	if !strings.Contains(string(out), "pet-id") {
		t.Errorf("error does not name the missing flag:\n%s", out)
	}

	// then: --debug logs the request with the secret-shaped --header
	// redacted, never the raw value
	debug := exec.Command(cliBin, "pets", "list", "--debug", "--header", "X-Api-Key: sekrit", "--base-url", srv.URL)
	out, err = debug.CombinedOutput()
	if err != nil {
		t.Fatalf("pets list --debug: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "X-Api-Key: [REDACTED]") {
		t.Errorf("--debug output missing redacted header:\n%s", out)
	}
	if strings.Contains(string(out), "sekrit") {
		t.Errorf("--debug output leaked the secret header value:\n%s", out)
	}

	// then: help surfaces show the quickstart on bare invocation
	bare := exec.Command(cliBin)
	out, _ = bare.CombinedOutput()
	if !strings.Contains(string(out), "Quickstart:") {
		t.Errorf("bare invocation missing quickstart:\n%s", out)
	}

	// then: @<tmpfile> on a string body flag reads the file and lands its
	// content verbatim as the wire value — the flagship @file use
	tmpfile := filepath.Join(work, "name.txt")
	if err := os.WriteFile(tmpfile, []byte(`{"nested":"value"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	atFile := exec.Command(cliBin, "pets", "create", "--id", "1", "--name", "@"+tmpfile, "--base-url", srv.URL)
	out, err = atFile.CombinedOutput()
	if err != nil {
		t.Fatalf("pets create --name @file: %v\n%s", err, out)
	}
	if err := json.Unmarshal(out, &echoed); err != nil {
		t.Fatalf("stdout not JSON: %q", out)
	}
	if name := wireName(t, echoed.Body); name != `{"nested":"value"}` {
		t.Errorf("wire name = %q, want file content verbatim", name)
	}

	// then: a leading \@ escapes to a literal @, never read as a file
	escaped := exec.Command(cliBin, "pets", "create", "--id", "1", "--name", `\@literal`, "--base-url", srv.URL)
	out, err = escaped.CombinedOutput()
	if err != nil {
		t.Fatalf(`pets create --name \@literal: %v\n%s`, err, out)
	}
	if err := json.Unmarshal(out, &echoed); err != nil {
		t.Fatalf("stdout not JSON: %q", out)
	}
	if name := wireName(t, echoed.Body); name != "@literal" {
		t.Errorf(`wire name = %q, want "@literal"`, name)
	}

	// then: @data:// forces a literal with no file read, even when the
	// literal itself starts with @
	dataLit := exec.Command(cliBin, "pets", "create", "--id", "1", "--name", "@data://@weird", "--base-url", srv.URL)
	out, err = dataLit.CombinedOutput()
	if err != nil {
		t.Fatalf("pets create --name @data://@weird: %v\n%s", err, out)
	}
	if err := json.Unmarshal(out, &echoed); err != nil {
		t.Fatalf("stdout not JSON: %q", out)
	}
	if name := wireName(t, echoed.Body); name != "@weird" {
		t.Errorf(`wire name = %q, want "@weird"`, name)
	}
}

// wireName decodes body (the raw JSON request body the echo server relayed
// back as a string) and returns its "name" field, sidestepping the double
// JSON-escaping a raw string comparison would otherwise require.
func wireName(t *testing.T, body string) string {
	t.Helper()
	var v struct{ Name string }
	if err := json.Unmarshal([]byte(body), &v); err != nil {
		t.Fatalf("wire body not JSON: %q", body)
	}
	return v.Name
}

// TestEndToEndGalaxyAuth pins the auth story end to end: galaxy.yaml is the
// multi-securityScheme spec (petstore.yaml declares none), so this generates
// galaxy-cli separately to prove a credential resolves from its env var onto
// the wire, and that an operation requiring auth refuses to dial out at all
// when no credential is configured.
func TestEndToEndGalaxyAuth(t *testing.T) {
	if testing.Short() {
		t.Skip("e2e skipped in -short")
	}

	// given: a real biscuit binary and an echo server that also reports
	// the headers it received
	work := t.TempDir()
	biscuitBin := filepath.Join(work, "biscuit")
	build := exec.Command("go", "build", "-o", biscuitBin, "./cmd/biscuit")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("building biscuit: %v\n%s", err, out)
	}
	var requests int
	var uploadedBytes []byte
	var uploadedContentType string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data") {
			// then (parsed server-side, not trusting the client's own
			// claims): the multipart body carries the file's raw bytes and
			// a sniffed per-part Content-Type, not a JSON-encoded string
			if err := r.ParseMultipartForm(1 << 20); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			file, header, err := r.FormFile("image")
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			defer func() { _ = file.Close() }()
			uploadedBytes, _ = io.ReadAll(file)
			uploadedContentType = header.Header.Get("Content-Type")
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]string{"message": "ok"})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"method": r.Method, "path": r.URL.Path, "authorization": r.Header.Get("Authorization"),
		})
	}))
	defer srv.Close()

	// when: generating galaxy-cli through the actual CLI — no config
	// override, so the binary derives to "scalar-galaxy"
	spec, err := filepath.Abs("testdata/specs/galaxy.yaml")
	if err != nil {
		t.Fatal(err)
	}
	outDir := filepath.Join(work, "galaxy-cli")
	gen := exec.Command(biscuitBin, "generate", "--spec", spec, "--out", outDir, "--quiet")
	gen.Dir = work
	if out, err := gen.CombinedOutput(); err != nil {
		t.Fatalf("biscuit generate: %v\n%s", err, out)
	}

	cliBin := filepath.Join(work, "scalar-galaxy")
	if runtime.GOOS == "windows" {
		cliBin += ".exe"
	}
	buildCLI := exec.Command("go", "build", "-o", cliBin, "./cmd/scalar-galaxy")
	buildCLI.Dir = outDir
	if out, err := buildCLI.CombinedOutput(); err != nil {
		t.Fatalf("building generated CLI: %v\n%s", err, out)
	}

	// then: "planets delete" has no per-operation security override, so it
	// inherits the spec's global requirement — a bearerAuth credential set
	// only via its env var arrives as an Authorization header on the wire
	del := exec.Command(cliBin, "planets", "delete", "--planet-id", "1", "--base-url", srv.URL)
	del.Env = append(os.Environ(), "SCALAR_GALAXY_BEARER_AUTH=tok123")
	out, err := del.CombinedOutput()
	if err != nil {
		t.Fatalf("planets delete with env auth: %v\n%s", err, out)
	}
	var echoed struct{ Method, Path, Authorization string }
	if err := json.Unmarshal(out, &echoed); err != nil {
		t.Fatalf("stdout is not the relayed response body: %q", out)
	}
	if echoed.Authorization != "Bearer tok123" {
		t.Errorf("Authorization header = %q, want %q", echoed.Authorization, "Bearer tok123")
	}
	if requests != 1 {
		t.Fatalf("requests reaching the server = %d, want 1", requests)
	}

	// then: the same operation with no credential configured anywhere
	// fails closed — exit code 3, and the server never sees the request
	noAuth := exec.Command(cliBin, "planets", "delete", "--planet-id", "1", "--base-url", srv.URL)
	noAuth.Env = envWithoutPrefix(t, "SCALAR_GALAXY_")
	out, err = noAuth.CombinedOutput()
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("missing credential: want *exec.ExitError, got %v\n%s", err, out)
	}
	if exitErr.ExitCode() != 3 {
		t.Errorf("missing credential exit code = %d, want 3 (auth)\n%s", exitErr.ExitCode(), out)
	}
	if !strings.Contains(string(out), "--bearer-auth") || !strings.Contains(string(out), "SCALAR_GALAXY_BEARER_AUTH") {
		t.Errorf("error does not name a flag/env var to set:\n%s", out)
	}
	if requests != 1 {
		t.Errorf("requests reaching the server = %d, want still 1 (no network I/O on missing auth)", requests)
	}

	// then: uploadImage's format:binary property rides a real multipart/
	// form-data body — the file's exact bytes and a sniffed image/png
	// Content-Type on the wire, not a base64 JSON string
	imgPath := filepath.Join(work, "photo.png")
	imgBytes := []byte("\x89PNG\r\n\x1a\nnot a real PNG, but starts with the magic bytes")
	if err := os.WriteFile(imgPath, imgBytes, 0o644); err != nil {
		t.Fatal(err)
	}
	upload := exec.Command(cliBin, "planets", "image", "--planet-id", "1", "--image", "@"+imgPath, "--base-url", srv.URL)
	upload.Env = append(os.Environ(), "SCALAR_GALAXY_BEARER_AUTH=tok123")
	out, err = upload.CombinedOutput()
	if err != nil {
		t.Fatalf("planets image: %v\n%s", err, out)
	}
	if !bytes.Equal(uploadedBytes, imgBytes) {
		t.Errorf("uploaded multipart bytes = %q, want %q", uploadedBytes, imgBytes)
	}
	if uploadedContentType != "image/png" {
		t.Errorf("uploaded multipart Content-Type = %q, want image/png", uploadedContentType)
	}
}

// envWithoutPrefix returns the current environment with every variable
// carrying prefix removed, so a missing-credential assertion can't be
// contaminated by the developer's own shell already exporting one.
func envWithoutPrefix(t *testing.T, prefix string) []string {
	t.Helper()
	var env []string
	for _, kv := range os.Environ() {
		if !strings.HasPrefix(kv, prefix) {
			env = append(env, kv)
		}
	}
	return env
}

// TestEndToEndPaginationWalk pins transparent page walking and SSE streaming
// end to end. paginated.yaml carries one endpoint per pagination family — a
// Stripe-shaped cursor and a classic offset/limit — plus one text/event-stream
// endpoint, because the walk's stop conditions, next-request arithmetic, and
// the SSE parse/dispatch loop only exist in generated code, so nothing short
// of a built binary against a real server proves them.
func TestEndToEndPaginationWalk(t *testing.T) {
	if testing.Short() {
		t.Skip("e2e skipped in -short")
	}

	// given: a server paging 6 widgets three at a time, cursored on the id of
	// each page's last widget, 4 gadgets behind an offset/limit next URL, and
	// a chat endpoint streaming 3 SSE events (one split across two data:
	// lines) followed by an OpenAI-style "[DONE]" tail
	work := t.TempDir()
	biscuitBin := filepath.Join(work, "biscuit")
	build := exec.Command("go", "build", "-o", biscuitBin, "./cmd/biscuit")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("building biscuit: %v\n%s", err, out)
	}

	widgets := []string{"w1", "w2", "w3", "w4", "w5", "w6"}
	var widgetRequests []string
	var chatAccept string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/widgets/chat" {
			chatAccept = r.Header.Get("Accept")
			w.Header().Set("Content-Type", "text/event-stream")
			flusher, _ := w.(http.Flusher)
			for _, chunk := range []string{
				"data: {\"delta\":\"Hello\"}\n\n",
				// a multi-line event: two data: lines join with \n, still
				// valid JSON since whitespace is allowed after the colon
				"data: {\"delta\":\ndata: \" world\"}\n\n",
				"data: {\"delta\":\"!\"}\n\n",
				"data: [DONE]\n\n",
			} {
				_, _ = fmt.Fprint(w, chunk)
				if flusher != nil {
					flusher.Flush()
				}
			}
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/gadgets" {
			offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
			next := ""
			if offset+2 < 4 {
				next = fmt.Sprintf("/gadgets?limit=2&offset=%d", offset+2)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"results": []map[string]string{{"id": fmt.Sprintf("g%d", offset+1)}, {"id": fmt.Sprintf("g%d", offset+2)}},
				"next":    next,
				"total":   4,
			})
			return
		}
		widgetRequests = append(widgetRequests, r.URL.RawQuery)
		start := 0
		if after := r.URL.Query().Get("after"); after != "" {
			for i, id := range widgets {
				if id == after {
					start = i + 1
				}
			}
		}
		end := min(start+2, len(widgets))
		page := make([]map[string]string, 0, end-start)
		for _, id := range widgets[start:end] {
			page = append(page, map[string]string{"id": id})
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"data": page, "has_more": end < len(widgets)})
	}))
	defer srv.Close()

	// when: generating and building the CLI for the paginated spec
	spec, err := filepath.Abs("testdata/specs/paginated.yaml")
	if err != nil {
		t.Fatal(err)
	}
	outDir := filepath.Join(work, "paginated-cli")
	gen := exec.Command(biscuitBin, "generate", "--spec", spec, "--out", outDir, "--quiet")
	gen.Dir = work
	if out, err := gen.CombinedOutput(); err != nil {
		t.Fatalf("biscuit generate: %v\n%s", err, out)
	}
	cliBin := filepath.Join(work, "paginated")
	if runtime.GOOS == "windows" {
		cliBin += ".exe"
	}
	buildCLI := exec.Command("go", "build", "-o", cliBin, "./cmd/paginated")
	buildCLI.Dir = outDir
	if out, err := buildCLI.CombinedOutput(); err != nil {
		t.Fatalf("building generated CLI: %v\n%s", err, out)
	}

	// then: the default walk fetches every page and emits every item —
	// stdout is a pipe here, so items stream as JSONL, one per line
	all := exec.Command(cliBin, "widgets", "list", "--limit", "2", "--base-url", srv.URL)
	out, err := all.CombinedOutput()
	if err != nil {
		t.Fatalf("widgets list: %v\n%s", err, out)
	}
	lines := strings.Split(strings.TrimRight(string(out), "\n"), "\n")
	if len(lines) != len(widgets) {
		t.Errorf("walked output = %d lines, want %d (one per widget):\n%s", len(lines), len(widgets), out)
	}
	for _, want := range []string{"w1", "w6"} {
		if !strings.Contains(string(out), want) {
			t.Errorf("walked output missing %s:\n%s", want, out)
		}
	}
	if len(widgetRequests) != 3 {
		t.Errorf("requests = %d (%v), want 3 pages", len(widgetRequests), widgetRequests)
	}
	// then: each page after the first carries the previous page's last id as
	// its cursor, so the walk advances rather than refetching page one
	if len(widgetRequests) > 1 && !strings.Contains(widgetRequests[1], "after=w2") {
		t.Errorf("second request query = %q, want it to carry after=w2", widgetRequests[1])
	}

	// then: --max-pages bounds the walk, stopping mid-stream
	widgetRequests = nil
	bounded := exec.Command(cliBin, "widgets", "list", "--limit", "2", "--max-pages", "2", "--base-url", srv.URL)
	out, err = bounded.CombinedOutput()
	if err != nil {
		t.Fatalf("widgets list --max-pages 2: %v\n%s", err, out)
	}
	if len(widgetRequests) != 2 {
		t.Errorf("--max-pages 2 made %d requests (%v), want 2", len(widgetRequests), widgetRequests)
	}
	if n := len(strings.Split(strings.TrimRight(string(out), "\n"), "\n")); n != 4 {
		t.Errorf("--max-pages 2 emitted %d items, want 4:\n%s", n, out)
	}

	// then: a document format merges every page into one JSON array instead
	// of printing one array per page
	merged := exec.Command(cliBin, "widgets", "list", "--limit", "2", "--format", "json", "--base-url", srv.URL)
	out, err = merged.CombinedOutput()
	if err != nil {
		t.Fatalf("widgets list --format json: %v\n%s", err, out)
	}
	var items []struct{ ID string }
	if err := json.Unmarshal(out, &items); err != nil {
		t.Fatalf("--format json output is not one JSON array: %v\n%s", err, out)
	}
	if len(items) != len(widgets) {
		t.Errorf("merged array = %d items, want %d:\n%s", len(items), len(widgets), out)
	}

	// then: --include-headers prints one header block, not one per page
	withHeaders := exec.Command(cliBin, "widgets", "list", "--limit", "2", "--include-headers", "--base-url", srv.URL)
	out, err = withHeaders.CombinedOutput()
	if err != nil {
		t.Fatalf("widgets list --include-headers: %v\n%s", err, out)
	}
	if n := strings.Count(string(out), "HTTP 200"); n != 1 {
		t.Errorf("--include-headers printed %d status lines over a 3-page walk, want 1:\n%s", n, out)
	}

	// then: an offset walk follows the response's next URL, applying its
	// query onto the same operation
	gadgets := exec.Command(cliBin, "gadgets", "list", "--limit", "2", "--format", "json", "--base-url", srv.URL)
	out, err = gadgets.CombinedOutput()
	if err != nil {
		t.Fatalf("gadgets list: %v\n%s", err, out)
	}
	if err := json.Unmarshal(out, &items); err != nil {
		t.Fatalf("gadgets output is not one JSON array: %v\n%s", err, out)
	}
	if len(items) != 4 {
		t.Errorf("gadgets merged array = %d items, want 4:\n%s", len(items), out)
	}

	// then: an SSE endpoint streams line-per-event JSONL when piped — the
	// multi-line event's data: lines joined, and the "[DONE]" sentinel tail
	// dropped rather than printed as a fourth line
	chat := exec.Command(cliBin, "widgets", "chat", "--message", "hi", "--base-url", srv.URL)
	out, err = chat.CombinedOutput()
	if err != nil {
		t.Fatalf("widgets chat: %v\n%s", err, out)
	}
	chatLines := strings.Split(strings.TrimRight(string(out), "\n"), "\n")
	wantChat := []string{`{"delta":"Hello"}`, `{"delta":" world"}`, `{"delta":"!"}`}
	if len(chatLines) != len(wantChat) {
		t.Fatalf("widgets chat lines = %d, want %d:\n%s", len(chatLines), len(wantChat), out)
	}
	for i, want := range wantChat {
		if chatLines[i] != want {
			t.Errorf("widgets chat line %d = %q, want %q", i, chatLines[i], want)
		}
	}
	if chatAccept != "text/event-stream" {
		t.Errorf("chat request Accept header = %q, want text/event-stream", chatAccept)
	}

	// then: --transform extracts a field from each event, not just once from
	// the whole (nonexistent, for a stream) response body
	chatTransform := exec.Command(cliBin, "widgets", "chat", "--message", "hi", "--transform", "delta", "--base-url", srv.URL)
	out, err = chatTransform.CombinedOutput()
	if err != nil {
		t.Fatalf("widgets chat --transform delta: %v\n%s", err, out)
	}
	// gjson's Raw preserves JSON quoting, same as --transform on a buffered
	// response (see TestEndToEndGeneratedCLIMakesRequests)
	wantDeltas := []string{`"Hello"`, `" world"`, `"!"`}
	deltaLines := strings.Split(strings.TrimRight(string(out), "\n"), "\n")
	if len(deltaLines) != len(wantDeltas) {
		t.Fatalf("widgets chat --transform delta lines = %d, want %d:\n%s", len(deltaLines), len(wantDeltas), out)
	}
	for i, want := range wantDeltas {
		if deltaLines[i] != want {
			t.Errorf("widgets chat --transform delta line %d = %q, want %q", i, deltaLines[i], want)
		}
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
