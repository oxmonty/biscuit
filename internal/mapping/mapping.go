// Package mapping builds the IR from a loaded spec. This phase is sequential
// by design: global ordering over sorted inputs is what guarantees
// byte-identical output regardless of scheduling (see PRD "Generation
// pipeline and concurrency model").
package mapping

import (
	"encoding/json"
	"sort"
	"strings"

	"github.com/pb33f/libopenapi/datamodel/high/base"
	v3 "github.com/pb33f/libopenapi/datamodel/high/v3"
	"github.com/pb33f/libopenapi/orderedmap"
	"go.yaml.in/yaml/v4"

	"github.com/oxmonty/biscuit/internal/ir"
	"github.com/oxmonty/biscuit/internal/spec"
)

// Map converts a loaded spec into the sorted, normalized IR. overrides is
// biscuit.yaml's per-operation set, keyed by operationId or "METHOD /path";
// in-spec x-biscuit-* extensions merge beneath it, sidecar winning field-wise.
func Map(doc *spec.Document, overrides map[string]ir.Override) *ir.API {
	m := doc.Model
	api := &ir.API{
		SpecVersion: m.Version,
	}
	if m.Info != nil {
		api.Title = m.Info.Title
		api.Description = m.Info.Description
		api.APIVersion = m.Info.Version
	}

	for _, s := range m.Servers {
		api.Servers = append(api.Servers, ir.Server{URL: s.URL, Description: s.Description})
	}
	sort.Slice(api.Servers, func(i, j int) bool { return api.Servers[i].URL < api.Servers[j].URL })

	for _, t := range m.Tags {
		api.Tags = append(api.Tags, ir.Tag{Name: t.Name, Description: t.Description})
	}
	sort.Slice(api.Tags, func(i, j int) bool { return api.Tags[i].Name < api.Tags[j].Name })

	if m.Paths != nil {
		for path, item := range m.Paths.PathItems.FromOldest() {
			api.Operations = append(api.Operations, mapPathItem(path, item)...)
		}
	}
	sortOperations(api.Operations)

	if m.Webhooks != nil {
		for name, item := range m.Webhooks.FromOldest() {
			api.Webhooks = append(api.Webhooks, mapPathItem(name, item)...)
		}
	}
	sortOperations(api.Webhooks)

	if m.Components != nil {
		if m.Components.Schemas != nil {
			for name, proxy := range m.Components.Schemas.FromOldest() {
				api.Schemas = append(api.Schemas, ir.NamedSchema{Name: name, Schema: mapSchemaProxy(proxy)})
			}
			sort.Slice(api.Schemas, func(i, j int) bool { return api.Schemas[i].Name < api.Schemas[j].Name })
		}
		if m.Components.SecuritySchemes != nil {
			for name, s := range m.Components.SecuritySchemes.FromOldest() {
				api.Security = append(api.Security, ir.SecurityScheme{
					Name:   name,
					Type:   s.Type,
					Scheme: s.Scheme,
					In:     s.In,
					Param:  s.Name,
				})
			}
			sort.Slice(api.Security, func(i, j int) bool { return api.Security[i].Name < api.Security[j].Name })
		}
	}

	deriveCommands(api, overrides)
	return api
}

func sortOperations(ops []ir.Operation) {
	sort.Slice(ops, func(i, j int) bool {
		if ops[i].Path != ops[j].Path {
			return ops[i].Path < ops[j].Path
		}
		return ops[i].Method < ops[j].Method
	})
}

func mapPathItem(path string, item *v3.PathItem) []ir.Operation {
	var ops []ir.Operation
	for method, op := range item.GetOperations().FromOldest() {
		mapped := ir.Operation{
			ID:          op.OperationId,
			Method:      strings.ToUpper(method),
			Path:        path,
			Summary:     op.Summary,
			Description: op.Description,
			Deprecated:  op.Deprecated != nil && *op.Deprecated,
		}
		mapped.Tags = append(mapped.Tags, op.Tags...)
		sort.Strings(mapped.Tags)

		if op.Extensions != nil {
			for key, node := range op.Extensions.FromOldest() {
				switch key {
				case "x-biscuit-name":
					_ = node.Decode(&mapped.XBiscuit.Name)
				case "x-biscuit-group":
					_ = node.Decode(&mapped.XBiscuit.Group)
				case "x-biscuit-ignore":
					_ = node.Decode(&mapped.XBiscuit.Ignore)
				case "x-biscuit-pagination":
					_ = node.Decode(&mapped.XBiscuit.Pagination)
				}
			}
		}

		// path-item-level parameters apply to every operation beneath it
		for _, p := range append(append([]*v3.Parameter{}, item.Parameters...), op.Parameters...) {
			mapped.Params = append(mapped.Params, ir.Param{
				Name:        p.Name,
				In:          p.In,
				Description: p.Description,
				Required:    p.Required != nil && *p.Required,
				Schema:      mapSchemaProxy(p.Schema),
			})
		}
		sort.Slice(mapped.Params, func(i, j int) bool {
			if mapped.Params[i].In != mapped.Params[j].In {
				return mapped.Params[i].In < mapped.Params[j].In
			}
			return mapped.Params[i].Name < mapped.Params[j].Name
		})

		if op.RequestBody != nil {
			mapped.RequestBody = mapContent(op.RequestBody.Content)
		}
		if op.Responses != nil {
			for status, resp := range op.Responses.Codes.FromOldest() {
				mapped.Responses = append(mapped.Responses, ir.Response{
					Status:      status,
					Description: resp.Description,
					Content:     mapContent(resp.Content),
				})
			}
			if op.Responses.Default != nil {
				mapped.Responses = append(mapped.Responses, ir.Response{
					Status:      "default",
					Description: op.Responses.Default.Description,
					Content:     mapContent(op.Responses.Default.Content),
				})
			}
			sort.Slice(mapped.Responses, func(i, j int) bool {
				return mapped.Responses[i].Status < mapped.Responses[j].Status
			})
		}
		ops = append(ops, mapped)
	}
	return ops
}

func mapContent(content *orderedmap.Map[string, *v3.MediaType]) []ir.MediaType {
	if content == nil {
		return nil
	}
	var media []ir.MediaType
	for mtype, mt := range content.FromOldest() {
		media = append(media, ir.MediaType{Type: mtype, Schema: mapSchemaProxy(mt.Schema)})
	}
	sort.Slice(media, func(i, j int) bool { return media[i].Type < media[j].Type })
	return media
}

// mapSchemaProxy turns a component $ref into a Ref-only node instead of
// following it — that indirection is what keeps cyclic specs finite.
func mapSchemaProxy(proxy *base.SchemaProxy) *ir.Schema {
	if proxy == nil {
		return nil
	}
	if ref := proxy.GetReference(); ref != "" {
		return &ir.Schema{Ref: refName(ref)}
	}
	s := proxy.Schema()
	if s == nil {
		return nil
	}
	return mapSchema(s)
}

func mapSchema(s *base.Schema) *ir.Schema {
	out := &ir.Schema{
		Format:      s.Format,
		Description: s.Description,
	}

	// 3.1 expresses nullability as a "null" entry in the type array; 3.0 as
	// `nullable: true`. Fold both into one flag over a single base type.
	for _, t := range s.Type {
		if t == "null" {
			out.Nullable = true
		} else if out.Type == "" {
			out.Type = t
		}
	}
	if s.Nullable != nil && *s.Nullable {
		out.Nullable = true
	}

	if s.Properties != nil {
		for name, p := range s.Properties.FromOldest() {
			out.Properties = append(out.Properties, ir.Property{Name: name, Schema: mapSchemaProxy(p)})
		}
		sort.Slice(out.Properties, func(i, j int) bool { return out.Properties[i].Name < out.Properties[j].Name })
	}
	out.Required = append(out.Required, s.Required...)
	sort.Strings(out.Required)

	if s.Items != nil && s.Items.IsA() {
		out.Items = mapSchemaProxy(s.Items.A)
	}

	for _, n := range s.Enum {
		out.Enum = append(out.Enum, nodeJSON(n))
	}

	for _, sub := range s.OneOf {
		out.OneOf = append(out.OneOf, mapSchemaProxy(sub))
	}
	for _, sub := range s.AnyOf {
		out.AnyOf = append(out.AnyOf, mapSchemaProxy(sub))
	}
	for _, sub := range s.AllOf {
		out.AllOf = append(out.AllOf, mapSchemaProxy(sub))
	}

	if s.Discriminator != nil {
		d := &ir.Discriminator{PropertyName: s.Discriminator.PropertyName}
		if s.Discriminator.Mapping != nil {
			for value, target := range s.Discriminator.Mapping.FromOldest() {
				d.Mapping = append(d.Mapping, ir.MappingEntry{Value: value, Schema: refName(target)})
			}
			sort.Slice(d.Mapping, func(i, j int) bool { return d.Mapping[i].Value < d.Mapping[j].Value })
		}
		out.Discriminator = d
	}

	// 3.0's single `example` folds into the 3.1 `examples` list.
	for _, n := range s.Examples {
		out.Examples = append(out.Examples, nodeJSON(n))
	}
	if s.Example != nil {
		out.Examples = append(out.Examples, nodeJSON(s.Example))
	}

	if s.Default != nil {
		out.Default = nodeJSON(s.Default)
	}

	return out
}

// nodeJSON renders a YAML node as canonical JSON so example/enum/default
// values stay structured but byte-deterministic.
func nodeJSON(n *yaml.Node) string {
	var v any
	if err := n.Decode(&v); err != nil {
		return ""
	}
	b, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	return string(b)
}

func refName(ref string) string {
	return ref[strings.LastIndex(ref, "/")+1:]
}
