package mapping

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/oxmonty/biscuit/internal/ir"
)

func obj(props ...ir.Property) *ir.Schema {
	return &ir.Schema{Type: "object", Properties: props}
}

func str() *ir.Schema { return &ir.Schema{Type: "string"} }

func flagNames(flags []ir.Flag) []string {
	var out []string
	for _, f := range flags {
		out = append(out, f.Name)
	}
	return out
}

func TestFlagsFromParams(t *testing.T) {
	// given: path, query, and cookie parameters
	op := &ir.Operation{Method: "GET", Path: "/users/{userId}", Params: []ir.Param{
		{Name: "userId", In: "path", Schema: str()},
		{Name: "page_size", In: "query", Schema: &ir.Schema{Type: "integer"}},
		{Name: "session", In: "cookie", Schema: str()},
	}}

	// then: path params are required, names kebab-cased, cookies deferred
	flags := flagsFor(op, nil)
	if got := flagNames(flags); !reflect.DeepEqual(got, []string{"page-size", "user-id"}) {
		t.Fatalf("flags = %v", got)
	}
	if !flags[1].Required || flags[1].In != "path" {
		t.Errorf("user-id = %+v, want required path flag", flags[1])
	}
	if flags[0].Type != "integer" || flags[0].Required {
		t.Errorf("page-size = %+v", flags[0])
	}
}

func TestFlagsBodyDotNotation(t *testing.T) {
	// given: a small nested body schema (the PRD's --name.first example)
	op := &ir.Operation{Method: "POST", Path: "/users", RequestBody: []ir.MediaType{
		{Type: "application/json", Schema: obj(
			ir.Property{Name: "name", Schema: obj(
				ir.Property{Name: "first", Schema: str()},
				ir.Property{Name: "last", Schema: str()},
			)},
			ir.Property{Name: "address", Schema: obj(
				ir.Property{Name: "city", Schema: str()},
			)},
		)},
	}}

	// then: nested objects expand into dot notation with original paths kept
	flags := flagsFor(op, nil)
	want := []string{"address.city", "name.first", "name.last"}
	if got := flagNames(flags); !reflect.DeepEqual(got, want) {
		t.Fatalf("flags = %v, want %v", got, want)
	}
	if !reflect.DeepEqual(flags[0].BodyPath, []string{"address", "city"}) {
		t.Errorf("BodyPath = %v", flags[0].BodyPath)
	}
}

func TestFlagsRequiredPropagation(t *testing.T) {
	// given: name required on the body, name.first required in name
	body := obj(
		ir.Property{Name: "name", Schema: &ir.Schema{Type: "object",
			Properties: []ir.Property{{Name: "first", Schema: str()}, {Name: "nick", Schema: str()}},
			Required:   []string{"first"}}},
		ir.Property{Name: "note", Schema: str()},
	)
	body.Required = []string{"name"}
	op := &ir.Operation{Method: "POST", Path: "/users", RequestBody: []ir.MediaType{{Type: "application/json", Schema: body}}}

	// then: required holds only where the whole ancestor chain is required
	byName := map[string]ir.Flag{}
	for _, f := range flagsFor(op, nil) {
		byName[f.Name] = f
	}
	if !byName["name.first"].Required {
		t.Error("name.first should be required")
	}
	if byName["name.nick"].Required || byName["note"].Required {
		t.Error("optional flags marked required")
	}
}

func TestFlagsArrays(t *testing.T) {
	// given: arrays of scalars and of objects
	op := &ir.Operation{Method: "POST", Path: "/things", RequestBody: []ir.MediaType{
		{Type: "application/json", Schema: obj(
			ir.Property{Name: "tags", Schema: &ir.Schema{Type: "array", Items: str()}},
			ir.Property{Name: "messages", Schema: &ir.Schema{Type: "array", Items: obj(
				ir.Property{Name: "role", Schema: str()},
			)}},
		)},
	}}

	// then: scalar items repeat as strings, object items repeat as json
	byName := map[string]ir.Flag{}
	for _, f := range flagsFor(op, nil) {
		byName[f.Name] = f
	}
	if f := byName["tags"]; !f.Repeated || f.Type != "string" {
		t.Errorf("tags = %+v", f)
	}
	if f := byName["messages"]; !f.Repeated || f.Type != "json" {
		t.Errorf("messages = %+v", f)
	}
}

func TestFlagsCycleFallsBackToJSON(t *testing.T) {
	// given: a self-referencing schema reached from the body
	schemas := map[string]*ir.Schema{
		"Node": obj(
			ir.Property{Name: "value", Schema: str()},
			ir.Property{Name: "child", Schema: &ir.Schema{Ref: "Node"}},
		),
	}
	op := &ir.Operation{Method: "POST", Path: "/trees", RequestBody: []ir.MediaType{
		{Type: "application/json", Schema: &ir.Schema{Ref: "Node"}},
	}}

	// then: expansion terminates; the cyclic subtree is a json flag
	byName := map[string]ir.Flag{}
	for _, f := range flagsFor(op, schemas) {
		byName[f.Name] = f
	}
	if f := byName["child"]; f.Type != "json" {
		t.Errorf("child = %+v, want json fallback", f)
	}
	if f := byName["value"]; f.Type != "string" {
		t.Errorf("value = %+v", f)
	}
}

func TestFlagsAdaptiveDepthCapsExplosion(t *testing.T) {
	// given: 10 top-level objects × 10 props each (100 leaves > the 64 budget)
	var props []ir.Property
	for i := 0; i < 10; i++ {
		var inner []ir.Property
		for j := 0; j < 10; j++ {
			inner = append(inner, ir.Property{Name: fmt.Sprintf("f%02d", j), Schema: str()})
		}
		props = append(props, ir.Property{Name: fmt.Sprintf("group%02d", i), Schema: obj(inner...)})
	}
	op := &ir.Operation{Method: "POST", Path: "/big", RequestBody: []ir.MediaType{
		{Type: "application/json", Schema: obj(props...)},
	}}

	// then: depth adapts down to 1 — ten json flags, no dot expansion
	flags := flagsFor(op, nil)
	if len(flags) != 10 {
		t.Fatalf("len = %d, want 10", len(flags))
	}
	for _, f := range flags {
		if f.Type != "json" {
			t.Errorf("%s = %s, want json", f.Name, f.Type)
		}
	}
}

func TestFlagsOneOfIsJSONUntilCascade(t *testing.T) {
	// given: a oneOf body property
	op := &ir.Operation{Method: "POST", Path: "/poly", RequestBody: []ir.MediaType{
		{Type: "application/json", Schema: obj(
			ir.Property{Name: "source", Schema: &ir.Schema{OneOf: []*ir.Schema{obj(), obj()}}},
		)},
	}}

	// then: it stays one json flag (the discriminator cascade refines this)
	flags := flagsFor(op, nil)
	if len(flags) != 1 || flags[0].Type != "json" {
		t.Fatalf("flags = %+v", flags)
	}
}

func TestFlagsAllOfMerges(t *testing.T) {
	// given: a body of allOf members with distinct properties
	schemas := map[string]*ir.Schema{
		"Base": obj(ir.Property{Name: "id", Schema: str()}),
	}
	body := &ir.Schema{AllOf: []*ir.Schema{
		{Ref: "Base"},
		obj(ir.Property{Name: "name", Schema: str()}),
	}}
	op := &ir.Operation{Method: "POST", Path: "/merged", RequestBody: []ir.MediaType{
		{Type: "application/json", Schema: body},
	}}

	// then: member properties merge into one flag set
	want := []string{"id", "name"}
	if got := flagNames(flagsFor(op, schemas)); !reflect.DeepEqual(got, want) {
		t.Fatalf("flags = %v, want %v", got, want)
	}
}

func TestFlagsParamBodyCollision(t *testing.T) {
	// given: a query param and a body property both named filter
	op := &ir.Operation{Method: "POST", Path: "/search", Params: []ir.Param{
		{Name: "filter", In: "query", Schema: str()},
	}, RequestBody: []ir.MediaType{
		{Type: "application/json", Schema: obj(ir.Property{Name: "filter", Schema: str()})},
	}}

	// then: the body flag keeps a body. prefix, deterministically
	want := []string{"body.filter", "filter"}
	if got := flagNames(flagsFor(op, nil)); !reflect.DeepEqual(got, want) {
		t.Fatalf("flags = %v, want %v", got, want)
	}
}

func TestFlagsScalarBodyIsSingleBodyFlag(t *testing.T) {
	// given: a request body that is a bare string
	op := &ir.Operation{Method: "POST", Path: "/raw", RequestBody: []ir.MediaType{
		{Type: "application/json", Schema: str()},
	}}

	// then: it maps to one required --body flag
	flags := flagsFor(op, nil)
	if len(flags) != 1 || flags[0].Name != "body" || !flags[0].Required {
		t.Fatalf("flags = %+v", flags)
	}
}

func TestFlagsLadderSmoke(t *testing.T) {
	// given: openai's chat completions create operation
	api := mustMap(t, ladder+"openai.yaml")
	var verb *ir.Verb
	for _, c := range api.Commands {
		if c.Name != "chat" {
			continue
		}
		for _, ch := range c.Children {
			if ch.Name != "completions" {
				continue
			}
			for i := range ch.Verbs {
				if ch.Verbs[i].Name == "create" {
					verb = &ch.Verbs[i]
				}
			}
		}
	}
	if verb == nil {
		t.Fatal("chat completions create not found")
	}

	// then: messages is a repeated json flag and model a required scalar-ish flag
	byName := map[string]ir.Flag{}
	for _, f := range verb.Flags {
		byName[f.Name] = f
	}
	if f, ok := byName["messages"]; !ok || !f.Repeated || f.Type != "json" || !f.Required {
		t.Errorf("messages = %+v", f)
	}
	if f, ok := byName["model"]; !ok || !f.Required {
		t.Errorf("model = %+v", f)
	}
	if len(verb.Flags) > maxFlagsPerOp {
		t.Errorf("flag count %d exceeds budget", len(verb.Flags))
	}
}
