package render

import (
	"testing"

	"github.com/oxmonty/biscuit/internal/ir"
)

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
