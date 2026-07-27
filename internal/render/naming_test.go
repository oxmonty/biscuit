package render

import (
	"strings"
	"testing"

	"github.com/oxmonty/biscuit/internal/config"
	"github.com/oxmonty/biscuit/internal/ir"
)

func TestGoExported(t *testing.T) {
	// given/then: punctuation splits, never leaks into the identifier
	cases := map[string]string{
		"chat-completions":  "ChatCompletions",
		"get?weird(chars)":  "GetWeirdChars",
		"v2.users_list":     "V2UsersList",
		"":                  "Op",
		"2fa":               "Op2fa",
	}
	for in, want := range cases {
		if got := goExported(in); got != want {
			t.Errorf("goExported(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestPkgNameAvoidsInvalidPackages(t *testing.T) {
	// given/then: keywords, main, and digit-led names are never package names
	cases := map[string]string{
		"pets":    "pets",
		"type":    "xtype",
		"main":    "xmain",
		"2fa":     "x2fa",
		"foo-bar": "foobar",
	}
	for in, want := range cases {
		if got := pkgName(in); got != want {
			t.Errorf("pkgName(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestBinaryNameIsASCIIAndTrimmed(t *testing.T) {
	// given/then: derived binary names survive as Go module path components
	// kebab splits acronym boundaries, so "OpenAI" derives as "open-ai" —
	// deterministic; output.binary pins anything nicer
	cases := map[string]string{
		"OpenAI API":       "open-ai",
		"Swagger Petstore": "swagger-petstore",
		"PokéAPI":          "pokapi",
		"The Foo API":      "foo",
		"":                 "api",
	}
	for in, want := range cases {
		if got := binaryName(in); got != want {
			t.Errorf("binaryName(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestAliasNameNeverKeyword(t *testing.T) {
	// given: a resource chain that lowers to a Go keyword
	if got := aliasName([]string{"type"}); got != "typex" {
		t.Errorf("aliasName(type) = %q, want typex", got)
	}
}

func TestRootUpgradeVerbRenamedOnCollision(t *testing.T) {
	// given: a spec whose only operation is a root-level "upgrade" verb —
	// its derived Ident would otherwise collide with pkg/cmd/upgrade.go's
	// own newUpgradeCmd
	api := &ir.API{
		Title:     "Test API",
		RootVerbs: []ir.Verb{{Name: "upgrade", Method: "GET", Path: "/upgrade"}},
	}
	cfg := &config.Config{Output: config.Output{Module: "example.com/test-cli"}}

	// then: buildModel claims "Upgrade" for the generated file first, so the
	// spec verb is deterministically renamed to avoid the clash
	m := buildModel(api, cfg, Provenance{})
	if len(m.RootVerbs) != 1 || m.RootVerbs[0].Ident != "Upgrade2" {
		t.Fatalf("root verb Ident = %+v, want Upgrade2", m.RootVerbs)
	}

	// then: the plan still renders without error — no duplicate newUpgradeCmd
	files, err := Render(api, cfg, Provenance{})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	root := string(findFile(t, files, "pkg/cmd/root.go").Contents)
	if want := "new" + m.RootVerbs[0].Ident + "Cmd"; !strings.Contains(root, want) {
		t.Errorf("root.go missing renamed constructor call %q:\n%s", want, root)
	}
}

func TestPathExpr(t *testing.T) {
	// given: a path with one matched and one unmatched parameter
	flags := []*flagView{{Wire: "petId", Field: "PetID"}}
	got := pathExpr("/pets/{petId}/toys/{toyId}", flags)
	want := `"/pets/" + url.PathEscape(req.PetID) + "/toys/" + "{toyId}"`
	if got != want {
		t.Errorf("pathExpr = %s, want %s", got, want)
	}
}
