package mapping

import (
	"strings"
	"testing"

	"github.com/oxmonty/biscuit/internal/config"
	"github.com/oxmonty/biscuit/internal/ir"
)

func query(names ...string) []ir.Param {
	var ps []ir.Param
	for _, n := range names {
		ps = append(ps, ir.Param{Name: n, In: "query", Schema: &ir.Schema{Type: "string"}})
	}
	return ps
}

// jsonResponse builds a 200 whose body is an object with the named fields:
// a name suffixed "[]" is an array, everything else a scalar. Nested paths
// ("meta.next") create the intermediate object.
func jsonResponse(fields ...string) []ir.Response {
	root := &ir.Schema{Type: "object"}
	for _, f := range fields {
		cur := root
		name := f
		for {
			seg, rest, nested := strings.Cut(name, ".")
			if !nested {
				break
			}
			child := findProp(cur, seg)
			if child == nil {
				child = &ir.Schema{Type: "object"}
				cur.Properties = append(cur.Properties, ir.Property{Name: seg, Schema: child})
			}
			cur, name = child, rest
		}
		leaf := &ir.Schema{Type: "string"}
		if n, isArray := strings.CutSuffix(name, "[]"); isArray {
			name, leaf = n, &ir.Schema{Type: "array", Items: &ir.Schema{Type: "object"}}
		}
		cur.Properties = append(cur.Properties, ir.Property{Name: name, Schema: leaf})
	}
	return []ir.Response{{
		Status:  "200",
		Content: []ir.MediaType{{Type: "application/json", Schema: root}},
	}}
}

func findProp(s *ir.Schema, name string) *ir.Schema {
	for _, p := range s.Properties {
		if p.Name == name {
			return p.Schema
		}
	}
	return nil
}

func match(t *testing.T, op ir.Operation, override string, cfg *config.Config) *ir.Pagination {
	t.Helper()
	op.Method, op.Path = "GET", "/things"
	p, _ := resolvePagination(&op, override, schemesFor(cfg), &flattener{})
	return p
}

func TestPaginationMatchesConventions(t *testing.T) {
	cases := []struct {
		name   string
		op     ir.Operation
		want   ir.Pagination
	}{
		{
			// given: Stripe's shape — a starting_after cursor and has_more
			name: "stripe cursor",
			op:   ir.Operation{Params: query("starting_after", "ending_before", "limit"), Responses: jsonResponse("data[]", "has_more")},
			want: ir.Pagination{Scheme: "cursor-has-more", Type: "cursor", Param: "starting_after", LimitParam: "limit", ItemsPath: "data", NextPath: "has_more", NextKind: "has_more"},
		},
		{
			// given: AIP-158's shape — pageToken in, nextPageToken out
			name: "google page token",
			op:   ir.Operation{Params: query("pageToken", "pageSize"), Responses: jsonResponse("items[]", "nextPageToken")},
			want: ir.Pagination{Scheme: "page-token", Type: "cursor", Param: "pageToken", LimitParam: "pageSize", ItemsPath: "items", NextPath: "nextPageToken", NextKind: "cursor"},
		},
		{
			// given: Slack's shape — the next cursor nested one level down
			name: "slack nested cursor",
			op:   ir.Operation{Params: query("cursor", "limit"), Responses: jsonResponse("members[]", "response_metadata.next_cursor")},
			want: ir.Pagination{Scheme: "cursor-next-cursor", Type: "cursor", Param: "cursor", LimitParam: "limit", ItemsPath: "members", NextPath: "response_metadata.next_cursor", NextKind: "cursor"},
		},
		{
			// given: a GitHub-style page walk whose only next signal is a
			// documented Link response header, over a bare array body
			name: "link header",
			op: ir.Operation{
				Params: query("page", "per_page"),
				Responses: []ir.Response{{
					Status:  "200",
					Headers: []string{"Link", "X-RateLimit-Remaining"},
					Content: []ir.MediaType{{Type: "application/json", Schema: &ir.Schema{Type: "array", Items: &ir.Schema{Type: "object"}}}},
				}},
			},
			want: ir.Pagination{Scheme: "link-header", Type: "page", Param: "page", LimitParam: "per_page", ItemsPath: "@this", NextKind: "link_header"},
		},
		{
			// given: an offset walk corroborated only by a total, which steps
			// arithmetically instead of following a link
			name: "offset with total",
			op:   ir.Operation{Params: query("offset", "limit"), Responses: jsonResponse("results[]", "total")},
			want: ir.Pagination{Scheme: "offset-limit", Type: "offset", Param: "offset", LimitParam: "limit", ItemsPath: "results", NextKind: "step"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// when: matching the operation against the built-in library
			got := match(t, tc.op, "", nil)

			// then: the resolved walk names the scheme and its wire details
			if got == nil {
				t.Fatalf("no pagination matched, want %+v", tc.want)
			}
			if *got != tc.want {
				t.Errorf("pagination = %+v, want %+v", *got, tc.want)
			}
		})
	}
}

func TestPaginationRejects(t *testing.T) {
	cases := []struct {
		name string
		op   ir.Operation
	}{
		// limit alone is a batch size, not pagination — it appears on
		// endpoints that fetch exactly one page
		{"limit only", ir.Operation{Params: query("limit"), Responses: jsonResponse("data[]", "has_more")}},
		// two arrays and no declared items_field: which one is the page?
		{"two array fields", ir.Operation{Params: query("after", "limit"), Responses: jsonResponse("data[]", "extras[]", "has_more")}},
		// both cursor dialects answer to this response; guessing one would
		// be a coin flip, so neither applies
		{"ambiguous schemes", ir.Operation{Params: query("cursor", "limit"), Responses: jsonResponse("data[]", "has_more", "next_cursor")}},
		// a page param with no next-page signal anywhere: nothing says when
		// to stop (museum.yaml's real shape)
		{"no next signal", ir.Operation{Params: query("page", "limit"), Responses: jsonResponse("data[]")}},
		// a substring hit, never a match: "next_page_offset" is not "offset"
		{"substring only", ir.Operation{Params: query("next_page_offset", "limit"), Responses: jsonResponse("data[]", "total")}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// when: matching against the built-in library
			// then: the operation fetches one page
			if got := match(t, tc.op, "", nil); got != nil {
				t.Errorf("pagination = %+v, want none", *got)
			}
		})
	}
}

func TestPaginationOverrideOff(t *testing.T) {
	// given: an operation the built-in library would otherwise claim
	op := ir.Operation{Params: query("starting_after", "limit"), Responses: jsonResponse("data[]", "has_more")}
	if match(t, op, "", nil) == nil {
		t.Fatal("fixture no longer matches; the override case proves nothing")
	}

	// when: the per-operation override turns pagination off
	// then: the verb fetches one page
	if got := match(t, op, "off", nil); got != nil {
		t.Errorf("pagination = %+v, want none (override off)", *got)
	}
}

func TestPaginationDeclaredSchemeShadowsBuiltin(t *testing.T) {
	// given: a declared scheme reusing a built-in's name with different wiring
	cfg := &config.Config{Pagination: []config.PaginationScheme{{
		Name:     "cursor-has-more",
		Type:     "cursor",
		Request:  config.PaginationRequest{CursorParam: "since_id", LimitParam: "count"},
		Response: config.PaginationResponse{ItemsField: "envelope.rows", NextField: "envelope.more", NextKind: "has_more"},
	}}}

	// when: matching an operation shaped for the declared scheme
	op := ir.Operation{Params: query("since_id", "count"), Responses: jsonResponse("envelope.rows[]", "envelope.more")}
	got := match(t, op, "", cfg)

	// then: the declared wiring wins, nested paths and all
	want := ir.Pagination{Scheme: "cursor-has-more", Type: "cursor", Param: "since_id", LimitParam: "count", ItemsPath: "envelope.rows", NextPath: "envelope.more", NextKind: "has_more"}
	if got == nil {
		t.Fatalf("no pagination matched, want %+v", want)
	}
	if *got != want {
		t.Errorf("pagination = %+v, want %+v", *got, want)
	}

	// then: the built-in it shadows is gone — its own shape no longer matches
	stripe := ir.Operation{Params: query("starting_after", "limit"), Responses: jsonResponse("data[]", "has_more")}
	if p := match(t, stripe, "", cfg); p != nil {
		t.Errorf("shadowed built-in still matched: %+v", *p)
	}
}

func TestPaginationDeclaredSchemeOutranksBuiltin(t *testing.T) {
	// given: a declared scheme under its own name whose shape a built-in also
	// fits — the two would otherwise veto each other as ambiguous
	cfg := &config.Config{Pagination: []config.PaginationScheme{{
		Name:     "widget-walk",
		Type:     "cursor",
		Request:  config.PaginationRequest{CursorParam: "after", LimitParam: "limit"},
		Response: config.PaginationResponse{ItemsField: "data", NextField: "has_more", NextKind: "has_more"},
	}}}
	op := ir.Operation{Params: query("after", "limit"), Responses: jsonResponse("data[]", "has_more")}

	// when: matching the operation
	got := match(t, op, "", cfg)

	// then: the declared scheme claims it — declaring one already said which
	// convention this API follows
	if got == nil {
		t.Fatal("no pagination matched; a declared scheme must outrank the built-in it overlaps")
	}
	if got.Scheme != "widget-walk" {
		t.Errorf("scheme = %q, want widget-walk", got.Scheme)
	}
}

func TestValidatePaginationUnknownSchemeName(t *testing.T) {
	// given: a per-operation override naming a scheme that does not exist
	cfg := &config.Config{Operations: map[string]config.Operation{
		"listThings": {Pagination: "cursor"},
	}}

	// when: validating the config
	err := ValidatePagination(cfg)

	// then: it fails, naming the offender and the valid choices
	if err == nil {
		t.Fatal("ValidatePagination = nil, want an error")
	}
	for _, want := range []string{"listThings", `"cursor"`, "cursor-has-more", "off"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q does not mention %q", err, want)
		}
	}
}
