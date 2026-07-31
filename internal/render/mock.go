package render

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/oxmonty/biscuit/internal/ir"
)

// Everything the generated mock server and smoke suite need is computed here,
// at generation time: each operation's canned response and the argument vector
// that drives its command. A generated repo therefore self-tests with no
// network, no fixtures, and no dependency on biscuit.

// mockMaxDepth bounds schema recursion. A $ref is a node in the IR, never
// inlined, so a cyclic spec is finite by construction — this bound only keeps
// the synthesized body small.
const mockMaxDepth = 6

// mockSynth synthesizes canned bodies against one spec's component schemas.
// The cache is what keeps a large spec cheap: stripe's 587 operations resolve
// the same few hundred $refs over and over, and a synthesized value is only
// ever marshaled, never mutated, so sharing one is safe.
type mockSynth struct {
	schemas map[string]*ir.Schema
	cache   map[string]any // keyed by ref name and remaining depth budget
}

func newMockSynth(schemas []ir.NamedSchema) *mockSynth {
	m := &mockSynth{schemas: make(map[string]*ir.Schema, len(schemas)), cache: map[string]any{}}
	for i := range schemas {
		m.schemas[schemas[i].Name] = schemas[i].Schema
	}
	return m
}

// mockResponse is the canned answer one operation's route serves: its first
// 2xx status, that response's content type, and a schema-valid body.
func (m *mockSynth) mockResponse(op *ir.Operation, sse bool) (status int, contentType, body string) {
	status = 200
	var content []ir.MediaType
	if op != nil {
		for _, r := range op.Responses {
			code, err := strconv.Atoi(r.Status)
			if err != nil || code < 200 || code > 299 {
				continue
			}
			status, content = code, r.Content
			break
		}
	}
	mt := pickMediaType(content, sse)
	if mt == nil {
		return status, "", ""
	}
	return status, mt.Type, m.synthesizeBody(mt.Schema)
}

// pickMediaType chooses which declared response body the mock serves: the
// event stream for an SSE operation, else JSON, else whatever the response
// declares first.
func pickMediaType(content []ir.MediaType, sse bool) *ir.MediaType {
	if len(content) == 0 {
		return nil
	}
	if sse {
		for i := range content {
			if strings.Contains(content[i].Type, "event-stream") {
				return &content[i]
			}
		}
	}
	for i := range content {
		if strings.Contains(content[i].Type, "json") {
			return &content[i]
		}
	}
	return &content[0]
}

func (m *mockSynth) synthesizeBody(s *ir.Schema) string {
	if s == nil {
		return ""
	}
	out, err := json.Marshal(m.synthesize(s, 0))
	if err != nil {
		return ""
	}
	return string(out)
}

// synthesize builds the smallest value satisfying s: a declared example or
// default when the spec offers one, else required properties only, with
// scalars taking their type's neutral value. Booleans coming out false and
// arrays holding one canned item is what makes a pagination walk terminate
// against the mock — no next-page signal ever says "more".
func (m *mockSynth) synthesize(s *ir.Schema, depth int) any {
	if s == nil || depth > mockMaxDepth {
		return map[string]any{}
	}
	if len(s.Examples) > 0 && json.Valid([]byte(s.Examples[0])) {
		return json.RawMessage(s.Examples[0])
	}
	if s.Default != "" && json.Valid([]byte(s.Default)) {
		return json.RawMessage(s.Default)
	}
	if s.Ref != "" {
		key := s.Ref + "\x00" + strconv.Itoa(depth)
		if v, ok := m.cache[key]; ok {
			return v
		}
		v := m.synthesize(m.schemas[s.Ref], depth+1)
		m.cache[key] = v
		return v
	}
	switch {
	case len(s.OneOf) > 0:
		return m.synthesize(s.OneOf[0], depth+1)
	case len(s.AnyOf) > 0:
		return m.synthesize(s.AnyOf[0], depth+1)
	case len(s.AllOf) > 0:
		return m.mergeAllOf(s.AllOf, depth)
	}
	switch s.Type {
	case "array":
		return []any{m.synthesize(s.Items, depth+1)}
	case "boolean":
		return false
	case "integer", "number":
		return 0
	case "null":
		return nil
	case "string":
		return synthesizeString(s)
	case "object":
		return m.synthesizeObject(s, depth)
	}
	if s.Items != nil {
		return []any{m.synthesize(s.Items, depth+1)}
	}
	return m.synthesizeObject(s, depth)
}

// mergeAllOf composes an allOf's branches into one object, since each branch
// contributes its own required properties. A branch that isn't an object wins
// outright — there is nothing to merge it into.
func (m *mockSynth) mergeAllOf(branches []*ir.Schema, depth int) any {
	out := map[string]any{}
	for _, b := range branches {
		merged, ok := m.synthesize(b, depth+1).(map[string]any)
		if !ok {
			return m.synthesize(b, depth+1)
		}
		for k, v := range merged {
			out[k] = v
		}
	}
	return out
}

func (m *mockSynth) synthesizeObject(s *ir.Schema, depth int) any {
	byName := make(map[string]*ir.Schema, len(s.Properties))
	for _, p := range s.Properties {
		byName[p.Name] = p.Schema
	}
	out := make(map[string]any, len(s.Required))
	for _, name := range s.Required {
		out[name] = m.synthesize(byName[name], depth+1)
	}
	return out
}

// synthesizeString picks a string a validator would accept: the first enum
// value when the schema constrains one, else a value shaped like the declared
// format, else a placeholder.
func synthesizeString(s *ir.Schema) any {
	if len(s.Enum) > 0 {
		if v, ok := jsonScalar(s.Enum[0]); ok {
			return v
		}
	}
	switch s.Format {
	case "date-time":
		return "2024-01-01T00:00:00Z"
	case "date":
		return "2024-01-01"
	case "time":
		return "00:00:00Z"
	case "duration":
		return "PT1S"
	case "uuid":
		return "00000000-0000-0000-0000-000000000000"
	case "email", "idn-email":
		return "user@example.com"
	case "uri", "url", "uri-reference", "iri":
		return "https://example.com"
	case "hostname", "idn-hostname":
		return "example.com"
	case "ipv4":
		return "127.0.0.1"
	case "ipv6":
		return "::1"
	case "byte":
		return "eA==" // base64 of "x"
	case "binary":
		return "x"
	}
	return "string"
}

func jsonScalar(encoded string) (any, bool) {
	var v any
	if err := json.Unmarshal([]byte(encoded), &v); err != nil {
		return nil, false
	}
	return v, true
}

// smokeArgs builds the argument vector the generated smoke suite drives one
// verb with: the command path, then every path parameter and required flag in
// --name=value form (pflag only accepts a boolean's value attached).
func smokeArgs(words []string, v *ir.Verb) []string {
	args := append([]string(nil), words...)
	for i := range v.Flags {
		f := &v.Flags[i]
		if !f.Required && f.In != "path" {
			continue
		}
		args = append(args, "--"+f.Name+"="+smokeValue(f))
	}
	return args
}

// smokeValue is a value the flag's type accepts. File-typed flags take
// @data:// rather than a temp file so the suite never touches the filesystem.
func smokeValue(f *ir.Flag) string {
	if len(f.Enum) > 0 {
		if v, ok := jsonScalar(f.Enum[0]); ok {
			return fmt.Sprint(v)
		}
	}
	switch f.Type {
	case "integer", "number":
		return "1"
	case "boolean":
		return "true"
	case "json":
		return "{}"
	}
	if f.Format == "binary" {
		return "@data://x"
	}
	return "x"
}

// routePath is the path the mock matches a request on. A spec path key can
// carry a query string (openai's "/responses?beta=true"); that never reaches
// the request's URL path, so the route template must not carry it either.
func routePath(path string) string {
	p, _, _ := strings.Cut(path, "?")
	return p
}

// smokeSkip reports why the generated suite can't drive this verb, empty when
// it can. Both shapes come from specs the mapping layer couldn't fully name;
// skipping keeps one malformed operation from failing an otherwise green
// suite, and the suite counts skips so they stay visible.
func smokeSkip(v *ir.Verb, pathFlags []*flagView) string {
	if param := unmappedPathParam(v.Path, pathFlags); param != "" {
		return "path parameter {" + param + "} has no flag to fill it"
	}
	for i := range v.Flags {
		if f := &v.Flags[i]; (f.Required || f.In == "path") && f.Name == "" {
			return "a required " + f.In + " parameter derived no flag name"
		}
	}
	return ""
}

// unmappedPathParam names the first {param} in path that no flag fills. Such
// an operation can't be driven generically — the request would carry the brace
// literal instead of a value.
func unmappedPathParam(path string, flags []*flagView) string {
	filled := make(map[string]bool, len(flags))
	for _, f := range flags {
		filled[f.Wire] = true
	}
	rest := path
	for {
		open := strings.Index(rest, "{")
		if open < 0 {
			return ""
		}
		closing := strings.Index(rest[open:], "}")
		if closing < 0 {
			return ""
		}
		if name := rest[open+1 : open+closing]; !filled[name] {
			return name
		}
		rest = rest[open+closing+1:]
	}
}
