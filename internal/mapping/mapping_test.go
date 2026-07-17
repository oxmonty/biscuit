package mapping

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/oxmonty/biscuit/internal/ir"
	"github.com/oxmonty/biscuit/internal/spec"
)

const ladder = "../../testdata/specs/"

func mustMap(t *testing.T, path string) *ir.API {
	t.Helper()
	doc, err := spec.Load(path)
	if err != nil {
		t.Fatalf("Load(%s): %v", path, err)
	}
	return Map(doc)
}

func TestMapPetstore(t *testing.T) {
	// given: the easy ladder spec
	api := mustMap(t, ladder+"petstore.yaml")

	// then: operations are sorted by (path, method)
	var got []string
	for _, op := range api.Operations {
		got = append(got, op.Method+" "+op.Path)
	}
	want := []string{"GET /pets", "POST /pets", "GET /pets/{petId}"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("operations = %v, want %v", got, want)
	}

	// then: component schemas are sorted by name
	var names []string
	for _, s := range api.Schemas {
		names = append(names, s.Name)
	}
	if !reflect.DeepEqual(names, []string{"Error", "Pet", "Pets"}) {
		t.Errorf("schemas = %v", names)
	}

	// then: the Pets array item is a Ref node, never inlined
	for _, s := range api.Schemas {
		if s.Name == "Pets" {
			if s.Schema.Items == nil || s.Schema.Items.Ref != "Pet" {
				t.Errorf("Pets.Items = %+v, want Ref to Pet", s.Schema.Items)
			}
		}
	}
}

func TestMapIsDeterministic(t *testing.T) {
	// given: the hard ladder spec mapped twice from separate loads
	a := mustMap(t, ladder+"openai.yaml")
	b := mustMap(t, ladder+"openai.yaml")

	// then: the IRs are byte-identical structures
	if !reflect.DeepEqual(a, b) {
		t.Error("two maps of the same spec differ")
	}
}

func TestMapCyclicSpecTerminates(t *testing.T) {
	// given: a spec with A→B→A and self-referencing schemas
	api := mustMap(t, ladder+"pathological/cyclic-refs.yaml")

	// then: mapping terminates and cycles survive as Ref nodes
	if len(api.Schemas) == 0 {
		t.Fatal("no schemas mapped")
	}
	for _, s := range api.Schemas {
		for _, p := range s.Schema.Properties {
			if p.Schema != nil && p.Schema.Ref == "" && len(p.Schema.Properties) > 0 {
				t.Errorf("schema %s property %s was inlined; want Ref indirection", s.Name, p.Name)
			}
		}
	}
}

func TestNormalizes30And31IntoOneShape(t *testing.T) {
	// given: the same nullable string-with-example modeled in 3.0 and 3.1 syntax
	v30 := `
openapi: 3.0.3
info: {title: t, version: "1"}
paths: {}
components:
  schemas:
    Name:
      type: string
      nullable: true
      example: "biscuit"
`
	v31 := `
openapi: 3.1.0
info: {title: t, version: "1"}
paths: {}
components:
  schemas:
    Name:
      type: [string, "null"]
      examples: ["biscuit"]
`
	dir := t.TempDir()
	var shapes []*ir.Schema
	for name, body := range map[string]string{"v30.yaml": v30, "v31.yaml": v31} {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		api := mustMap(t, path)
		shapes = append(shapes, api.Schemas[0].Schema)
	}

	// then: both normalize to type=string, nullable, examples=["biscuit"]
	for _, s := range shapes {
		if s.Type != "string" || !s.Nullable {
			t.Errorf("got type=%q nullable=%v, want string/true", s.Type, s.Nullable)
		}
		if !reflect.DeepEqual(s.Examples, []string{`"biscuit"`}) {
			t.Errorf("examples = %v, want [\"biscuit\"]", s.Examples)
		}
	}
	if !reflect.DeepEqual(shapes[0], shapes[1]) {
		t.Errorf("3.0 and 3.1 shapes differ:\n3.0: %+v\n3.1: %+v", shapes[0], shapes[1])
	}
}
