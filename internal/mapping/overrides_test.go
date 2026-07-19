package mapping

import (
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/oxmonty/biscuit/internal/ir"
)

func writeFile(path, content string) error {
	return os.WriteFile(path, []byte(content), 0o644)
}

func deriveWith(ops []ir.Operation, overrides map[string]ir.Override) *ir.API {
	api := &ir.API{Operations: ops}
	sortOperations(api.Operations)
	deriveCommands(api, overrides)
	return api
}

func TestOverrideSidecarRenameGroupAliases(t *testing.T) {
	// given: sidecar overrides keyed by operationId
	api := deriveWith(
		[]ir.Operation{{ID: "listUsers", Method: "GET", Path: "/users"}},
		map[string]ir.Override{"listUsers": {
			Name: "ls", Group: "admin accounts", Aliases: []string{"listAll"},
		}},
	)

	// then: the verb is renamed, regrouped, and kebab-cased aliases attach
	if got := flatten(api.Commands, ""); !reflect.DeepEqual(got, []string{"admin accounts ls"}) {
		t.Fatalf("commands = %v", got)
	}
	verb := api.Commands[0].Children[0].Verbs[0]
	if !reflect.DeepEqual(verb.Aliases, []string{"list-all"}) {
		t.Errorf("aliases = %v", verb.Aliases)
	}
}

func TestOverrideIgnoreDropsOperation(t *testing.T) {
	// given: an internal endpoint ignored via sidecar
	api := deriveWith(
		[]ir.Operation{
			{ID: "listUsers", Method: "GET", Path: "/users"},
			{ID: "debugDump", Method: "GET", Path: "/users/debug"},
		},
		map[string]ir.Override{"debugDump": {Ignore: true}},
	)

	// then: only the surviving operation appears
	if got := flatten(api.Commands, ""); !reflect.DeepEqual(got, []string{"users list"}) {
		t.Errorf("commands = %v", got)
	}
}

func TestOverrideExtensionAndSidecarPrecedence(t *testing.T) {
	// given: an x-biscuit-name in-spec and a sidecar rename for the same op
	api := deriveWith(
		[]ir.Operation{{
			ID: "listUsers", Method: "GET", Path: "/users",
			XBiscuit: ir.Override{Name: "from-extension", Pagination: "cursor"},
		}},
		map[string]ir.Override{"listUsers": {Name: "from-sidecar"}},
	)

	// then: the sidecar name wins; the extension pagination hint survives
	verb := api.Commands[0].Verbs[0]
	if verb.Name != "from-sidecar" {
		t.Errorf("name = %q, want from-sidecar", verb.Name)
	}
	if verb.Pagination != "cursor" {
		t.Errorf("pagination = %q, want cursor (extension value untouched)", verb.Pagination)
	}
}

func TestOverrideMethodPathKeyForAnonymousOps(t *testing.T) {
	// given: an operation with no operationId, keyed by "METHOD /path"
	api := deriveWith(
		[]ir.Operation{{Method: "GET", Path: "/users"}},
		map[string]ir.Override{"GET /users": {Name: "everyone"}},
	)

	// then: the override lands
	if got := flatten(api.Commands, ""); !reflect.DeepEqual(got, []string{"users everyone"}) {
		t.Errorf("commands = %v", got)
	}
}

func TestOverrideUnmatchedKeyDiagnostic(t *testing.T) {
	// given: an override key matching nothing in the spec
	api := deriveWith(
		[]ir.Operation{{ID: "listUsers", Method: "GET", Path: "/users"}},
		map[string]ir.Override{"nosuchOp": {Name: "x"}},
	)

	// then: a diagnostic names the dangling key
	if len(api.Diagnostics) != 1 || !strings.Contains(api.Diagnostics[0], `"nosuchOp"`) {
		t.Errorf("diagnostics = %v", api.Diagnostics)
	}
}

func TestOverrideExtensionFromSpecFile(t *testing.T) {
	// given: a spec carrying x-biscuit-* extensions on an operation
	dir := t.TempDir()
	specYAML := `openapi: 3.0.3
info: {title: Ext, version: 1.0.0}
paths:
  /users:
    get:
      operationId: listUsers
      x-biscuit-name: everyone
      x-biscuit-group: admin
      x-biscuit-pagination: cursor
      responses:
        '200': {description: OK}
  /internal/dump:
    get:
      operationId: dump
      x-biscuit-ignore: true
      responses:
        '200': {description: OK}
`
	path := dir + "/openapi.yaml"
	if err := writeFile(path, specYAML); err != nil {
		t.Fatal(err)
	}
	api := mustMap(t, path)

	// then: the extensions shape the tree without any sidecar
	if got := flatten(api.Commands, ""); !reflect.DeepEqual(got, []string{"admin everyone"}) {
		t.Fatalf("commands = %v", got)
	}
	if api.Commands[0].Verbs[0].Pagination != "cursor" {
		t.Errorf("pagination = %q", api.Commands[0].Verbs[0].Pagination)
	}
}
