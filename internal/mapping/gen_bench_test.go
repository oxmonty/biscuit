package mapping

import (
	"testing"

	"github.com/oxmonty/biscuit/internal/ir"
	"github.com/oxmonty/biscuit/internal/spec"
)

var mapSink *ir.API

func benchmarkMap(b *testing.B, path string) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		doc, err := spec.Load(path)
		if err != nil {
			b.Fatalf("Load(%s): %v", path, err)
		}
		mapSink = Map(doc)
	}
}

func BenchmarkMap_Petstore(b *testing.B) {
	benchmarkMap(b, ladder+"petstore.yaml")
}

func BenchmarkMap_TrainTravel(b *testing.B) {
	benchmarkMap(b, ladder+"train-travel.yaml")
}

func BenchmarkMap_OpenAI(b *testing.B) {
	benchmarkMap(b, ladder+"openai.yaml")
}
