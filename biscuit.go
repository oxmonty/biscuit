// Package biscuit converts an OpenAPI 3.x spec into a complete CLI repository.
// The CLI (cmd/biscuit) is the first consumer of this API. Generate is a pure
// function — no disk I/O, no side effects; writing the plan is the separate
// FilePlan.Write step. That split is what yields --dry-run for free and keeps
// a future hosted service a thin wrapper.
package biscuit

import (
	"context"
	"os"
	"path/filepath"

	"github.com/oxmonty/biscuit/internal/config"
	"github.com/oxmonty/biscuit/internal/mapping"
	"github.com/oxmonty/biscuit/internal/spec"
)

// Config is the parsed biscuit.yaml. LoadConfig reads and validates one;
// unknown keys are rejected so a malformed config never mis-generates.
type Config = config.Config

// Operation is one per-operation override entry in Config.Operations.
type Operation = config.Operation

// LoadConfig reads dir/biscuit.yaml; a missing file yields an empty config.
func LoadConfig(dir string) (*Config, error) { return config.Load(dir) }

// Document is a parsed, validated OpenAPI 3.x spec ready for generation.
type Document struct {
	doc *spec.Document
}

// Load reads, parses, and resolves the spec at path. Blocking correctness
// problems (unparseable spec, unresolvable $refs, duplicate operationIds)
// fail the load; advisory findings are available via Diagnostics.
func Load(path string) (*Document, error) {
	d, err := spec.Load(path)
	if err != nil {
		return nil, err
	}
	return &Document{doc: d}, nil
}

// Title returns the spec's info.title.
func (d *Document) Title() string { return d.doc.Model.Info.Title }

// Version returns the OpenAPI version the spec declares (e.g. "3.1.0").
func (d *Document) Version() string { return d.doc.Model.Version }

// Operations returns the number of path operations (webhooks not included).
func (d *Document) Operations() int { return d.doc.Operations() }

// Diagnostics returns advisory findings that degrade generated-CLI quality
// but never block generation.
func (d *Document) Diagnostics() []string { return d.doc.Diagnostics }

// FilePlan is the complete set of files a generation run would write.
type FilePlan struct {
	Files []PlannedFile // sorted by Path; empty until the rendering epic lands
}

type PlannedFile struct {
	Path     string // relative to the output dir, slash-separated
	Contents []byte
}

// Generate derives the command surface from a loaded spec and plans the
// output repository. Pure: nothing is written until FilePlan.Write. The
// rendering epic fills Files; today the run derives and validates the
// command tree (overrides included) and returns an empty plan.
func Generate(_ context.Context, doc *Document, cfg *Config) (*FilePlan, error) {
	mapping.Map(doc.doc, mapping.OverridesFromConfig(cfg))
	return &FilePlan{}, nil
}

// Write materializes the plan under dir, creating directories as needed.
func (p *FilePlan) Write(dir string) error {
	for _, f := range p.Files {
		target := filepath.Join(dir, filepath.FromSlash(f.Path))
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(target, f.Contents, 0o644); err != nil {
			return err
		}
	}
	return nil
}
