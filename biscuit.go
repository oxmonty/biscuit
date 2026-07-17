// Package biscuit converts an OpenAPI 3.x spec into a complete CLI repository.
// The CLI (cmd/biscuit) is the first consumer of this API; Generate arrives
// with the mapping and rendering epics.
package biscuit

import "github.com/oxmonty/biscuit/internal/spec"

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
