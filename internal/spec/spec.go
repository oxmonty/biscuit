// Package spec loads an OpenAPI 3.x document into libopenapi's model,
// classifying failures per biscuit's contract: blocking correctness problems
// (unparseable spec, unresolvable $refs, duplicate operationIds) surface as
// *InvalidError, while advisory findings (circular refs, $refs inside vendor
// extensions, resolver noise) land in Document.Diagnostics.
package spec

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/pb33f/libopenapi"
	"github.com/pb33f/libopenapi/datamodel"
	v3 "github.com/pb33f/libopenapi/datamodel/high/v3"
	"github.com/pb33f/libopenapi/index"
)

// Document is a parsed, resolved spec plus everything downstream phases need.
type Document struct {
	Path  string
	Bytes []byte // raw spec; doctor (vacuum) parses these, not the model
	Model *v3.Document
	Index *index.SpecIndex
	// Diagnostics are advisory: they degrade output quality but never block.
	Diagnostics []string
}

// InvalidError reports blocking correctness problems; the CLI maps it to exit code 4.
type InvalidError struct {
	Path     string
	Problems []string
}

func (e *InvalidError) Error() string {
	return fmt.Sprintf("spec %s is invalid:\n  - %s", e.Path, strings.Join(e.Problems, "\n  - "))
}

// Load reads, parses, and resolves the spec at path.
func Load(path string) (*Document, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err // fs errors keep their type; the CLI maps ErrNotExist to exit 3
	}

	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}

	// libopenapi logs resolver trouble (missing files, unlocatable refs) instead
	// of returning it; collect those lines as diagnostics rather than stderr noise.
	var logged logCollector
	doc, err := libopenapi.NewDocumentWithConfiguration(data, &datamodel.DocumentConfiguration{
		BasePath:            filepath.Dir(abs),
		SpecFilePath:        abs,
		AllowFileReferences: true,
		Logger:              slog.New(&logged),
	})
	if err != nil {
		return nil, &InvalidError{Path: path, Problems: []string{err.Error()}}
	}

	model, buildErr := doc.BuildV3Model()
	if model == nil {
		return nil, &InvalidError{Path: path, Problems: unwrapAll(buildErr)}
	}

	d := &Document{Path: path, Bytes: data, Model: &model.Model, Index: model.Index}

	var problems []string
	if buildErr != nil {
		// $refs inside vendor extensions are opaque per the OpenAPI spec, but
		// libopenapi chases them anyway; failing to resolve one is advisory.
		// Circular references arrive as build errors too when the cycle is a
		// required chain ("infinite") — still advisory per the cycle policy;
		// stripe.yaml has two such cycles.
		extRefs := model.Index.GetExtensionRefsSequenced()
		for _, e := range unwrapAll(buildErr) {
			if refersToExtensionRef(e, extRefs) || strings.Contains(e, "circular reference") {
				d.Diagnostics = append(d.Diagnostics, e)
			} else {
				problems = append(problems, e)
			}
		}
	}

	problems = append(problems, duplicateOperationIDs(&model.Model)...)

	// Cycle-safety means we survive cycles and report them — codegen can break
	// cycles with pointers, so they degrade nothing on their own.
	model.Index.GetRolodex().CheckForCircularReferences()
	for _, c := range model.Index.GetCircularReferences() {
		d.Diagnostics = append(d.Diagnostics, "circular reference: "+c.GenerateJourneyPath())
	}

	d.Diagnostics = append(d.Diagnostics, logged.take()...)

	if len(problems) > 0 {
		return nil, &InvalidError{Path: path, Problems: problems}
	}
	return d, nil
}

// Operations returns path operations; webhooks are a separate collection
// (Model.Webhooks) and are deliberately not folded in here.
func (d *Document) Operations() int {
	if d.Model.Paths == nil {
		return 0
	}
	n := 0
	for _, item := range d.Model.Paths.PathItems.FromOldest() {
		n += item.GetOperations().Len()
	}
	return n
}

func duplicateOperationIDs(model *v3.Document) []string {
	if model.Paths == nil {
		return nil
	}
	seen := map[string]string{}
	var problems []string
	for path, item := range model.Paths.PathItems.FromOldest() {
		for method, op := range item.GetOperations().FromOldest() {
			if op.OperationId == "" {
				continue
			}
			where := method + " " + path
			if first, dup := seen[op.OperationId]; dup {
				problems = append(problems, fmt.Sprintf(
					"duplicate operationId %q: %s and %s", op.OperationId, first, where))
			} else {
				seen[op.OperationId] = where
			}
		}
	}
	return problems
}

func refersToExtensionRef(errText string, extRefs []*index.Reference) bool {
	// ponytail: substring match on the ref definition; swap for structured
	// error inspection if libopenapi ever types its build errors.
	for _, ref := range extRefs {
		if ref.Definition != "" && strings.Contains(errText, ref.Definition) {
			return true
		}
	}
	return false
}

func unwrapAll(err error) []string {
	if err == nil {
		return nil
	}
	if joined, ok := err.(interface{ Unwrap() []error }); ok {
		var all []string
		for _, e := range joined.Unwrap() {
			all = append(all, unwrapAll(e)...)
		}
		return all
	}
	return []string{err.Error()}
}

// logCollector is a minimal slog.Handler that buffers warn/error lines.
type logCollector struct {
	lines []string
}

func (c *logCollector) Enabled(_ context.Context, level slog.Level) bool {
	return level >= slog.LevelWarn
}

func (c *logCollector) Handle(_ context.Context, r slog.Record) error {
	line := r.Message
	r.Attrs(func(a slog.Attr) bool {
		if a.Key == "error" || a.Key == "reference" || a.Key == "file" {
			line += fmt.Sprintf(" (%s: %v)", a.Key, a.Value)
		}
		return true
	})
	c.lines = append(c.lines, line)
	return nil
}

func (c *logCollector) WithAttrs([]slog.Attr) slog.Handler { return c }
func (c *logCollector) WithGroup(string) slog.Handler      { return c }

func (c *logCollector) take() []string {
	lines := c.lines
	c.lines = nil
	return lines
}
