package mapping

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/oxmonty/biscuit/internal/ir"
)

// deriveCommands builds the resource/verb tree on the API. Paths are primary —
// static segments carry the nesting — and tags only group operations whose
// path has no static segments. Verbs come from operationIds with resource/tag
// stutter stripped (Speakeasy's disclosed heuristic), falling back to
// method+shape names when the id is missing or reduces to a bare HTTP verb.
func deriveCommands(api *ir.API) {
	extended := make(map[string]bool) // normalized path prefixes that longer paths extend
	postOnly := make(map[string]bool) // normalized full paths whose every operation is POST
	for _, op := range api.Operations {
		segs := pathSegments(op.Path)
		for i := 1; i < len(segs); i++ {
			extended[normJoin(segs[:i])] = true
		}
		key := normJoin(segs)
		if _, seen := postOnly[key]; !seen {
			postOnly[key] = true
		}
		if op.Method != "POST" {
			postOnly[key] = false
		}
	}

	root := newNode("")
	for i := range api.Operations {
		op := &api.Operations[i]
		chain, verb := commandFor(op, extended, postOnly)
		n := root
		for _, name := range chain {
			n = n.child(name)
		}
		name := verb
		if prev, taken := n.verbs[name]; taken {
			name = verb + "-" + strings.ToLower(op.Method)
			for i := 2; ; i++ {
				if _, still := n.verbs[name]; !still {
					break
				}
				name = fmt.Sprintf("%s-%s-%d", verb, strings.ToLower(op.Method), i)
			}
			api.Diagnostics = append(api.Diagnostics, fmt.Sprintf(
				"command %q: %s %s and %s %s both map to verb %q; renamed the latter to %q — set an override in biscuit.yaml to choose better names",
				strings.Join(chain, " "), prev.Method, prev.Path, op.Method, op.Path, verb, name))
		}
		n.verbs[name] = &ir.Verb{
			Name:        name,
			Method:      op.Method,
			Path:        op.Path,
			OperationID: op.ID,
			Summary:     op.Summary,
			Deprecated:  op.Deprecated,
		}
	}

	tagDesc := make(map[string]string, len(api.Tags))
	for _, t := range api.Tags {
		tagDesc[kebab(t.Name)] = t.Description
	}
	api.Commands = root.freeze(tagDesc)
	api.RootVerbs = frozenVerbs(root.verbs)
}

// commandFor derives (resource chain, verb) for one operation.
func commandFor(op *ir.Operation, extended, postOnly map[string]bool) ([]string, string) {
	segs := pathSegments(op.Path)
	// leading /api and /vN segments are mount points, not resources:
	// /v1/users and /api/v2/ability both shed their prefixes
	for len(segs) > 0 && (segs[0] == "api" || versionSeg.MatchString(segs[0])) {
		segs = segs[1:]
	}

	// A singular trailing static segment right after a path param, on a
	// POST-only path nothing nests beneath, is a custom action (Stripe's
	// /{id}/confirm shape): it becomes the verb, not a resource. Plural
	// segments stay resources — /{id}/login_links is a create-only
	// sub-collection, not an action. Heuristic misses are what biscuit.yaml
	// overrides exist for.
	if n := len(segs); n >= 2 && !isParam(segs[n-1]) && isParam(segs[n-2]) &&
		!isPlural(segs[n-1]) &&
		op.Method == "POST" && postOnly[normJoin(pathSegments(op.Path))] &&
		!extended[normJoin(pathSegments(op.Path))] {
		var chain []string
		for _, s := range segs[:n-1] {
			if !isParam(s) {
				chain = append(chain, kebab(s))
			}
		}
		return chain, kebab(segs[n-1])
	}

	var chain []string
	for _, s := range segs {
		if !isParam(s) {
			chain = append(chain, kebab(s))
		}
	}
	if len(chain) == 0 && len(op.Tags) > 0 {
		chain = []string{kebab(op.Tags[0])}
	}
	return chain, verbFor(op, chain)
}

func verbFor(op *ir.Operation, chain []string) string {
	shape := shapeVerb(op)
	if op.ID == "" {
		return shape
	}
	stop := stutterSet(chain, op)
	var kept []string
	for _, t := range kebabTokens(op.ID) {
		if !stop[t] && !versionSeg.MatchString(t) {
			kept = append(kept, t)
		}
	}
	// a "by" whose object was stripped as stutter dangles: showPetById → show
	if n := len(kept); n > 0 && kept[n-1] == "by" {
		kept = kept[:n-1]
	}
	if len(kept) == 0 {
		return shape
	}
	v := strings.Join(kept, "-")
	// An id that reduces to a bare HTTP verb ("GetCustomersSubscriptions" →
	// "get") says less than the path shape does; prefer list/get/create/....
	if httpVerbs[v] {
		return shape
	}
	return v
}

// shapeVerb names an operation from method + path shape alone.
func shapeVerb(op *ir.Operation) string {
	switch op.Method {
	case "GET":
		segs := pathSegments(op.Path)
		if len(segs) > 0 && isParam(segs[len(segs)-1]) {
			return "get"
		}
		return "list"
	case "POST":
		// POST on an instance path is an update in the wild — Stripe's entire
		// API updates via POST /v1/{resource}/{id}, never PUT/PATCH
		segs := pathSegments(op.Path)
		if len(segs) > 0 && isParam(segs[len(segs)-1]) {
			return "update"
		}
		return "create"
	case "PUT", "PATCH":
		return "update"
	case "DELETE":
		return "delete"
	default:
		return strings.ToLower(op.Method)
	}
}

// stutterSet is the token set stripped from operationIds: every resource-chain,
// tag, and path-param token plus its naive singular.
func stutterSet(chain []string, op *ir.Operation) map[string]bool {
	stop := make(map[string]bool)
	add := func(name string) {
		for _, t := range kebabTokens(name) {
			stop[t] = true
			stop[singular(t)] = true
		}
	}
	for _, c := range chain {
		add(c)
	}
	for _, t := range op.Tags {
		add(t)
	}
	for _, s := range pathSegments(op.Path) {
		if isParam(s) {
			add(strings.Trim(s, "{}"))
		}
	}
	return stop
}

func isPlural(seg string) bool {
	tokens := kebabTokens(seg)
	if len(tokens) == 0 {
		return false
	}
	last := tokens[len(tokens)-1]
	return singular(last) != last
}

// ponytail: naive singularizer (ies→y, trailing s), enough for stutter
// stripping; a real inflector only if the bench shows misses that matter.
func singular(s string) string {
	switch {
	case strings.HasSuffix(s, "ies"):
		return s[:len(s)-3] + "y"
	case strings.HasSuffix(s, "s") && !strings.HasSuffix(s, "ss"):
		return s[:len(s)-1]
	}
	return s
}

var (
	versionSeg = regexp.MustCompile(`^v[0-9]+$`)
	httpVerbs  = map[string]bool{"get": true, "post": true, "put": true, "patch": true, "delete": true, "head": true, "options": true, "trace": true}
)

func pathSegments(p string) []string {
	var segs []string
	for _, s := range strings.Split(p, "/") {
		if s != "" {
			segs = append(segs, s)
		}
	}
	return segs
}

func isParam(seg string) bool {
	return strings.HasPrefix(seg, "{") && strings.HasSuffix(seg, "}")
}

// normJoin joins segments with param names erased, so prefix relationships
// hold across paths that name the same param differently.
func normJoin(segs []string) string {
	norm := make([]string, len(segs))
	for i, s := range segs {
		if isParam(s) {
			norm[i] = "{}"
		} else {
			norm[i] = s
		}
	}
	return strings.Join(norm, "/")
}

// kebab lowercases and hyphenates snake_case, camelCase, and acronym runs
// (HTTPProxy → http-proxy).
func kebab(s string) string {
	var out []rune
	runes := []rune(s)
	for i, r := range runes {
		switch {
		case r == '_' || r == ' ' || r == '.' || r == '-':
			if len(out) > 0 && out[len(out)-1] != '-' {
				out = append(out, '-')
			}
			continue
		case isUpper(r):
			prevLower := i > 0 && isLower(runes[i-1])
			nextLower := i+1 < len(runes) && isLower(runes[i+1])
			if len(out) > 0 && out[len(out)-1] != '-' && (prevLower || nextLower) {
				out = append(out, '-')
			}
			out = append(out, r+('a'-'A'))
		default:
			out = append(out, r)
		}
	}
	return strings.Trim(string(out), "-")
}

func isUpper(r rune) bool { return r >= 'A' && r <= 'Z' }
func isLower(r rune) bool { return r >= 'a' && r <= 'z' }

func kebabTokens(s string) []string {
	k := kebab(s)
	if k == "" {
		return nil
	}
	return strings.Split(k, "-")
}

type node struct {
	name     string
	verbs    map[string]*ir.Verb
	children map[string]*node
}

func newNode(name string) *node {
	return &node{name: name, verbs: make(map[string]*ir.Verb), children: make(map[string]*node)}
}

func (n *node) child(name string) *node {
	c, ok := n.children[name]
	if !ok {
		c = newNode(name)
		n.children[name] = c
	}
	return c
}

func (n *node) freeze(tagDesc map[string]string) []ir.Command {
	names := make([]string, 0, len(n.children))
	for name := range n.children {
		names = append(names, name)
	}
	sort.Strings(names)
	var out []ir.Command
	for _, name := range names {
		c := n.children[name]
		out = append(out, ir.Command{
			Name:        name,
			Description: tagDesc[name],
			Verbs:       frozenVerbs(c.verbs),
			Children:    c.freeze(tagDesc),
		})
	}
	return out
}

func frozenVerbs(verbs map[string]*ir.Verb) []ir.Verb {
	names := make([]string, 0, len(verbs))
	for v := range verbs {
		names = append(names, v)
	}
	sort.Strings(names)
	var out []ir.Verb
	for _, v := range names {
		out = append(out, *verbs[v])
	}
	return out
}
