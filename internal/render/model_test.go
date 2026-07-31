package render

import (
	"testing"

	"github.com/oxmonty/biscuit/internal/ir"
)

func TestBuildSecurityDedupesEnvVarsOnKebabCollision(t *testing.T) {
	// given: two securitySchemes whose names kebab identically
	m := &repoModel{Binary: "acme"}
	schemes := []ir.SecurityScheme{
		{Name: "apiKey", Type: "apiKey", In: "header", Param: "X-Api-Key"},
		{Name: "api_key", Type: "apiKey", In: "header", Param: "X-Api-Key-2"},
	}

	// when: building the security views
	got := m.buildSecurity(schemes)

	// then: both the flag and the env var are distinct, not just the flag
	if len(got) != 2 {
		t.Fatalf("buildSecurity returned %d views, want 2", len(got))
	}
	if got[0].Flag == got[1].Flag {
		t.Errorf("flags collide: both %q", got[0].Flag)
	}
	if got[0].EnvVar == got[1].EnvVar {
		t.Errorf("env vars collide: both %q", got[0].EnvVar)
	}
}

func TestSubstituteServerVariables(t *testing.T) {
	cases := []struct {
		name string
		srv  ir.Server
		want string
	}{
		{
			name: "no variables",
			srv:  ir.Server{URL: "https://api.example.com"},
			want: "https://api.example.com",
		},
		{
			name: "default substituted",
			srv: ir.Server{
				URL: "{protocol}://void.scalar.com/{path}",
				Variables: []ir.ServerVariable{
					{Name: "path", Default: ""},
					{Name: "protocol", Default: "https", Enum: []string{"https", "http"}},
				},
			},
			want: "https://void.scalar.com/",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := substituteServerVariables(tc.srv); got != tc.want {
				t.Errorf("substituteServerVariables(%+v) = %q, want %q", tc.srv, got, tc.want)
			}
		})
	}
}
