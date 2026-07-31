// Package config loads and validates biscuit.yaml, the canonical override
// surface. The config drives codegen, so a malformed file fails loudly —
// unknown keys are rejected with the decoder's line-precise errors, and a
// version key exists for forward migration. The same per-operation overrides
// can ride in-spec as x-biscuit-* extensions; the sidecar wins on conflict
// (merging happens in the mapping layer).
package config

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"go.yaml.in/yaml/v4"
)

const FileName = "biscuit.yaml"

type Config struct {
	Version      int                  `yaml:"version,omitempty"`
	Spec         Spec                 `yaml:"spec,omitempty"`
	Output       Output               `yaml:"output,omitempty"`
	Lint         Lint                 `yaml:"lint,omitempty"`
	Distribution Distribution         `yaml:"distribution,omitempty"`
	Pagination   []PaginationScheme   `yaml:"pagination,omitempty"`
	Operations   map[string]Operation `yaml:"operations,omitempty"`
}

// PaginationScheme is one named pagination convention. A declared scheme
// shadows the built-in of the same name and is matched against operation
// shapes the same way — declaring one is how an API whose convention biscuit
// doesn't already know gets transparent page walking.
type PaginationScheme struct {
	Name     string             `yaml:"name,omitempty"`
	Type     string             `yaml:"type,omitempty"` // cursor | offset | page
	Request  PaginationRequest  `yaml:"request,omitempty"`
	Response PaginationResponse `yaml:"response,omitempty"`
}

// PaginationRequest names the query params the scheme sends. Exactly one of
// the three page params is set, matching the scheme's type.
type PaginationRequest struct {
	CursorParam string `yaml:"cursor_param,omitempty"`
	OffsetParam string `yaml:"offset_param,omitempty"`
	PageParam   string `yaml:"page_param,omitempty"`
	LimitParam  string `yaml:"limit_param,omitempty"`
}

// PaginationResponse names the response fields the scheme reads. Paths are
// dot-separated for nesting (Slack's response_metadata.next_cursor).
type PaginationResponse struct {
	ItemsField string `yaml:"items_field,omitempty"`
	NextField  string `yaml:"next_field,omitempty"`
	NextKind   string `yaml:"next_kind,omitempty"` // has_more | cursor | url | link_header
}

var (
	paginationTypes     = []string{"cursor", "offset", "page"}
	paginationNextKinds = []string{"cursor", "has_more", "link_header", "url"}
)

// Output shapes the emitted repository. Every field has a spec-derived
// default; the config only pins what the derivation gets wrong.
type Output struct {
	Binary      string `yaml:"binary,omitempty"`      // command name; default derived from info.title
	Module      string `yaml:"module,omitempty"`      // Go module path; default example.com/{binary}-cli
	Dir         string `yaml:"dir,omitempty"`         // output directory; default ./{binary}-cli
	Description string `yaml:"description,omitempty"` // overrides info.description in the generated README/Makefile/root.Long
}

type Spec struct {
	Path string `yaml:"path,omitempty"`
}

type Lint struct {
	MinGrade int `yaml:"min_grade,omitempty"`
}

// Distribution controls which release channels the generated repo ships
// with. Only Homebrew gates template output today; NPM and InstallScript are
// read by later stories (npm publishing, install.sh templating).
type Distribution struct {
	Homebrew      bool `yaml:"homebrew,omitempty"`
	InstallScript bool `yaml:"install_script,omitempty"`
	NPM           NPM  `yaml:"npm,omitempty"`
}

type NPM struct {
	Package string `yaml:"package,omitempty"`
}

// Operation overrides how one operation maps into the command tree, keyed by
// operationId — or "METHOD /path" when the spec has none.
type Operation struct {
	Name       string   `yaml:"name,omitempty"`       // verb name
	Group      string   `yaml:"group,omitempty"`      // whitespace-separated resource chain
	Ignore     bool     `yaml:"ignore,omitempty"`     // drop the operation from the CLI
	Aliases    []string `yaml:"aliases,omitempty"`    // extra verb names
	Pagination string   `yaml:"pagination,omitempty"` // pagination scheme name to force, or "off"
}

// Load reads dir/biscuit.yaml. A missing file is not an error: it returns an
// empty config, the state before discovery or init has written anything.
func Load(dir string) (*Config, error) {
	path := filepath.Join(dir, FileName)
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return &Config{}, nil
	}
	if err != nil {
		return nil, err
	}

	var cfg Config
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	if err := dec.Decode(&cfg); err != nil && !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	if cfg.Version > 1 {
		return nil, fmt.Errorf("%s: config version %d is newer than this biscuit understands (max 1); upgrade biscuit", path, cfg.Version)
	}
	if err := validatePagination(cfg.Pagination); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return &cfg, nil
}

// validatePagination rejects a scheme the matcher could never apply, at load
// time and by name — a scheme silently ignored would look like biscuit simply
// failed to detect pagination.
func validatePagination(schemes []PaginationScheme) error {
	seen := map[string]bool{}
	for _, s := range schemes {
		if s.Name == "" {
			return errors.New("pagination: every scheme needs a name")
		}
		if seen[s.Name] {
			return fmt.Errorf("pagination scheme %q: declared twice", s.Name)
		}
		seen[s.Name] = true

		if !slices.Contains(paginationTypes, s.Type) {
			return fmt.Errorf("pagination scheme %q: unknown type %q (want %s)", s.Name, s.Type, strings.Join(paginationTypes, ", "))
		}
		want, got := map[string]string{
			"cursor": "cursor_param", "offset": "offset_param", "page": "page_param",
		}[s.Type], map[string]string{
			"cursor": s.Request.CursorParam, "offset": s.Request.OffsetParam, "page": s.Request.PageParam,
		}[s.Type]
		if got == "" {
			return fmt.Errorf("pagination scheme %q: type %s needs request.%s", s.Name, s.Type, want)
		}

		kind := s.Response.NextKind
		if kind != "" && !slices.Contains(paginationNextKinds, kind) {
			return fmt.Errorf("pagination scheme %q: unknown next_kind %q (want %s)", s.Name, kind, strings.Join(paginationNextKinds, ", "))
		}
		switch {
		case s.Response.NextField != "" && kind == "":
			return fmt.Errorf("pagination scheme %q: response.next_field needs response.next_kind", s.Name)
		case s.Response.NextField == "" && kind != "" && kind != "link_header":
			return fmt.Errorf("pagination scheme %q: next_kind %s needs response.next_field", s.Name, kind)
		case s.Type == "cursor" && s.Response.NextField == "":
			// a cursor walk has no arithmetic to fall back on
			return fmt.Errorf("pagination scheme %q: a cursor scheme needs response.next_field", s.Name)
		}
	}
	return nil
}
