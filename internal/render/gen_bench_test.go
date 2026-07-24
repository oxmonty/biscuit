package render

import (
	"testing"

	"github.com/oxmonty/biscuit/internal/config"
	"github.com/oxmonty/biscuit/internal/mapping"
	"github.com/oxmonty/biscuit/internal/spec"
)

const ladder = "../../testdata/specs/"

var renderSink []File

// benchmarkRender measures the full pipeline: parse → IR → rendered plan —
// the number the PRD budgets at low single-digit seconds for openai.yaml.
func benchmarkRender(b *testing.B, path string) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		doc, err := spec.Load(path)
		if err != nil {
			b.Fatalf("Load(%s): %v", path, err)
		}
		api := mapping.Map(doc, nil)
		renderSink, err = Render(api, &config.Config{}, Provenance{SpecPath: path, SpecSHA256: "bench"})
		if err != nil {
			b.Fatalf("Render(%s): %v", path, err)
		}
	}
}

func BenchmarkRender_Petstore(b *testing.B) {
	benchmarkRender(b, ladder+"petstore.yaml")
}

func BenchmarkRender_TrainTravel(b *testing.B) {
	benchmarkRender(b, ladder+"train-travel.yaml")
}

func BenchmarkRender_OpenAI(b *testing.B) {
	benchmarkRender(b, ladder+"openai.yaml")
}

func BenchmarkRender_Stripe(b *testing.B) {
	benchmarkRender(b, ladder+"stripe.yaml")
}
