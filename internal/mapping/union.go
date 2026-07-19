package mapping

import (
	"sort"

	"github.com/oxmonty/biscuit/internal/ir"
)

// discriminate runs the discriminator-inference cascade on a oneOf schema:
// explicit discriminator → unique field → JSON type → enum value, opaque
// when nothing identifies the variants. The verdict rides on the flag so
// dry-run, help/completions, and the execution layer all share one answer.
// ponytail: the flag itself stays a single json value; expanding the union
// of variant properties into flags is the upgrade path if the bench asks.
func (fl *flattener) discriminate(s *ir.Schema) *ir.Union {
	// explicit discriminator first — it needs no variant inspection, so an
	// unresolvable variant can't knock it out
	if s.Discriminator != nil && s.Discriminator.PropertyName != "" {
		u := &ir.Union{Kind: "discriminator", Property: s.Discriminator.PropertyName}
		if len(s.Discriminator.Mapping) > 0 {
			for _, m := range s.Discriminator.Mapping {
				u.Variants = append(u.Variants, ir.UnionVariant{Value: m.Value, Schema: m.Schema})
			}
		} else {
			// no mapping: the wire value is the schema name, per the OpenAPI
			// default convention
			for _, v := range s.OneOf {
				u.Variants = append(u.Variants, ir.UnionVariant{Value: v.Ref, Schema: v.Ref})
			}
		}
		sortVariants(u)
		return u
	}

	type variant struct {
		name string // component name, "" for inline
		r    *ir.Schema
	}
	var vars []variant
	for _, v := range s.OneOf {
		r := fl.resolve(v, nil)
		if r == nil {
			return &ir.Union{Kind: "opaque"} // cyclic or unresolvable variant
		}
		vars = append(vars, variant{name: v.Ref, r: r})
	}
	if len(vars) < 2 {
		return nil
	}

	// unique field: every variant has a property no other variant has
	presence := map[string]int{}
	for _, v := range vars {
		for _, p := range v.r.Properties {
			presence[p.Name]++
		}
	}
	unique := make([]string, len(vars))
	allUnique := true
	for i, v := range vars {
		// prefer a required unique property; properties are sorted, so the
		// first hit is deterministic
		for _, p := range v.r.Properties {
			if presence[p.Name] == 1 && (unique[i] == "" || contains(v.r.Required, p.Name) && !contains(v.r.Required, unique[i])) {
				unique[i] = p.Name
			}
		}
		if unique[i] == "" {
			allUnique = false
		}
	}
	if allUnique {
		u := &ir.Union{Kind: "unique-field"}
		for i, v := range vars {
			u.Variants = append(u.Variants, ir.UnionVariant{Value: unique[i], Schema: v.name})
		}
		sortVariants(u)
		return u
	}

	// JSON type: variants have pairwise-distinct primitive shapes
	types := map[string]bool{}
	distinct := true
	for _, v := range vars {
		typ := v.r.Type
		if typ == "" || types[typ] {
			distinct = false
			break
		}
		types[typ] = true
	}
	if distinct {
		u := &ir.Union{Kind: "json-type"}
		for _, v := range vars {
			u.Variants = append(u.Variants, ir.UnionVariant{Value: v.r.Type, Schema: v.name})
		}
		sortVariants(u)
		return u
	}

	// enum value: a property shared by every variant, const-valued (single
	// enum entry) with pairwise-distinct values
	shared := map[string]int{}
	for _, v := range vars {
		for _, p := range v.r.Properties {
			shared[p.Name]++
		}
	}
	var candidates []string
	for name, n := range shared {
		if n == len(vars) {
			candidates = append(candidates, name)
		}
	}
	sort.Strings(candidates)
	for _, prop := range candidates {
		values := make([]string, len(vars))
		seen := map[string]bool{}
		ok := true
		for i, v := range vars {
			var e []string
			for _, p := range v.r.Properties {
				if p.Name == prop {
					e = enumOf(fl, p.Schema)
				}
			}
			if len(e) != 1 || seen[e[0]] {
				ok = false
				break
			}
			seen[e[0]] = true
			values[i] = e[0]
		}
		if ok {
			u := &ir.Union{Kind: "enum-value", Property: prop}
			for i, v := range vars {
				u.Variants = append(u.Variants, ir.UnionVariant{Value: values[i], Schema: v.name})
			}
			sortVariants(u)
			return u
		}
	}

	return &ir.Union{Kind: "opaque"}
}

func enumOf(fl *flattener, s *ir.Schema) []string {
	r := fl.resolve(s, nil)
	if r == nil {
		return nil
	}
	return r.Enum
}

func contains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

func sortVariants(u *ir.Union) {
	sort.Slice(u.Variants, func(i, j int) bool { return u.Variants[i].Value < u.Variants[j].Value })
}
