package render

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/oxmonty/biscuit/internal/config"
	"github.com/oxmonty/biscuit/internal/mapping"
	"github.com/oxmonty/biscuit/internal/spec"
)

var update = flag.Bool("update", false, "rewrite golden repos and the FilePlan hash file")

const goldenDir = "../../testdata/golden"

// goldenSpecs are the committed golden repos: the easy rung plus the
// feature-dense small spec (multi-auth, uploads, a real cycle). Larger ladder
// specs are pinned by FilePlan hash instead of committed bytes — see
// TestFilePlanHashes.
var goldenSpecs = []struct {
	name   string
	spec   string
	config config.Config
}{
	{
		name: "petstore",
		spec: "../../testdata/specs/petstore.yaml",
		config: config.Config{
			Output:       config.Output{Binary: "petstore", Module: "github.com/oxmonty/petstore-cli"},
			Distribution: config.Distribution{Homebrew: true, InstallScript: true},
		},
	},
	{
		name: "galaxy",
		spec: "../../testdata/specs/galaxy.yaml",
		config: config.Config{
			Output:       config.Output{Binary: "galaxy", Module: "github.com/oxmonty/galaxy-cli"},
			Distribution: config.Distribution{Homebrew: true},
		},
	},
}

func renderSpec(t *testing.T, specPath string, cfg *config.Config) []File {
	t.Helper()
	doc, err := spec.Load(specPath)
	if err != nil {
		t.Fatalf("Load(%s): %v", specPath, err)
	}
	api := mapping.Map(doc, cfg)
	files, err := Render(api, cfg, Provenance{SpecPath: filepath.Base(specPath), SpecSHA256: "golden"})
	if err != nil {
		t.Fatalf("Render(%s): %v", specPath, err)
	}
	return files
}

// TestGoldenRepos pins the full byte output for the committed corpus. Extra
// files in a golden dir (the representative internal/custom fixture) are
// deliberately tolerated: the comparison walks the plan, not the dir.
func TestGoldenRepos(t *testing.T) {
	for _, g := range goldenSpecs {
		t.Run(g.name, func(t *testing.T) {
			// given: the rendered plan for the pinned spec+config
			files := renderSpec(t, g.spec, &g.config)
			repoDir := filepath.Join(goldenDir, g.name, "repo")

			if *update {
				// rewrite only planned paths; fixtures and stale files are
				// handled below and by review
				if err := os.RemoveAll(repoDir); err != nil {
					t.Fatal(err)
				}
				for _, f := range files {
					target := filepath.Join(repoDir, filepath.FromSlash(f.Path))
					if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
						t.Fatal(err)
					}
					mode := os.FileMode(0o644)
					if f.Executable {
						mode = 0o755
					}
					if err := os.WriteFile(target, f.Contents, mode); err != nil {
						t.Fatal(err)
					}
				}
				restoreFixtures(t, g.name, repoDir)
				return
			}

			// then: every planned file matches its committed golden bytes
			for _, f := range files {
				golden, err := os.ReadFile(filepath.Join(repoDir, filepath.FromSlash(f.Path)))
				if err != nil {
					t.Errorf("%s: missing golden file (run go test ./internal/render -update): %v", f.Path, err)
					continue
				}
				if !bytes.Equal(golden, f.Contents) {
					t.Errorf("%s: differs from golden (run go test ./internal/render -update and review the diff)", f.Path)
				}
			}
		})
	}
}

// restoreFixtures re-adds the hand-written files -update wiped: the
// representative internal/custom fixture that makes compile-the-output a
// contract gate, not just a syntax gate.
func restoreFixtures(t *testing.T, name, repoDir string) {
	t.Helper()
	fixtures := filepath.Join(goldenDir, name, "fixtures")
	entries, err := os.ReadDir(fixtures)
	if os.IsNotExist(err) {
		return
	}
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		src, err := os.ReadFile(filepath.Join(fixtures, e.Name()))
		if err != nil {
			t.Fatal(err)
		}
		target := filepath.Join(repoDir, "internal", "custom", e.Name())
		if err := os.WriteFile(target, src, 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

// TestFilePlanHashes pins determinism for the ladder specs too large to
// commit: same biscuit + same spec must produce the same plan bytes. The
// compile gate for these runs in TestCompileBigSpecs.
func TestFilePlanHashes(t *testing.T) {
	specs := []string{"museum.yaml", "train-travel.yaml", "pokeapi.yml", "openai.yaml", "stripe.yaml", "paginated.yaml"}
	hashFile := filepath.Join(goldenDir, "hashes.txt")

	var lines []string
	for _, name := range specs {
		files := renderSpec(t, "../../testdata/specs/"+name, &config.Config{})
		h := sha256.New()
		for _, f := range files {
			_, _ = fmt.Fprintf(h, "%s\x00%d\x00", f.Path, len(f.Contents))
			h.Write(f.Contents)
		}
		lines = append(lines, fmt.Sprintf("%s  %s", hex.EncodeToString(h.Sum(nil)), name))
	}
	sort.Strings(lines)
	got := strings.Join(lines, "\n") + "\n"

	if *update {
		if err := os.WriteFile(hashFile, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(hashFile)
	if err != nil {
		t.Fatalf("missing %s (run go test ./internal/render -update): %v", hashFile, err)
	}
	if string(want) != got {
		t.Errorf("FilePlan hashes changed — a template or mapping change altered large-spec output.\nIf intended, run go test ./internal/render -update.\ngot:\n%swant:\n%s", got, want)
	}
}

// TestCompileGoldenRepos is the gate that matters most: go build ./... on the
// committed golden repos, including galaxy's internal/custom fixture — a
// regeneration that breaks the custom/ contract fails here, in biscuit's CI.
// It then runs each repo's own smoke suite, which drives every generated
// command against the spec-derived mock: compile-the-output extended to
// run-the-output.
func TestCompileGoldenRepos(t *testing.T) {
	if testing.Short() {
		t.Skip("compile gate skipped in -short")
	}
	for _, g := range goldenSpecs {
		t.Run(g.name, func(t *testing.T) {
			repoDir, err := filepath.Abs(filepath.Join(goldenDir, g.name, "repo"))
			if err != nil {
				t.Fatal(err)
			}
			if _, err := os.Stat(repoDir); err != nil {
				t.Fatalf("golden repo missing (run go test ./internal/render -update): %v", err)
			}
			for _, args := range [][]string{{"build", "./..."}, {"test", "./..."}} {
				cmd := exec.Command("go", args...)
				cmd.Dir = repoDir
				if out, err := cmd.CombinedOutput(); err != nil {
					t.Errorf("go %s failed:\n%s", strings.Join(args, " "), out)
				}
			}
		})
	}
}

// openaiCustomFixture is injected into the ephemeral openai build so the
// internal/custom contract is compile-checked on the hardest spec too, not
// only on galaxy's committed golden repo.
const openaiCustomFixture = `package custom

import (
	"context"

	"example.com/open-ai-cli/internal/client"
)

// CreateChatCompletionRaw depends on the client's stable surface on the
// hardest ladder spec: Client construction, a Request with Body, an
// operation method, and Response fields.
func CreateChatCompletionRaw(ctx context.Context, baseURL string, body []byte) (int, []byte, error) {
	c := &client.Client{BaseURL: baseURL}
	resp, err := c.ChatCompletionsCreate(ctx, &client.ChatCompletionsCreateRequest{Body: body})
	if err != nil {
		return 0, nil, err
	}
	return resp.Status, resp.Body, nil
}
`

// TestCompileBigSpecs generates every non-golden ladder spec into a scratch
// dir, builds it, and throws the result away — full compile coverage with no
// committed bytes. The openai run also carries a custom fixture, so the
// custom/ contract gate covers the hardest spec.
//
// go test follows go build so each repo's generated smoke suite runs too — go
// build alone never compiles test files, and these specs are where the odd
// shapes live (stripe's 587 commands run in under two seconds).
func TestCompileBigSpecs(t *testing.T) {
	if testing.Short() {
		t.Skip("compile gate skipped in -short")
	}
	for _, name := range []string{"museum.yaml", "train-travel.yaml", "pokeapi.yml", "openai.yaml", "stripe.yaml"} {
		t.Run(name, func(t *testing.T) {
			files := renderSpec(t, "../../testdata/specs/"+name, &config.Config{})
			dir := t.TempDir()
			if _, err := WriteFiles(dir, files); err != nil {
				t.Fatal(err)
			}
			if name == "openai.yaml" {
				fixture := filepath.Join(dir, "internal", "custom", "hooks.go")
				if err := os.WriteFile(fixture, []byte(openaiCustomFixture), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			for _, args := range [][]string{{"build", "./..."}, {"test", "./..."}} {
				cmd := exec.Command("go", args...)
				cmd.Dir = dir
				if out, err := cmd.CombinedOutput(); err != nil {
					t.Errorf("go %s failed:\n%s", strings.Join(args, " "), out)
				}
			}
		})
	}
}
