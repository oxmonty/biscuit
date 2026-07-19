package mapping

import (
	"reflect"
	"testing"

	"github.com/oxmonty/biscuit/internal/ir"
)

func derive(ops []ir.Operation, tags ...ir.Tag) *ir.API {
	api := &ir.API{Operations: ops, Tags: tags}
	sortOperations(api.Operations)
	deriveCommands(api)
	return api
}

func flatten(cmds []ir.Command, prefix string) []string {
	var out []string
	for _, c := range cmds {
		name := prefix + c.Name
		for _, v := range c.Verbs {
			out = append(out, name+" "+v.Name)
		}
		out = append(out, flatten(c.Children, name+" ")...)
	}
	return out
}

func TestDeriveNestedSubResources(t *testing.T) {
	// given: the PRD's nested example path
	api := derive([]ir.Operation{
		{Method: "GET", Path: "/orgs/{org}/repos/{repo}/issues"},
	})

	// then: static segments nest — orgs repos issues list
	got := flatten(api.Commands, "")
	if !reflect.DeepEqual(got, []string{"orgs repos issues list"}) {
		t.Errorf("commands = %v", got)
	}
}

func TestDeriveStutterRemoval(t *testing.T) {
	// given: a users tag with a list-users operation (the PRD's example)
	api := derive(
		[]ir.Operation{{ID: "list-users", Method: "GET", Path: "/users", Tags: []string{"users"}}},
		ir.Tag{Name: "users", Description: "Manage users"},
	)

	// then: the verb is list, not list-users, and the tag description lands
	if got := flatten(api.Commands, ""); !reflect.DeepEqual(got, []string{"users list"}) {
		t.Errorf("commands = %v", got)
	}
	if api.Commands[0].Description != "Manage users" {
		t.Errorf("description = %q", api.Commands[0].Description)
	}
}

func TestDeriveStripsVersionPrefixAndKebabs(t *testing.T) {
	// given: a versioned snake_case path with a camelCase operationId
	api := derive([]ir.Operation{
		{ID: "createPaymentIntent", Method: "POST", Path: "/v1/payment_intents"},
	})

	// then: v1 is stripped, names kebab-cased, stutter (incl. singular) removed
	if got := flatten(api.Commands, ""); !reflect.DeepEqual(got, []string{"payment-intents create"}) {
		t.Errorf("commands = %v", got)
	}
}

func TestDeriveCustomAction(t *testing.T) {
	// given: Stripe's POST-only custom-action shape next to normal CRUD
	api := derive([]ir.Operation{
		{Method: "GET", Path: "/v1/payment_intents"},
		{Method: "GET", Path: "/v1/payment_intents/{intent}"},
		{Method: "POST", Path: "/v1/payment_intents/{intent}/confirm"},
	})

	// then: confirm is a verb on payment-intents, not a sub-resource
	want := []string{"payment-intents confirm", "payment-intents get", "payment-intents list"}
	if got := flatten(api.Commands, ""); !reflect.DeepEqual(got, want) {
		t.Errorf("commands = %v, want %v", got, want)
	}
}

func TestDeriveCollectionAfterParamStaysResource(t *testing.T) {
	// given: a GET+POST collection nested under a param (not POST-only)
	api := derive([]ir.Operation{
		{Method: "GET", Path: "/users/{id}/keys"},
		{Method: "POST", Path: "/users/{id}/keys"},
	})

	// then: keys is a sub-resource with shape-derived verbs
	want := []string{"users keys create", "users keys list"}
	if got := flatten(api.Commands, ""); !reflect.DeepEqual(got, want) {
		t.Errorf("commands = %v, want %v", got, want)
	}
}

func TestDerivePostOnInstanceIsUpdate(t *testing.T) {
	// given: Stripe's update convention — POST to the instance, no PUT/PATCH
	api := derive([]ir.Operation{
		{Method: "POST", Path: "/v1/accounts"},
		{Method: "POST", Path: "/v1/accounts/{account}"},
	})

	// then: instance POST maps to update, no collision
	want := []string{"accounts create", "accounts update"}
	if got := flatten(api.Commands, ""); !reflect.DeepEqual(got, want) {
		t.Errorf("commands = %v, want %v", got, want)
	}
	if len(api.Diagnostics) != 0 {
		t.Errorf("diagnostics = %v, want none", api.Diagnostics)
	}
}

func TestDerivePluralLeafStaysResource(t *testing.T) {
	// given: a create-only plural sub-collection in action position
	api := derive([]ir.Operation{
		{Method: "POST", Path: "/v1/accounts/{account}/login_links"},
	})

	// then: login-links is a sub-resource with create, not an action verb
	want := []string{"accounts login-links create"}
	if got := flatten(api.Commands, ""); !reflect.DeepEqual(got, want) {
		t.Errorf("commands = %v, want %v", got, want)
	}
}

func TestDeriveShapeVerbsWithoutOperationID(t *testing.T) {
	// given: bare CRUD paths with no operationIds
	api := derive([]ir.Operation{
		{Method: "GET", Path: "/users"},
		{Method: "POST", Path: "/users"},
		{Method: "GET", Path: "/users/{id}"},
		{Method: "PATCH", Path: "/users/{id}"},
		{Method: "DELETE", Path: "/users/{id}"},
	})

	// then: verbs come from method + path shape
	want := []string{"users create", "users delete", "users get", "users list", "users update"}
	if got := flatten(api.Commands, ""); !reflect.DeepEqual(got, want) {
		t.Errorf("commands = %v, want %v", got, want)
	}
}

func TestDeriveBareHTTPVerbIdPrefersShape(t *testing.T) {
	// given: an operationId that reduces to a bare "get" on a collection path
	api := derive([]ir.Operation{
		{ID: "GetCustomersCustomerSubscriptions", Method: "GET", Path: "/v1/customers/{customer}/subscriptions"},
	})

	// then: the shape verb list wins over the reduced id
	want := []string{"customers subscriptions list"}
	if got := flatten(api.Commands, ""); !reflect.DeepEqual(got, want) {
		t.Errorf("commands = %v, want %v", got, want)
	}
}

func TestDeriveCollisionRenamesDeterministically(t *testing.T) {
	// given: two paths collapsing to the same chain and shape verb
	api := derive([]ir.Operation{
		{Method: "GET", Path: "/users/settings"},
		{Method: "GET", Path: "/users/{id}/settings"},
	})

	// then: the later (path,method)-sorted op gets a method suffix + diagnostic
	want := []string{"users settings list", "users settings list-get"}
	if got := flatten(api.Commands, ""); !reflect.DeepEqual(got, want) {
		t.Errorf("commands = %v, want %v", got, want)
	}
	if len(api.Diagnostics) != 1 {
		t.Errorf("diagnostics = %v, want one collision warning", api.Diagnostics)
	}
}

func TestDeriveTagFallbackForRootPaths(t *testing.T) {
	// given: a path with no static segments but a tag
	api := derive([]ir.Operation{
		{Method: "GET", Path: "/", Tags: []string{"Meta"}},
	})

	// then: the tag groups it; untagged root ops would land in RootVerbs
	if got := flatten(api.Commands, ""); !reflect.DeepEqual(got, []string{"meta list"}) {
		t.Errorf("commands = %v", got)
	}
}

func TestDeriveRootVerbs(t *testing.T) {
	// given: an untagged operation on /
	api := derive([]ir.Operation{
		{Method: "GET", Path: "/"},
	})

	// then: it survives as a root verb, not dropped
	if len(api.RootVerbs) != 1 || api.RootVerbs[0].Name != "list" {
		t.Errorf("root verbs = %+v", api.RootVerbs)
	}
}

func TestDeriveLadderSmoke(t *testing.T) {
	// given: the easy ladder spec
	api := mustMap(t, ladder+"petstore.yaml")

	// then: a pets resource exists; showPetById sheds pet/by/id stutter
	got := flatten(api.Commands, "")
	want := []string{"pets create", "pets list", "pets show"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("commands = %v, want %v", got, want)
	}
}

func TestDeriveIsDeterministic(t *testing.T) {
	// given: the hard ladder spec mapped twice
	a := mustMap(t, ladder+"openai.yaml")
	b := mustMap(t, ladder+"openai.yaml")

	// then: trees and diagnostics are identical
	if !reflect.DeepEqual(a.Commands, b.Commands) || !reflect.DeepEqual(a.Diagnostics, b.Diagnostics) {
		t.Error("two derivations of the same spec differ")
	}
}

func TestKebab(t *testing.T) {
	// given/then: kebab handles snake, camel, and acronym runs
	cases := map[string]string{
		"payment_intents": "payment-intents",
		"listUsers":       "list-users",
		"HTTPProxy":       "http-proxy",
		"Chat":            "chat",
		"fine_tuning.jobs": "fine-tuning-jobs",
	}
	for in, want := range cases {
		if got := kebab(in); got != want {
			t.Errorf("kebab(%q) = %q, want %q", in, got, want)
		}
	}
}
