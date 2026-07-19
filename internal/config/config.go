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

	"go.yaml.in/yaml/v4"
)

const FileName = "biscuit.yaml"

type Config struct {
	Version    int                  `yaml:"version,omitempty"`
	Spec       Spec                 `yaml:"spec,omitempty"`
	Lint       Lint                 `yaml:"lint,omitempty"`
	Operations map[string]Operation `yaml:"operations,omitempty"`
}

type Spec struct {
	Path string `yaml:"path,omitempty"`
}

type Lint struct {
	MinGrade int `yaml:"min_grade,omitempty"`
}

// Operation overrides how one operation maps into the command tree, keyed by
// operationId — or "METHOD /path" when the spec has none.
type Operation struct {
	Name       string   `yaml:"name,omitempty"`       // verb name
	Group      string   `yaml:"group,omitempty"`      // whitespace-separated resource chain
	Ignore     bool     `yaml:"ignore,omitempty"`     // drop the operation from the CLI
	Aliases    []string `yaml:"aliases,omitempty"`    // extra verb names
	Pagination string   `yaml:"pagination,omitempty"` // hint for the execution layer
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
	return &cfg, nil
}
