package mapping

import (
	"fmt"
	"sort"
	"strings"

	"github.com/oxmonty/biscuit/internal/ir"
)

// Flag flattening: parameters map one-to-one; the JSON request body flattens
// into dot-notation flags with a schema-adaptive depth. Per operation, the
// deepest expansion whose total flag count fits the budget wins (iterative
// deepening), so small schemas expand fully and exploding ones cap early —
// the "deeper than Stainless's fixed two levels where the schema is small"
// deviation. Subtrees cut by depth, budget, or a $ref cycle become single
// json-typed flags (the inline-JSON/YAML layer). Arrays never dot-expand:
// scalar items are repeated scalar flags, object items repeated json flags.
//
// 3.1 constructs, explicitly: type arrays incl. null are already normalized
// in the IR; oneOf/anyOf collapse to json here (the discriminator cascade
// refines oneOf); allOf merges member properties; if/then/else and
// prefixItems are not represented in the IR yet, so conditional/tuple shapes
// fall through to their base type or json — deferred, not crashed.
const (
	// ponytail: fixed budget/bound, config knobs only if the bench shows
	// real specs need tuning
	maxFlagsPerOp  = 64
	hardDepthBound = 8
)

func flagsFor(op *ir.Operation, schemas map[string]*ir.Schema) []ir.Flag {
	fl := &flattener{schemas: schemas}
	var flags []ir.Flag
	taken := map[string]bool{}
	add := func(f ir.Flag) {
		// a body flag colliding with a parameter flag keeps a body. prefix;
		// numeric suffixes cover the pathological rest, deterministically
		if taken[f.Name] {
			f.Name = "body." + f.Name
		}
		for i := 2; taken[f.Name]; i++ {
			f.Name = fmt.Sprintf("%s-%d", f.Name, i)
		}
		taken[f.Name] = true
		flags = append(flags, f)
	}

	for i := range op.Params {
		p := &op.Params[i]
		if p.In == "cookie" {
			continue // deferred: cookie params have no flag mapping yet
		}
		f := ir.Flag{
			Name:        kebab(p.Name),
			In:          p.In,
			Description: p.Description,
			Required:    p.Required || p.In == "path",
		}
		fl.fill(&f, fl.resolve(p.Schema, nil))
		add(f)
	}

	if body := jsonBodySchema(op); body != nil {
		if root := fl.resolve(body, nil); root != nil && len(root.Properties) > 0 {
			fl.expand(body, nil, nil, nil, fl.chooseDepth(body), true, add)
		} else {
			f := ir.Flag{Name: "body", In: "body", Required: true}
			fl.fill(&f, root)
			add(f)
		}
	}

	sort.Slice(flags, func(i, j int) bool { return flags[i].Name < flags[j].Name })
	return flags
}

// jsonBodySchema picks the flag-bearing request media type: JSON-ish first,
// multipart/form-urlencoded otherwise (their fields flatten the same way).
func jsonBodySchema(op *ir.Operation) *ir.Schema {
	var fallback *ir.Schema
	for _, mt := range op.RequestBody {
		switch {
		case strings.Contains(mt.Type, "json"):
			if mt.Schema != nil {
				return mt.Schema
			}
		case strings.HasPrefix(mt.Type, "multipart/") || mt.Type == "application/x-www-form-urlencoded":
			if fallback == nil {
				fallback = mt.Schema
			}
		}
	}
	return fallback
}

type flattener struct {
	schemas map[string]*ir.Schema
}

// resolve follows component refs and merges allOf members. seen guards the
// ref path walked so far; a revisit (cycle) returns nil.
func (fl *flattener) resolve(s *ir.Schema, seen []string) *ir.Schema {
	for s != nil && s.Ref != "" {
		for _, name := range seen {
			if name == s.Ref {
				return nil
			}
		}
		seen = append(seen, s.Ref)
		s = fl.schemas[s.Ref]
	}
	if s == nil || len(s.AllOf) == 0 {
		return s
	}
	merged := *s
	merged.AllOf = nil
	for _, member := range s.AllOf {
		m := fl.resolve(member, seen)
		if m == nil {
			continue
		}
		merged.Properties = append(merged.Properties, m.Properties...)
		merged.Required = append(merged.Required, m.Required...)
		if merged.Type == "" {
			merged.Type = m.Type
		}
	}
	sort.Slice(merged.Properties, func(i, j int) bool {
		return merged.Properties[i].Name < merged.Properties[j].Name
	})
	sort.Strings(merged.Required)
	return &merged
}

// chooseDepth returns the deepest limit whose flag count fits the budget.
func (fl *flattener) chooseDepth(body *ir.Schema) int {
	for d := hardDepthBound; d > 1; d-- {
		if fl.count(body, nil, d) <= maxFlagsPerOp {
			return d
		}
	}
	return 1
}

func (fl *flattener) count(s *ir.Schema, seen []string, depth int) int {
	seen, r := fl.walk(s, seen)
	if r == nil || depth == 0 || !expandable(r) {
		return 1
	}
	n := 0
	for _, p := range r.Properties {
		n += fl.count(p.Schema, seen, depth-1)
	}
	return n
}

func (fl *flattener) expand(s *ir.Schema, path, bodyPath, seen []string, depth int, required bool, add func(ir.Flag)) {
	seen, r := fl.walk(s, seen)
	name := strings.Join(path, ".")
	if r == nil {
		// ref cycle: the subtree can't expand statically; inline JSON takes it
		add(ir.Flag{Name: name, In: "body", BodyPath: bodyPath, Type: "json", Required: required})
		return
	}
	if depth > 0 && expandable(r) {
		reqSet := make(map[string]bool, len(r.Required))
		for _, req := range r.Required {
			reqSet[req] = true
		}
		for _, p := range r.Properties {
			childPath := append(path[:len(path):len(path)], kebab(p.Name))
			childBody := append(bodyPath[:len(bodyPath):len(bodyPath)], p.Name)
			fl.expand(p.Schema, childPath, childBody, seen, depth-1, required && reqSet[p.Name], add)
		}
		return
	}
	f := ir.Flag{Name: name, In: "body", BodyPath: bodyPath, Required: required}
	fl.fill(&f, r)
	add(f)
}

func expandable(r *ir.Schema) bool {
	return len(r.Properties) > 0 && len(r.OneOf) == 0 && len(r.AnyOf) == 0
}

// walk resolves one node against the cycle guard, keeping the guard state.
func (fl *flattener) walk(s *ir.Schema, seen []string) ([]string, *ir.Schema) {
	for s != nil && s.Ref != "" {
		for _, name := range seen {
			if name == s.Ref {
				return seen, nil
			}
		}
		seen = append(seen[:len(seen):len(seen)], s.Ref)
		s = fl.schemas[s.Ref]
	}
	if s != nil && len(s.AllOf) > 0 {
		return seen, fl.resolve(s, seen)
	}
	return seen, s
}

// fill sets Type/Repeated/Enum/Default/Description from a resolved schema;
// anything without a scalar shape is json.
func (fl *flattener) fill(f *ir.Flag, s *ir.Schema) {
	if s == nil {
		f.Type = "json"
		return
	}
	if f.Description == "" {
		f.Description = s.Description
	}
	f.Enum = s.Enum
	f.Default = s.Default
	if len(s.OneOf) > 0 {
		f.Type = "json"
		f.Union = fl.discriminate(s)
		return
	}
	if len(s.AnyOf) > 0 {
		f.Type = "json" // anyOf carries no exclusive-variant story to infer
		return
	}
	switch s.Type {
	case "string", "boolean", "integer", "number":
		f.Type = s.Type
	case "array":
		f.Repeated = true
		var inner ir.Flag
		fl.fill(&inner, fl.resolve(s.Items, nil))
		if inner.Type == "json" || inner.Repeated {
			f.Type = "json" // object or nested-array items: one JSON value per occurrence
		} else {
			f.Type = inner.Type
			f.Enum = inner.Enum
		}
	default:
		f.Type = "json"
	}
}
