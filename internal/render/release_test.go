package render

import (
	"encoding/json"
	"strings"
	"testing"

	"go.yaml.in/yaml/v4"

	"github.com/oxmonty/biscuit/internal/config"
	"github.com/oxmonty/biscuit/internal/mapping"
	"github.com/oxmonty/biscuit/internal/spec"
)

func findFile(t *testing.T, files []File, path string) File {
	t.Helper()
	for _, f := range files {
		if f.Path == path {
			return f
		}
	}
	t.Fatalf("plan missing %s", path)
	return File{}
}

func TestRenderReleaseFiles(t *testing.T) {
	// given: a petstore spec rendered with homebrew distribution enabled and
	// a github.com module path
	doc, err := spec.Load(ladder + "petstore.yaml")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	api := mapping.Map(doc, nil)
	cfg := &config.Config{
		Output:       config.Output{Module: "github.com/acme/petstore-cli"},
		Distribution: config.Distribution{Homebrew: true},
	}
	files, err := Render(api, cfg, Provenance{SpecPath: "petstore.yaml", SpecSHA256: "test"})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}

	// when: pulling out each release file
	goreleaser := findFile(t, files, ".goreleaser.yaml")
	ci := findFile(t, files, ".github/workflows/ci.yaml")
	release := findFile(t, files, ".github/workflows/release.yaml")
	rpConfig := findFile(t, files, "release-please-config.json")
	rpManifest := findFile(t, files, ".release-please-manifest.json")

	// then: the goreleaser config and both workflows parse as YAML
	var goreleaserDoc map[string]any
	if err := yaml.Unmarshal(goreleaser.Contents, &goreleaserDoc); err != nil {
		t.Fatalf(".goreleaser.yaml did not parse: %v\n%s", err, goreleaser.Contents)
	}
	var ciDoc map[string]any
	if err := yaml.Unmarshal(ci.Contents, &ciDoc); err != nil {
		t.Fatalf("ci.yaml did not parse: %v\n%s", err, ci.Contents)
	}
	var releaseDoc map[string]any
	if err := yaml.Unmarshal(release.Contents, &releaseDoc); err != nil {
		t.Fatalf("release.yaml did not parse: %v\n%s", err, release.Contents)
	}

	// then: the homebrew cask section is present because Distribution.Homebrew is true
	if _, ok := goreleaserDoc["homebrew_casks"]; !ok {
		t.Error(".goreleaser.yaml missing homebrew_casks with Homebrew=true")
	}

	// then: the release workflow carries the dedicated release-go- cache key
	if !strings.Contains(string(release.Contents), "release-go-") {
		t.Error("release.yaml missing the release-go- cache key")
	}

	// then: the release-please files are valid and start from a fresh manifest
	var rpConfigDoc map[string]any
	if err := json.Unmarshal(rpConfig.Contents, &rpConfigDoc); err != nil {
		t.Fatalf("release-please-config.json did not parse: %v\n%s", err, rpConfig.Contents)
	}
	if strings.TrimSpace(string(rpManifest.Contents)) != `{".": "0.0.0"}` {
		t.Errorf(".release-please-manifest.json = %q, want {\".\": \"0.0.0\"}", rpManifest.Contents)
	}
}

func TestRenderOmitsHomebrewCasksWhenDisabled(t *testing.T) {
	// given: the same spec rendered with Distribution.Homebrew left false
	doc, err := spec.Load(ladder + "petstore.yaml")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	api := mapping.Map(doc, nil)
	cfg := &config.Config{Output: config.Output{Module: "github.com/acme/petstore-cli"}}
	files, err := Render(api, cfg, Provenance{SpecPath: "petstore.yaml", SpecSHA256: "test"})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}

	// when: parsing the goreleaser config
	goreleaser := findFile(t, files, ".goreleaser.yaml")
	var doc2 map[string]any
	if err := yaml.Unmarshal(goreleaser.Contents, &doc2); err != nil {
		t.Fatalf(".goreleaser.yaml did not parse: %v\n%s", err, goreleaser.Contents)
	}

	// then: no cask section is emitted
	if _, ok := doc2["homebrew_casks"]; ok {
		t.Error(".goreleaser.yaml has homebrew_casks with Homebrew=false")
	}
}
