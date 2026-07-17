// Package ir is biscuit's intermediate representation: the immutable,
// deterministic shape between spec and templates. Every slice is sorted at
// mapping time; 3.0 and 3.1 differences (nullable vs type arrays, example vs
// examples) are already normalized away. Nothing downstream may reach back
// into the spec — each output file's bytes depend only on this.
package ir

// API is the root of the IR.
type API struct {
	Title       string
	Description string
	SpecVersion string // OpenAPI version the spec declares, e.g. "3.1.0"
	APIVersion  string // info.version
	Servers     []Server
	Tags        []Tag
	Operations  []Operation // sorted by (Path, Method)
	Webhooks    []Operation // sorted by (Path, Method); Path holds the webhook name
	Schemas     []NamedSchema
	Security    []SecurityScheme
}

type Server struct {
	URL         string
	Description string
}

type Tag struct {
	Name        string
	Description string
}

type Operation struct {
	ID          string // operationId; empty means E3 derives a path-based name
	Method      string
	Path        string
	Summary     string
	Description string
	Deprecated  bool
	Tags        []string
	Params      []Param // sorted by (In, Name)
	RequestBody []MediaType
	Responses   []Response
}

type Param struct {
	Name        string
	In          string // path | query | header | cookie
	Description string
	Required    bool
	Schema      *Schema
}

type MediaType struct {
	Type   string // e.g. "application/json"
	Schema *Schema
}

type Response struct {
	Status      string // "200", "default"
	Description string
	Content     []MediaType
}

type NamedSchema struct {
	Name   string
	Schema *Schema
}

// Schema is the normalized schema shape. A component reference is a node with
// only Ref set — never inlined, which is what keeps cyclic specs finite.
type Schema struct {
	Ref           string // component schema name when this node is a $ref
	Type          string // single base type; "null" entries fold into Nullable
	Nullable      bool
	Format        string
	Description   string
	Properties    []Property
	Required      []string
	Items         *Schema
	Enum          []string // JSON-encoded scalars
	OneOf         []*Schema
	AnyOf         []*Schema
	AllOf         []*Schema
	Discriminator *Discriminator
	Examples      []string // JSON-encoded; 3.0's single example folds in here
	Default       string   // JSON-encoded; empty means unset
}

type Property struct {
	Name   string
	Schema *Schema
}

type Discriminator struct {
	PropertyName string
	Mapping      []MappingEntry
}

type MappingEntry struct {
	Value  string
	Schema string // component schema name
}

type SecurityScheme struct {
	Name   string // key in components/securitySchemes
	Type   string // apiKey | http | oauth2 | openIdConnect
	Scheme string // http only: bearer, basic, ...
	In     string // apiKey only: header | query | cookie
	Param  string // apiKey only: the header/query/cookie name
}
