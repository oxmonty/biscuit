package mapping

import (
	"reflect"
	"testing"

	"github.com/oxmonty/biscuit/internal/ir"
)

func unionOf(t *testing.T, schemas map[string]*ir.Schema, s *ir.Schema) *ir.Union {
	t.Helper()
	op := &ir.Operation{Method: "POST", Path: "/u", RequestBody: []ir.MediaType{
		{Type: "application/json", Schema: obj(ir.Property{Name: "value", Schema: s})},
	}}
	for _, f := range flagsFor(op, schemas) {
		if f.Name == "value" {
			return f.Union
		}
	}
	t.Fatal("value flag not found")
	return nil
}

func TestCascadeExplicitDiscriminator(t *testing.T) {
	// given: a oneOf with a spec-declared discriminator and mapping
	u := unionOf(t, nil, &ir.Schema{
		OneOf: []*ir.Schema{{Ref: "Card"}, {Ref: "Bank"}},
		Discriminator: &ir.Discriminator{PropertyName: "type", Mapping: []ir.MappingEntry{
			{Value: "card", Schema: "Card"},
			{Value: "bank", Schema: "Bank"},
		}},
	})

	// then: the explicit discriminator wins the cascade
	want := &ir.Union{Kind: "discriminator", Property: "type", Variants: []ir.UnionVariant{
		{Value: "bank", Schema: "Bank"}, {Value: "card", Schema: "Card"},
	}}
	if !reflect.DeepEqual(u, want) {
		t.Errorf("union = %+v, want %+v", u, want)
	}
}

func TestCascadeUniqueField(t *testing.T) {
	// given: variants each carrying a property the other lacks
	schemas := map[string]*ir.Schema{
		"Email": obj(ir.Property{Name: "email", Schema: str()}, ir.Property{Name: "note", Schema: str()}),
		"Phone": obj(ir.Property{Name: "phone", Schema: str()}, ir.Property{Name: "note", Schema: str()}),
	}
	u := unionOf(t, schemas, &ir.Schema{OneOf: []*ir.Schema{{Ref: "Email"}, {Ref: "Phone"}}})

	// then: unique fields identify the variants
	want := &ir.Union{Kind: "unique-field", Variants: []ir.UnionVariant{
		{Value: "email", Schema: "Email"}, {Value: "phone", Schema: "Phone"},
	}}
	if !reflect.DeepEqual(u, want) {
		t.Errorf("union = %+v, want %+v", u, want)
	}
}

func TestCascadeJSONType(t *testing.T) {
	// given: openai's string-or-array shape
	u := unionOf(t, nil, &ir.Schema{OneOf: []*ir.Schema{
		str(),
		{Type: "array", Items: str()},
	}})

	// then: distinct JSON types discriminate
	want := &ir.Union{Kind: "json-type", Variants: []ir.UnionVariant{
		{Value: "array"}, {Value: "string"},
	}}
	if !reflect.DeepEqual(u, want) {
		t.Errorf("union = %+v, want %+v", u, want)
	}
}

func TestCascadeEnumValue(t *testing.T) {
	// given: variants sharing a const-valued type property, no declared
	// discriminator, same remaining shape (so unique-field can't fire)
	schemas := map[string]*ir.Schema{
		"A": obj(
			ir.Property{Name: "kind", Schema: &ir.Schema{Type: "string", Enum: []string{`"a"`}}},
			ir.Property{Name: "value", Schema: str()},
		),
		"B": obj(
			ir.Property{Name: "kind", Schema: &ir.Schema{Type: "string", Enum: []string{`"b"`}}},
			ir.Property{Name: "value", Schema: str()},
		),
	}
	u := unionOf(t, schemas, &ir.Schema{OneOf: []*ir.Schema{{Ref: "A"}, {Ref: "B"}}})

	// then: the shared const property discriminates by value
	want := &ir.Union{Kind: "enum-value", Property: "kind", Variants: []ir.UnionVariant{
		{Value: `"a"`, Schema: "A"}, {Value: `"b"`, Schema: "B"},
	}}
	if !reflect.DeepEqual(u, want) {
		t.Errorf("union = %+v, want %+v", u, want)
	}
}

func TestCascadeOpaque(t *testing.T) {
	// given: two identical object variants — nothing tells them apart
	schemas := map[string]*ir.Schema{
		"X": obj(ir.Property{Name: "v", Schema: str()}),
		"Y": obj(ir.Property{Name: "v", Schema: str()}),
	}
	u := unionOf(t, schemas, &ir.Schema{OneOf: []*ir.Schema{{Ref: "X"}, {Ref: "Y"}}})

	// then: the cascade bottoms out at opaque, never crashes
	if u == nil || u.Kind != "opaque" {
		t.Errorf("union = %+v, want opaque", u)
	}
}

func TestCascadeLadderSmoke(t *testing.T) {
	// given: openai, whose request bodies carry oneOf throughout (stripe's
	// polymorphism sits in responses, which flags don't map)
	api := mustMap(t, ladder+"openai.yaml")

	// then: the cascade resolves a meaningful share of oneOf flags
	kinds := map[string]int{}
	var walk func([]ir.Command)
	count := func(verbs []ir.Verb) {
		for _, v := range verbs {
			for _, f := range v.Flags {
				if f.Union != nil {
					kinds[f.Union.Kind]++
				}
			}
		}
	}
	walk = func(cmds []ir.Command) {
		for _, c := range cmds {
			count(c.Verbs)
			walk(c.Children)
		}
	}
	walk(api.Commands)
	total, opaque := 0, kinds["opaque"]
	for _, n := range kinds {
		total += n
	}
	if total == 0 {
		t.Fatal("no unions found on stripe — the cascade never ran")
	}
	if opaque == total {
		t.Errorf("all %d unions opaque — no cascade step ever fired: %v", total, kinds)
	}
	t.Logf("union kinds: %v", kinds)
}
