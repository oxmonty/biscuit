package render

import (
	"go/token"
	"strconv"
	"strings"
	"unicode"

	"github.com/oxmonty/biscuit/internal/mapping"
)

// goExported joins kebab-case segments into one exported Go identifier:
// ("chat", "completions", "create") → ChatCompletionsCreate. Split on every
// non-alphanumeric rune — spec names carry arbitrary punctuation.
func goExported(segments ...string) string {
	var b strings.Builder
	for _, seg := range segments {
		for _, part := range strings.FieldsFunc(seg, func(r rune) bool {
			return !unicode.IsLetter(r) && !unicode.IsDigit(r)
		}) {
			runes := []rune(part)
			b.WriteRune(unicode.ToUpper(runes[0]))
			b.WriteString(string(runes[1:]))
		}
	}
	s := b.String()
	if s == "" || !unicode.IsLetter([]rune(s)[0]) {
		s = "Op" + s
	}
	return s
}

// pkgName turns a kebab resource name into a Go package (and directory) name.
func pkgName(kebab string) string {
	s := strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, strings.ToLower(kebab))
	if s == "" || s[0] >= '0' && s[0] <= '9' || token.IsKeyword(s) || s == "main" {
		s = "x" + s
	}
	return s
}

// aliasName derives the import alias parents use for a resource package.
// Keywords can never be import aliases; reservedAliases pre-seeds the claim
// set with every identifier the templates themselves bind (fixed imports plus
// the cmd/f locals in scope where aliases are referenced).
func aliasName(chain []string) string {
	s := strings.ToLower(goExported(chain...))
	if token.IsKeyword(s) {
		s += "x"
	}
	return s
}

func reservedAliases() identSet {
	set := identSet{}
	for _, w := range []string{
		"cmd", "f", "client", "cmdutil", "factory", "iostreams",
		"cobra", "json", "fmt", "http", "url", "strconv", "os", "main",
		"version", // NewRootCmd's parameter shadows same-named imports
	} {
		set[w] = true
	}
	return set
}

// binaryName derives the default command name from the spec title:
// "OpenAI API" → "openai". ASCII only — the name seeds the default module
// path, and Go module paths reject anything else. The output.binary config
// key pins anything the derivation gets wrong.
func binaryName(title string) string {
	b := strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '-' {
			return r
		}
		return -1
	}, mapping.Kebab(title))
	b = strings.Trim(b, "-")
	b = strings.TrimPrefix(b, "the-")
	b = strings.TrimSuffix(b, "-api")
	if b == "" {
		b = "api"
	}
	return b
}

// ident allocates a unique name from base by appending 2, 3, ... on collision,
// in whatever deterministic order the caller iterates.
type identSet map[string]bool

func (set identSet) claim(base string) string {
	name := base
	for i := 2; set[name]; i++ {
		name = base + strconv.Itoa(i)
	}
	set[name] = true
	return name
}
