package render

import (
	"fmt"
	"strings"

	"github.com/oxmonty/biscuit/internal/config"
	"github.com/oxmonty/biscuit/internal/ir"
)

// The view model precomputes every name, expression, and branch the templates
// need, so the templates themselves stay dumb string assembly. All ordering
// here follows the IR's sorted slices — the model must never introduce its
// own nondeterminism.

type repoModel struct {
	Binary      string
	Module      string
	Title       string
	Description string
	APIVersion  string
	BaseURL     string
	SpecPath    string
	SpecSHA     string

	Homebrew  bool
	TapOwner  string // owner of the homebrew-tap repo casks publish into
	RepoOwner string // GitHub owner derived from Module, "OWNER" if undetermined
	RepoName  string // GitHub repo name derived from Module, Binary+"-cli" if undetermined

	Resources    []*resourceView // top-level resource nodes
	AllResources []*resourceView // every node, depth-first — one output file each
	RootVerbs    []*verbView
	Ops          []*verbView // every verb incl. root verbs, in Ident-claim order
}

type resourceView struct {
	Name        string // kebab command name
	Short       string
	Ident       string // NewCmd<Ident> within its own package
	PkgName     string
	Dir         string // slash path under pkg/cmd
	ImportAlias string // globally unique alias parents and root import it as
	ImportPath  string
	Verbs       []*verbView
	Children    []*resourceView
}

type verbView struct {
	Use        string
	Short      string
	Aliases    []string
	Deprecated bool
	Ident      string // client method + Request struct + command constructor
	Method     string
	Path       string
	PathExpr   string // Go expression building the request path from req fields

	Flags      []*flagView
	PathFlags  []*flagView
	QueryFlags []*flagView
	HeadFlags  []*flagView
	BodyFlags  []*flagView // structured dot-notation body flags
	WholeBody  *flagView   // the single opaque body flag, when the body didn't flatten
}

type flagView struct {
	Name        string
	Description string
	Type        string // string | integer | number | boolean | json
	Required    bool
	Repeated    bool
	Wire        string // path/query/header wire name
	Field       string // path only: request struct field
	BodyPathLit string // body only: Go literal like []string{"a", "b"}

	// registration and read shapes, precomputed off Type/Repeated
	RegCall  string // String | Int64 | Float64 | Bool | StringArray
	DefLit   string // "" | 0 | false | nil
	GetCall  string // GetString | GetInt64 | ...
	WireExpr string // expression over v yielding the wire string form
}

// OutputDir resolves where the generated repo lands: output.dir when set,
// else ./{binary}-cli next to the working directory.
func OutputDir(api *ir.API, cfg *config.Config) string {
	if cfg.Output.Dir != "" {
		return cfg.Output.Dir
	}
	binary := cfg.Output.Binary
	if binary == "" {
		binary = binaryName(api.Title)
	}
	return binary + "-cli"
}

// repoOwnerName derives the GitHub owner/repo the release templates publish
// to from a module path like "github.com/acme/foo-cli" -> ("acme", "foo-cli").
// Modules hosted elsewhere get a placeholder the templates flag for editing.
func repoOwnerName(module, binary string) (owner, repo string) {
	const githubPrefix = "github.com/"
	if rest, ok := strings.CutPrefix(module, githubPrefix); ok {
		if o, r, ok := strings.Cut(rest, "/"); ok && o != "" && r != "" {
			return o, r
		}
	}
	return "OWNER", binary + "-cli"
}

func buildModel(api *ir.API, cfg *config.Config, prov Provenance) *repoModel {
	binary := cfg.Output.Binary
	if binary == "" {
		binary = binaryName(api.Title)
	}
	module := cfg.Output.Module
	if module == "" {
		module = "example.com/" + binary + "-cli"
	}
	baseURL := "http://localhost:8080"
	if len(api.Servers) > 0 {
		// ponytail: servers are URL-sorted in the IR, so this is the first
		// alphabetically, not the spec's primary; output.base_url if it matters
		baseURL = api.Servers[0].URL
	}

	owner, repo := repoOwnerName(module, binary)
	m := &repoModel{
		Binary:      binary,
		Module:      module,
		Title:       api.Title,
		Description: api.Description,
		APIVersion:  api.APIVersion,
		BaseURL:     baseURL,
		SpecPath:    prov.SpecPath,
		SpecSHA:     prov.SpecSHA256,
		Homebrew:    cfg.Distribution.Homebrew,
		TapOwner:    owner,
		RepoOwner:   owner,
		RepoName:    repo,
	}

	idents := identSet{}
	aliases := reservedAliases()
	for i := range api.Commands {
		m.Resources = append(m.Resources, m.buildResource(&api.Commands[i], nil, "pkg/cmd", idents, aliases, identSet{}))
	}
	for i := range api.RootVerbs {
		v := m.buildVerb(&api.RootVerbs[i], nil, idents)
		m.RootVerbs = append(m.RootVerbs, v)
	}
	return m
}

func (m *repoModel) buildResource(c *ir.Command, chain []string, parentDir string, idents, aliases, siblingDirs identSet) *resourceView {
	chain = append(append([]string(nil), chain...), c.Name)
	r := &resourceView{
		Name:    c.Name,
		Short:   firstLine(c.Description),
		Ident:   goExported(c.Name),
		PkgName: siblingDirs.claim(pkgName(c.Name)),
	}
	if r.Short == "" {
		r.Short = "Operations on " + strings.ReplaceAll(c.Name, "-", " ")
	}
	r.Dir = parentDir + "/" + r.PkgName
	r.ImportPath = m.Module + "/" + r.Dir
	r.ImportAlias = aliases.claim(aliasName(chain))
	m.AllResources = append(m.AllResources, r)

	for i := range c.Verbs {
		r.Verbs = append(r.Verbs, m.buildVerb(&c.Verbs[i], chain, idents))
	}
	childDirs := identSet{}
	for i := range c.Children {
		r.Children = append(r.Children, m.buildResource(&c.Children[i], chain, r.Dir, idents, aliases, childDirs))
	}
	return r
}

func (m *repoModel) buildVerb(v *ir.Verb, chain []string, idents identSet) *verbView {
	vv := &verbView{
		Use:        v.Name,
		Short:      firstLine(v.Summary),
		Aliases:    v.Aliases,
		Deprecated: v.Deprecated,
		Ident:      idents.claim(goExported(append(append([]string(nil), chain...), v.Name)...)),
		Method:     v.Method,
		Path:       v.Path,
	}
	if vv.Short == "" {
		vv.Short = v.Method + " " + v.Path
	}

	fields := identSet{}
	for i := range v.Flags {
		f := &v.Flags[i]
		fv := &flagView{
			Name:        f.Name,
			Description: firstLine(f.Description),
			Type:        f.Type,
			Required:    f.Required,
			Repeated:    f.Repeated,
			Wire:        f.Param,
		}
		fv.RegCall, fv.DefLit, fv.GetCall, fv.WireExpr = flagShapes(f)
		switch f.In {
		case "path":
			fv.Field = fields.claim(goExported(f.Name))
			vv.PathFlags = append(vv.PathFlags, fv)
		case "query":
			vv.QueryFlags = append(vv.QueryFlags, fv)
		case "header":
			vv.HeadFlags = append(vv.HeadFlags, fv)
		case "body":
			if len(f.BodyPath) == 0 {
				vv.WholeBody = fv
			} else {
				fv.BodyPathLit = goStringSliceLit(f.BodyPath)
				vv.BodyFlags = append(vv.BodyFlags, fv)
			}
		}
		vv.Flags = append(vv.Flags, fv)
	}

	vv.PathExpr = pathExpr(v.Path, vv.PathFlags)
	m.Ops = append(m.Ops, vv)
	return vv
}

// flagShapes maps an IR flag onto its cobra registration call, default
// literal, read call, and the expression turning the read value into wire
// string form. Repeated flags are always string arrays; items are coerced at
// insertion time.
func flagShapes(f *ir.Flag) (reg, def, get, wire string) {
	if f.Repeated {
		return "StringArray", "nil", "GetStringArray", "v"
	}
	switch f.Type {
	case "integer":
		return "Int64", "0", "GetInt64", "strconv.FormatInt(v, 10)"
	case "number":
		return "Float64", "0", "GetFloat64", "strconv.FormatFloat(v, 'g', -1, 64)"
	case "boolean":
		return "Bool", "false", "GetBool", "strconv.FormatBool(v)"
	default: // string, json
		return "String", `""`, "GetString", "v"
	}
}

// pathExpr renders "/pets/{petId}" into a Go expression over the request's
// path fields: "/pets/" + url.PathEscape(req.PetID). A {param} with no
// matching path flag stays literal — pathological, but never invalid Go.
func pathExpr(path string, pathFlags []*flagView) string {
	byWire := make(map[string]string, len(pathFlags))
	for _, f := range pathFlags {
		byWire[f.Wire] = f.Field
	}
	var parts []string
	rest := path
	for {
		open := strings.Index(rest, "{")
		if open < 0 {
			break
		}
		closing := strings.Index(rest[open:], "}")
		if closing < 0 {
			break
		}
		if lit := rest[:open]; lit != "" {
			parts = append(parts, fmt.Sprintf("%q", lit))
		}
		name := rest[open+1 : open+closing]
		if field, ok := byWire[name]; ok {
			parts = append(parts, "url.PathEscape(req."+field+")")
		} else {
			parts = append(parts, fmt.Sprintf("%q", "{"+name+"}"))
		}
		rest = rest[open+closing+1:]
	}
	if rest != "" || len(parts) == 0 {
		parts = append(parts, fmt.Sprintf("%q", rest))
	}
	return strings.Join(parts, " + ")
}

func goStringSliceLit(ss []string) string {
	quoted := make([]string, len(ss))
	for i, s := range ss {
		quoted[i] = fmt.Sprintf("%q", s)
	}
	return "[]string{" + strings.Join(quoted, ", ") + "}"
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		s = s[:i]
	}
	return strings.TrimSpace(s)
}
