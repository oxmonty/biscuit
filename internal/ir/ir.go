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
	Commands    []Command // resource/verb tree; children and verbs sorted by name
	RootVerbs   []Verb    // operations with no resource segments and no tag: {binary} verb
	Diagnostics []string  // mapping-level warnings (name collisions etc.), surfaced by dry-run
}

// Command is one resource node in the derived command tree:
// {binary} [resource [sub-resource...]] verb --flag value.
type Command struct {
	Name        string // kebab-case
	Description string // tag description when a tag matches this node's name
	Verbs       []Verb
	Children    []Command
}

// Verb is one invocable operation under a resource node.
type Verb struct {
	Name        string // kebab-case
	Method      string
	Path        string
	OperationID string // empty when the name was path-derived
	Summary     string
	Deprecated  bool
	Aliases     []string // kebab-case, from overrides
	Pagination  string   // pagination hint, from overrides
	Flags       []Flag   // sorted by Name
}

// Flag is one statically defined flag on a verb. Static definition is the
// constraint the whole layered argument design hangs on: completions and
// --help need the full set known at generation time.
type Flag struct {
	Name        string   // kebab-case; dots mirror body nesting (--address.city)
	In          string   // path | query | header | body
	BodyPath    []string // body only: original property names from the body root
	Type        string   // string | integer | number | boolean | json
	Description string
	Required    bool
	Repeated    bool     // array, passed as a repeated flag
	Enum        []string // JSON-encoded scalars
	Default     string   // JSON-encoded; empty means unset
	Union       *Union   // set when the schema is a oneOf: how variants are told apart
}

// Union is the discriminator-inference cascade's verdict on a oneOf:
// explicit discriminator → unique field → JSON type → enum value, opaque
// when nothing identifies the variants (ogen's cascade).
type Union struct {
	Kind     string // discriminator | unique-field | json-type | enum-value | opaque
	Property string // discriminating property; discriminator and enum-value kinds only
	Variants []UnionVariant
}

type UnionVariant struct {
	Value  string // discriminator value, identifying field name, JSON type, or enum value
	Schema string // component schema name when the variant is a $ref
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
	XBiscuit    Override // x-biscuit-* extension values carried in-spec
}

// Override adjusts how one operation maps into the command tree. It rides
// in-spec as x-biscuit-* extensions or in biscuit.yaml, sidecar winning
// field-wise (non-zero wins — a sidecar zero can't unset an extension value).
type Override struct {
	Name       string   // verb name
	Group      string   // whitespace-separated resource chain
	Ignore     bool     // drop the operation from the CLI
	Aliases    []string // extra verb names (sidecar only)
	Pagination string   // hint for the execution layer
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
