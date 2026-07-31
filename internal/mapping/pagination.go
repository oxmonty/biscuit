package mapping

import (
	"fmt"
	"sort"
	"strings"

	"github.com/oxmonty/biscuit/internal/config"
	"github.com/oxmonty/biscuit/internal/ir"
)

// Pagination detection runs here, at generation time, and is deliberately
// conservative: an operation walks only when a request query param exactly
// names one of a scheme's page params AND its success response corroborates
// with an items array plus a next-page signal. Never a substring match, never
// `limit` alone, and an operation two schemes both claim paginates under
// neither — a wrong guess turns one command into an unbounded crawl of
// someone's API. The vocabulary is sourced in
// docs/research/pagination-conventions.md.

// Scheme is one pagination convention. Built-ins carry alias lists — the same
// convention is spelled several ways in the wild — and a biscuit.yaml scheme
// produces the same struct with single-element lists. Within a list, order is
// preference: the first name the operation declares is the one the walk uses.
type Scheme struct {
	Name         string
	Declared     bool     // came from biscuit.yaml, so it outranks the built-in library
	Type         string   // cursor | offset | page
	Params       []string // query params that advance a page
	LimitParams  []string // query params carrying the page size
	RequireLimit bool     // only match when a LimitParams member is present too
	ItemsField   string   // response items path; empty infers the response's single array field
	NextFields   []string // response next-signal paths
	NextKind     string   // has_more | cursor | url | link_header
	// Corroborate lists offset/page fields accepted as the next-page signal
	// when no NextFields member is present: a documented total says the
	// response is a page of something, and the walk then steps arithmetically.
	Corroborate []string
}

// builtinSchemes is the convention library, sorted by name — section 3 of the
// survey, which ranks these by prevalence.
var builtinSchemes = []Scheme{
	{
		Name: "cursor-has-more", Type: "cursor",
		Params:      []string{"starting_after", "after", "cursor", "ending_before"},
		LimitParams: []string{"limit"},
		NextFields:  []string{"has_more"},
		NextKind:    "has_more",
	},
	{
		Name: "cursor-next-cursor", Type: "cursor",
		Params:      []string{"cursor", "page_info"},
		LimitParams: []string{"limit"},
		NextFields:  []string{"next_cursor", "response_metadata.next_cursor"},
		NextKind:    "cursor",
	},
	{
		Name: "link-header", Type: "page",
		Params:      []string{"page"},
		LimitParams: []string{"per_page", "page_size", "perPage", "pageSize"},
		NextKind:    "link_header",
	},
	{
		Name: "next-token", Type: "cursor",
		Params:      []string{"NextToken", "NextPageToken", "NextMarker"},
		LimitParams: []string{"MaxResults", "MaxItems", "Limit"},
		NextFields:  []string{"NextToken", "NextPageToken", "NextMarker"},
		NextKind:    "cursor",
	},
	{
		Name: "offset-limit", Type: "offset",
		Params:       []string{"offset"},
		LimitParams:  []string{"limit", "per_page", "page_size", "pageSize"},
		RequireLimit: true,
		NextFields:   []string{"next", "links.next", "meta.next", "next_page_uri", "pagination.next"},
		NextKind:     "url",
		Corroborate:  []string{"total", "total_count", "totalCount", "total_items", "count", "meta.total", "pagination.total"},
	},
	{
		Name: "page-number", Type: "page",
		Params:       []string{"page"},
		LimitParams:  []string{"per_page", "page_size", "per-page", "pageSize", "limit"},
		RequireLimit: true,
		NextFields: []string{"links.next", "next", "meta.next", "next_page_uri"},
		NextKind:   "url",
		// page-count fields only: a documented total_pages says the page
		// param really is a number, whereas a bare total says nothing —
		// Stripe's search endpoints pair `page` with `total_count` and yet
		// take an opaque cursor there, which numeric stepping would 400 on.
		Corroborate: []string{"total_pages", "totalPages", "page_count", "last_page", "meta.total_pages", "pagination.total_pages"},
	},
	{
		Name: "page-token", Type: "cursor",
		Params:      []string{"pageToken", "page_token"},
		LimitParams: []string{"pageSize", "page_size", "maxResults", "max_results"},
		NextFields:  []string{"nextPageToken", "next_page_token"},
		NextKind:    "cursor",
	},
}

// SchemeNames lists every scheme a per-operation override may name, sorted.
func SchemeNames(cfg *config.Config) []string {
	var names []string
	for _, s := range schemesFor(cfg) {
		names = append(names, s.Name)
	}
	sort.Strings(names)
	return names
}

// ValidatePagination rejects a per-operation pagination override naming a
// scheme that does not exist. Shape validation of the schemes themselves
// happens in config.Load; this needs the built-in library to answer.
func ValidatePagination(cfg *config.Config) error {
	if cfg == nil {
		return nil
	}
	keys := make([]string, 0, len(cfg.Operations))
	for key := range cfg.Operations {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	names := SchemeNames(cfg)
	for _, key := range keys {
		if err := checkSchemeName(cfg.Operations[key].Pagination, names); err != nil {
			return fmt.Errorf("operations.%s: %w", key, err)
		}
	}
	return nil
}

func checkSchemeName(name string, valid []string) error {
	if name == "" || name == paginationOff {
		return nil
	}
	for _, v := range valid {
		if v == name {
			return nil
		}
	}
	return fmt.Errorf("unknown pagination scheme %q (want off, %s)", name, strings.Join(valid, ", "))
}

const paginationOff = "off"

// schemesFor is the scheme list the matcher runs, declared first: a
// biscuit.yaml scheme shadows the built-in sharing its name.
func schemesFor(cfg *config.Config) []Scheme {
	if cfg == nil || len(cfg.Pagination) == 0 {
		return builtinSchemes
	}
	declared := make([]Scheme, 0, len(cfg.Pagination))
	shadowed := make(map[string]bool, len(cfg.Pagination))
	for _, s := range cfg.Pagination {
		shadowed[s.Name] = true
		declared = append(declared, schemeFromConfig(s))
	}
	sort.Slice(declared, func(i, j int) bool { return declared[i].Name < declared[j].Name })
	out := declared
	for _, b := range builtinSchemes {
		if !shadowed[b.Name] {
			out = append(out, b)
		}
	}
	return out
}

func schemeFromConfig(s config.PaginationScheme) Scheme {
	out := Scheme{
		Name:       s.Name,
		Declared:   true,
		Type:       s.Type,
		ItemsField: s.Response.ItemsField,
		NextKind:   s.Response.NextKind,
	}
	for _, p := range []string{s.Request.CursorParam, s.Request.OffsetParam, s.Request.PageParam} {
		if p != "" {
			out.Params = append(out.Params, p)
		}
	}
	if s.Request.LimitParam != "" {
		out.LimitParams = []string{s.Request.LimitParam}
	}
	if s.Response.NextField != "" {
		out.NextFields = []string{s.Response.NextField}
	}
	return out
}

// resolvePagination decides whether one operation walks. override is the
// effective per-operation hint: "off" disables the walk outright, a scheme
// name pins that scheme (skipping the ambiguity veto, which is what the hint
// exists to break), and empty auto-detects. The second result is a diagnostic
// for a hint that names something unusable.
func resolvePagination(op *ir.Operation, override string, schemes []Scheme, fl *flattener) (*ir.Pagination, string) {
	if override == paginationOff {
		return nil, ""
	}
	if override != "" {
		for i := range schemes {
			if schemes[i].Name != override {
				continue
			}
			if p := schemes[i].match(op, fl); p != nil {
				return p, ""
			}
			return nil, fmt.Sprintf("%s %s: pagination scheme %q does not fit this operation's shape — no pagination",
				op.Method, op.Path, override)
		}
		return nil, fmt.Sprintf("%s %s: unknown pagination scheme %q — no pagination", op.Method, op.Path, override)
	}

	// Declared schemes are a tier of their own: a built-in that happens to fit
	// the same operation is not a conflict worth vetoing, because declaring a
	// scheme already said which convention this API follows.
	matched := matchTier(op, schemes, fl, true)
	if len(matched) == 0 {
		matched = matchTier(op, schemes, fl, false)
	}
	switch len(matched) {
	case 1:
		return matched[0], ""
	case 0:
		return nil, ""
	default:
		names := make([]string, len(matched))
		for i, m := range matched {
			names[i] = m.Scheme
		}
		return nil, fmt.Sprintf("%s %s: pagination schemes %s all match — no pagination; set operations.<id>.pagination to pick one",
			op.Method, op.Path, strings.Join(names, ", "))
	}
}

func matchTier(op *ir.Operation, schemes []Scheme, fl *flattener, declared bool) []*ir.Pagination {
	var matched []*ir.Pagination
	for i := range schemes {
		if schemes[i].Declared != declared {
			continue
		}
		if p := schemes[i].match(op, fl); p != nil {
			matched = append(matched, p)
		}
	}
	return matched
}

// match tests one scheme against one operation, returning the resolved walk
// or nil. Both sides must corroborate: a page param on the request, an items
// array plus a next-page signal on the success response.
func (s *Scheme) match(op *ir.Operation, fl *flattener) *ir.Pagination {
	query := make(map[string]bool, len(op.Params))
	for _, p := range op.Params {
		if p.In == "query" {
			query[p.Name] = true
		}
	}
	param := firstPresent(s.Params, query)
	if param == "" {
		return nil
	}
	limit := firstPresent(s.LimitParams, query)
	if s.RequireLimit && limit == "" {
		return nil
	}

	resp := successResponse(op)
	if resp == nil {
		return nil
	}
	body := fl.resolve(jsonResponseSchema(resp), nil)
	items, ok := itemsPath(body, s.ItemsField, fl)
	if !ok {
		return nil
	}
	next, kind := s.nextSignal(body, resp, fl)
	if kind == "" {
		return nil
	}
	return &ir.Pagination{
		Scheme:     s.Name,
		Type:       s.Type,
		Param:      param,
		LimitParam: limit,
		ItemsPath:  items,
		NextPath:   next,
		NextKind:   kind,
	}
}

// nextSignal picks the response-side evidence that more pages exist, and with
// it the walk kind. An empty kind means the response corroborates nothing.
func (s *Scheme) nextSignal(body *ir.Schema, resp *ir.Response, fl *flattener) (path, kind string) {
	if s.NextKind == "link_header" {
		for _, h := range resp.Headers {
			if strings.EqualFold(h, "Link") {
				return "", "link_header"
			}
		}
		return "", ""
	}
	for _, f := range s.NextFields {
		if lookupPath(body, f, fl) != nil {
			return f, s.NextKind
		}
	}
	for _, f := range s.Corroborate {
		if lookupPath(body, f, fl) != nil {
			return "", "step"
		}
	}
	return "", ""
}

func firstPresent(candidates []string, have map[string]bool) string {
	for _, c := range candidates {
		if have[c] {
			return c
		}
	}
	return ""
}

// successResponse returns the operation's first 2xx response — responses are
// status-sorted in the IR, so "200" wins over "201" and "2XX" deterministically.
func successResponse(op *ir.Operation) *ir.Response {
	for i := range op.Responses {
		if strings.HasPrefix(op.Responses[i].Status, "2") {
			return &op.Responses[i]
		}
	}
	return nil
}

// isSSE reports whether op's success response streams Server-Sent Events —
// a 200-family response declaring a text/event-stream body, per the Protocol
// scope section: SSE endpoints are ordinary operations with that content type.
func isSSE(op *ir.Operation) bool {
	resp := successResponse(op)
	if resp == nil {
		return false
	}
	for _, mt := range resp.Content {
		if mt.Type == "text/event-stream" {
			return true
		}
	}
	return false
}

func jsonResponseSchema(resp *ir.Response) *ir.Schema {
	for _, mt := range resp.Content {
		if strings.Contains(mt.Type, "json") {
			return mt.Schema
		}
	}
	return nil
}

// itemsPath locates the response's items array: the scheme's declared field
// when it names one, a bare top-level array as itself, else the response's
// single array-typed property — one array is unambiguous, two make the guess
// a coin flip.
func itemsPath(body *ir.Schema, field string, fl *flattener) (string, bool) {
	if body == nil {
		return "", false
	}
	if body.Type == "array" {
		return "@this", true
	}
	if field != "" {
		return field, isArraySchema(lookupPath(body, field, fl))
	}
	found := ""
	for _, p := range body.Properties { // sorted at mapping time
		if isArraySchema(fl.resolve(p.Schema, nil)) {
			if found != "" {
				return "", false
			}
			found = p.Name
		}
	}
	return found, found != ""
}

func isArraySchema(s *ir.Schema) bool { return s != nil && s.Type == "array" }

// lookupPath resolves a dot-separated field path against a response schema,
// returning nil when any segment is absent.
func lookupPath(body *ir.Schema, path string, fl *flattener) *ir.Schema {
	cur := body
	for _, seg := range strings.Split(path, ".") {
		if cur == nil {
			return nil
		}
		next := (*ir.Schema)(nil)
		for _, p := range cur.Properties {
			if p.Name == seg {
				next = fl.resolve(p.Schema, nil)
				break
			}
		}
		if next == nil {
			return nil
		}
		cur = next
	}
	return cur
}
