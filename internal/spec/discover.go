package spec

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ErrNoSpecFound means discovery ran and found nothing; the CLI maps it to exit 3.
var ErrNoSpecFound = errors.New("no OpenAPI spec found (looked for openapi|swagger.{yaml,yml,json}, then sniffed yaml/json files)")

// wellKnown are conventional spec filenames, in rank order.
var wellKnown = []string{
	"openapi.yaml", "openapi.yml", "openapi.json",
	"swagger.yaml", "swagger.yml", "swagger.json",
}

// DiscoverCandidates scans dir (flat — deeper enumeration ships with E8's
// discovery UX) for specs: well-known names first, then remaining yaml/json
// whose first ~1KB carries an `openapi:` root key. Best-ranked first.
func DiscoverCandidates(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	names := map[string]bool{}
	for _, e := range entries {
		if !e.IsDir() {
			names[e.Name()] = true
		}
	}

	var candidates []string
	for _, name := range wellKnown {
		if names[name] {
			candidates = append(candidates, name)
			delete(names, name)
		}
	}

	var sniffed []string
	for name := range names {
		switch strings.ToLower(filepath.Ext(name)) {
		case ".yaml", ".yml", ".json":
			if sniffsAsOpenAPI(filepath.Join(dir, name)) {
				sniffed = append(sniffed, name)
			}
		}
	}
	sort.Strings(sniffed)
	candidates = append(candidates, sniffed...)

	if len(candidates) == 0 {
		return nil, ErrNoSpecFound
	}
	for i, c := range candidates {
		candidates[i] = filepath.Join(dir, c)
	}
	return candidates, nil
}

// sniffsAsOpenAPI reports whether the file's head looks like an OpenAPI doc:
// a root-level `openapi:` key (yaml) or `"openapi":` (json).
func sniffsAsOpenAPI(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer func() { _ = f.Close() }()
	head := make([]byte, 1024)
	n, _ := f.Read(head)
	head = head[:n]
	if bytes.Contains(head, []byte(`"openapi"`)) {
		return true
	}
	for line := range strings.Lines(string(head)) {
		if strings.HasPrefix(line, "openapi:") {
			return true
		}
	}
	return false
}

// PersistSpecPath records the chosen spec as spec.path in dir/biscuit.yaml so
// discovery runs once. A fresh file is created; an existing one gets the spec
// block appended. ponytail: if a spec: block already exists without a path we
// leave the file alone (re-discovery is harmless) — E3's schema-validated
// config loader owns real config rewriting.
func PersistSpecPath(dir, specPath string) error {
	rel, err := filepath.Rel(dir, specPath)
	if err != nil {
		rel = specPath
	}
	cfgPath := filepath.Join(dir, "biscuit.yaml")
	block := fmt.Sprintf("spec:\n  path: %s\n", rel)

	data, err := os.ReadFile(cfgPath)
	switch {
	case errors.Is(err, os.ErrNotExist):
		return os.WriteFile(cfgPath, []byte(block), 0o644)
	case err != nil:
		return err
	case strings.Contains(string(data), "spec:"):
		return nil
	}
	if !bytes.HasSuffix(data, []byte("\n")) {
		data = append(data, '\n')
	}
	return os.WriteFile(cfgPath, append(data, block...), 0o644)
}
