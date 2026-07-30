package render

import (
	"fmt"
	"sort"
	"strings"
	"unicode"

	"github.com/oxmonty/biscuit/internal/config"
	"github.com/oxmonty/biscuit/internal/ir"
	"github.com/oxmonty/biscuit/internal/mapping"
)

// The view model precomputes every name, expression, and branch the templates
// need, so the templates themselves stay dumb string assembly. All ordering
// here follows the IR's sorted slices — the model must never introduce its
// own nondeterminism.

type repoModel struct {
	Binary      string
	Module      string
	Title       string
	Description string // full spec/config description; README's body paragraph
	Tagline     string // first line of Description; Makefile header and root.Long share this
	MakeTagline string // Tagline escaped for a single-quoted Makefile recipe argument
	Long        string // root.Long: Tagline + quickstart + docs link
	APIVersion  string
	BaseURL     string
	SpecPath    string
	SpecSHA     string

	Homebrew      bool
	InstallScript bool
	NPMPackage    string // npm package name upgrade.go targets; Binary+"-cli" unless overridden
	TapOwner      string // owner of the homebrew-tap repo casks publish into
	RepoOwner     string // GitHub owner derived from Module, "OWNER" if undetermined
	RepoName      string // GitHub repo name derived from Module, Binary+"-cli" if undetermined

	Resources    []*resourceView // top-level resource nodes
	AllResources []*resourceView // every node, depth-first — one output file each
	RootVerbs    []*verbView
	Ops          []*verbView // every verb incl. root verbs, in Ident-claim order

	Security           []*securityView // securitySchemes, sorted by Name
	SecurityWireParams []string        // apiKey scheme Param names, lowercased and deduped, for redaction

	// ManPages lists every page cobra's GenManTree emits for the generated
	// tree ({binary}-{chain}-{verb}.1), sorted — the cask manpage stanzas
	// must name each page individually, and a partial list would leave
	// nested commands unresolvable via man after brew install.
	ManPages []string
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
	Multipart  bool        // BodyFlags become multipart/form-data parts, not a JSON object

	SecurityLit string         // Go [][]string literal: this verb's security OR-list, passed to Client.do
	Pagination  *ir.Pagination // set when the operation matched a pagination scheme: the verb walks pages
}

// securityView is one components/securitySchemes entry's generated shape:
// the root flag and env var that resolve its credential, plus the wire
// details the client needs to apply it.
type securityView struct {
	Name       string // key in components/securitySchemes
	Flag       string // kebab root flag name
	VarName    string // Go local var in root.go holding the flag value
	EnvVar     string // {BINARY}_{SCHEME} upper-snake env var
	Type       string // apiKey | http | oauth2 | openIdConnect
	HTTPScheme string // http only: bearer, basic, ...
	In         string // apiKey only: header | query | cookie
	Param      string // apiKey only: the header/query/cookie name
}

type flagView struct {
	Name        string
	Description string
	Type        string // string | integer | number | boolean | json
	Format      string // schema format (e.g. "binary"); string-typed flags only
	Required    bool
	Repeated    bool
	Wire        string // path/query/header wire name; body flags reuse it as the multipart part name (leaf property)
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
		baseURL = substituteServerVariables(api.Servers[0])
	}

	description := cfg.Output.Description
	if description == "" {
		description = api.Description
	}
	tagline := firstLine(description)

	npmPackage := cfg.Distribution.NPM.Package
	if npmPackage == "" {
		npmPackage = binary + "-cli"
	}

	owner, repo := repoOwnerName(module, binary)
	m := &repoModel{
		Binary:        binary,
		Module:        module,
		Title:         api.Title,
		Description:   description,
		Tagline:       tagline,
		MakeTagline:   makeShellEscape(tagline),
		APIVersion:    api.APIVersion,
		BaseURL:       baseURL,
		SpecPath:      prov.SpecPath,
		SpecSHA:       prov.SpecSHA256,
		Homebrew:      cfg.Distribution.Homebrew,
		InstallScript: cfg.Distribution.InstallScript,
		NPMPackage:    npmPackage,
		TapOwner:      owner,
		RepoOwner:     owner,
		RepoName:      repo,
	}

	m.Security = m.buildSecurity(api.Security)
	m.SecurityWireParams = wireParams(m.Security)

	idents := identSet{}
	// pkg/cmd/upgrade.go always declares newUpgradeCmd; claiming it here
	// forces a spec operation that would derive the same Ident to Upgrade2
	// instead of colliding with the generated file.
	idents.claim("Upgrade")
	aliases := reservedAliases()
	for i := range api.Commands {
		m.Resources = append(m.Resources, m.buildResource(&api.Commands[i], nil, "pkg/cmd", idents, aliases, identSet{}))
	}
	for i := range api.RootVerbs {
		v := m.buildVerb(&api.RootVerbs[i], nil, idents)
		m.RootVerbs = append(m.RootVerbs, v)
	}
	m.ManPages = append(m.ManPages, m.Binary+".1")
	sort.Strings(m.ManPages)
	m.Long = m.buildLong()
	return m
}

// buildSecurity turns each components/securitySchemes entry into its
// generated root flag, env var, and Go local var name. Flag and var names
// are claimed against root.go.tmpl's own fixed identifiers so a scheme name
// like "header" can never collide with the persistent --header flag.
func (m *repoModel) buildSecurity(schemes []ir.SecurityScheme) []*securityView {
	if len(schemes) == 0 {
		return nil
	}
	flags := identSet{}
	for _, f := range []string{
		"base-url", "max-retries", "no-retries", "retry-max-elapsed-time", "timeout", "debug", "debug-unsafe", "header",
		"format", "transform", "transform-error", "format-error", "output", "include-headers", "max-pages",
	} {
		flags.claim(f)
	}
	vars := identSet{}
	for _, v := range []string{
		"baseURL", "maxRetries", "noRetries", "retryMaxElapsedTime", "timeout", "debug", "debugUnsafe", "headers", "credentials", "f", "cmd",
		"format", "transform", "transformError", "formatError", "output", "includeHeaders", "maxPages",
	} {
		vars.claim(v)
	}
	out := make([]*securityView, 0, len(schemes))
	for _, s := range schemes {
		kebabName := mapping.Kebab(s.Name)
		out = append(out, &securityView{
			Name:       s.Name,
			Flag:       flags.claim(kebabName),
			VarName:    vars.claim(lowerCamel(kebabName)),
			EnvVar:     upperSnake(m.Binary) + "_" + upperSnake(kebabName),
			Type:       s.Type,
			HTTPScheme: s.Scheme,
			In:         s.In,
			Param:      s.Param,
		})
	}
	return out
}

// wireParams collects apiKey schemes' Param names for the redaction set,
// lowercased and deduped — multiple schemes commonly reuse the same wire
// name (e.g. an "api_key" query param and cookie), which would otherwise
// emit a duplicate map key into the generated literal.
func wireParams(schemes []*securityView) []string {
	seen := map[string]bool{}
	var out []string
	for _, s := range schemes {
		if s.Type != "apiKey" {
			continue
		}
		lower := strings.ToLower(s.Param)
		if !seen[lower] {
			seen[lower] = true
			out = append(out, lower)
		}
	}
	sort.Strings(out)
	return out
}

// securityLit renders a verb's security OR-list as a Go [][]string literal
// for the do() call; do() treats a "nil"-length result identically to a
// declared-but-empty one, so nil is fine when the operation has none.
func securityLit(reqs []ir.SecurityRequirement) string {
	if len(reqs) == 0 {
		return "nil"
	}
	alts := make([]string, len(reqs))
	for i, r := range reqs {
		quoted := make([]string, len(r.Schemes))
		for j, s := range r.Schemes {
			quoted[j] = fmt.Sprintf("%q", s)
		}
		alts[i] = "{" + strings.Join(quoted, ", ") + "}"
	}
	return "[][]string{" + strings.Join(alts, ", ") + "}"
}

func upperSnake(kebab string) string {
	return strings.ToUpper(strings.ReplaceAll(kebab, "-", "_"))
}

func lowerCamel(kebab string) string {
	s := goExported(kebab)
	if s == "" {
		return s
	}
	r := []rune(s)
	r[0] = unicode.ToLower(r[0])
	return string(r)
}

// buildLong composes root.Long: the same Tagline the Makefile shows, then a
// short quickstart, then a docs link once the module resolves to a real
// GitHub repo. Plain text — cobra prints it above Usage on help, --help, -h,
// and bare invocation alike.
func (m *repoModel) buildLong() string {
	var b strings.Builder
	if m.Tagline != "" {
		b.WriteString(m.Tagline)
		b.WriteString("\n\n")
	}
	fmt.Fprintf(&b, "Quickstart:\n  %s %s\n  %s --help\n", m.Binary, m.ExampleUse(), m.Binary)
	if len(m.Security) > 0 {
		fmt.Fprintf(&b, "\nAuth: --%s or %s (see README)\n", m.Security[0].Flag, m.Security[0].EnvVar)
	}
	if m.RepoOwner != "OWNER" {
		fmt.Fprintf(&b, "\nDocs: https://github.com/%s/%s", m.RepoOwner, m.RepoName)
	}
	return strings.TrimRight(b.String(), "\n")
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
	m.ManPages = append(m.ManPages, m.Binary+"-"+strings.Join(chain, "-")+".1")

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
	m.ManPages = append(m.ManPages, m.Binary+"-"+strings.Join(append(append([]string(nil), chain...), v.Name), "-")+".1")
	vv := &verbView{
		Use:         v.Name,
		Short:       firstLine(v.Summary),
		Aliases:     v.Aliases,
		Deprecated:  v.Deprecated,
		Ident:       idents.claim(goExported(append(append([]string(nil), chain...), v.Name)...)),
		Method:      v.Method,
		Path:        v.Path,
		Multipart:   v.Multipart,
		SecurityLit: securityLit(v.Security),
		Pagination:  v.Pagination,
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
			Format:      f.Format,
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
				fv.Wire = f.BodyPath[len(f.BodyPath)-1]
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

// substituteServerVariables fills a server URL template's {name} placeholders
// with each variable's default value, e.g. "{region}.api.example.com" ->
// "us-east.api.example.com".
func substituteServerVariables(s ir.Server) string {
	url := s.URL
	for _, v := range s.Variables {
		url = strings.ReplaceAll(url, "{"+v.Name+"}", v.Default)
	}
	return url
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

// makeShellEscape prepares s for embedding as a single-quoted argument inside
// a Makefile recipe line. Make expands "$" of its own before the shell ever
// sees the line, so "$" must be doubled to survive as one literal "$"; each
// "'" would otherwise close the shell quote early, so it's closed, escaped,
// and reopened.
func makeShellEscape(s string) string {
	s = strings.ReplaceAll(s, "$", "$$")
	s = strings.ReplaceAll(s, "'", `'\''`)
	return s
}

// ExampleUse returns a concrete example command line for the README and the
// root.Long quickstart: the first top-level resource's first verb (walking
// into children when a resource only groups sub-resources), else the first
// root verb, else a bare --help. Deterministic — Resources/RootVerbs are
// already sorted at mapping time.
func (m *repoModel) ExampleUse() string {
	for _, r := range m.Resources {
		if use, ok := r.exampleUse(); ok {
			return use
		}
	}
	if len(m.RootVerbs) > 0 {
		return m.RootVerbs[0].Use
	}
	return "--help"
}

func (r *resourceView) exampleUse() (string, bool) {
	if len(r.Verbs) > 0 {
		return r.Name + " " + r.Verbs[0].Use, true
	}
	for _, c := range r.Children {
		if use, ok := c.exampleUse(); ok {
			return r.Name + " " + use, true
		}
	}
	return "", false
}
