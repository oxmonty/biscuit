package render

import "testing"

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

func TestPathExpr(t *testing.T) {
	// given: a path with one matched and one unmatched parameter
	flags := []*flagView{{Wire: "petId", Field: "PetID"}}
	got := pathExpr("/pets/{petId}/toys/{toyId}", flags)
	want := `"/pets/" + url.PathEscape(req.PetID) + "/toys/" + "{toyId}"`
	if got != want {
		t.Errorf("pathExpr = %s, want %s", got, want)
	}
}
