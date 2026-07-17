// Package main is the E2 parser spike: pb33f/libopenapi vs speakeasy-api/openapi,
// scored on cycle-safe $ref resolution, 3.0/3.1 handling, parse time/memory,
// API ergonomics, and error quality. Verdict (libopenapi + vacuum) and the
// metrics table live in PRD.md's decision log; kept for the E2 write-up.
package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/pb33f/libopenapi"
	"github.com/pb33f/libopenapi/datamodel"
	"github.com/speakeasy-api/openapi/openapi"
	"github.com/speakeasy-api/openapi/references"
)

type result struct {
	parser     string
	elapsed    time.Duration
	heapMB     float64
	paths      int
	operations int
	schemas    int
	valErrs    int
	circulars  int
	fatal      string
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: spike <spec.yaml> [...]")
		os.Exit(2)
	}
	for _, path := range os.Args[1:] {
		data, err := os.ReadFile(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "read %s: %v\n", path, err)
			continue
		}
		fmt.Printf("\n=== %s (%d bytes) ===\n", filepath.Base(path), len(data))
		for _, r := range []result{runLibopenapi(path, data), runSpeakeasy(path, data)} {
			if r.fatal != "" {
				fmt.Printf("%-12s FATAL: %s (%.0fms)\n", r.parser, r.fatal, float64(r.elapsed.Milliseconds()))
				continue
			}
			fmt.Printf("%-12s %7.0fms %7.1fMB  paths=%-4d ops=%-4d schemas=%-4d valErrs=%-3d circular=%d\n",
				r.parser, float64(r.elapsed.Microseconds())/1000, r.heapMB,
				r.paths, r.operations, r.schemas, r.valErrs, r.circulars)
		}
	}
}

func heapNow() float64 {
	runtime.GC()
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return float64(m.HeapAlloc) / (1 << 20)
}

func runLibopenapi(path string, data []byte) result {
	r := result{parser: "libopenapi"}
	abs, _ := filepath.Abs(path)
	base := heapNow()
	start := time.Now()
	defer func() {
		if p := recover(); p != nil {
			r.fatal = fmt.Sprintf("panic: %v", p)
			r.elapsed = time.Since(start)
		}
	}()
	doc, err := libopenapi.NewDocumentWithConfiguration(data, &datamodel.DocumentConfiguration{
		BasePath:            filepath.Dir(abs),
		SpecFilePath:        abs,
		AllowFileReferences: true,
	})
	if err != nil {
		r.fatal = fmt.Sprintf("parse: %v", err)
		r.elapsed = time.Since(start)
		return r
	}
	model, err := doc.BuildV3Model()
	r.elapsed = time.Since(start)
	r.heapMB = heapNow() - base
	if err != nil {
		r.valErrs = -1
		r.fatal = truncate(fmt.Sprintf("build: %v", err))
		if model == nil {
			return r
		}
		r.fatal = "" // partial model still usable — count what we can
		r.valErrs = 1
	}
	if model.Model.Paths != nil {
		r.paths = model.Model.Paths.PathItems.Len()
		for _, item := range model.Model.Paths.PathItems.FromOldest() {
			r.operations += item.GetOperations().Len()
		}
	}
	if model.Model.Components != nil && model.Model.Components.Schemas != nil {
		r.schemas = model.Model.Components.Schemas.Len()
	}
	model.Index.GetRolodex().CheckForCircularReferences()
	r.circulars = len(model.Index.GetCircularReferences()) +
		len(model.Index.GetRolodex().GetSafeCircularReferences())
	return r
}

func runSpeakeasy(path string, data []byte) result {
	r := result{parser: "speakeasy"}
	ctx := context.Background()
	base := heapNow()
	start := time.Now()
	defer func() {
		if p := recover(); p != nil {
			r.fatal = fmt.Sprintf("panic: %v", p)
			r.elapsed = time.Since(start)
		}
	}()
	f, err := os.Open(path)
	if err != nil {
		r.fatal = err.Error()
		return r
	}
	defer f.Close()
	doc, valErrs, err := openapi.Unmarshal(ctx, f)
	if err != nil {
		r.fatal = truncate(fmt.Sprintf("unmarshal: %v", err))
		r.elapsed = time.Since(start)
		return r
	}
	abs, _ := filepath.Abs(path)
	idx := openapi.BuildIndex(ctx, doc, references.ResolveOptions{
		RootDocument:   doc,
		TargetDocument: doc,
		TargetLocation: abs,
	})
	r.elapsed = time.Since(start)
	r.heapMB = heapNow() - base
	if doc.Paths != nil {
		r.paths = doc.Paths.Len()
	}
	r.operations = len(idx.Operations)
	if doc.Components != nil && doc.Components.Schemas != nil {
		r.schemas = doc.Components.Schemas.Len()
	}
	r.circulars = idx.GetValidCircularRefCount() + idx.GetInvalidCircularRefCount()
	r.valErrs = len(valErrs) + len(idx.GetResolutionErrors())
	for _, e := range append(valErrs, idx.GetResolutionErrors()...) {
		fmt.Printf("             speakeasy err: %s\n", truncate(e.Error()))
	}
	return r
}

func truncate(s string) string {
	if len(s) > 300 {
		return s[:300] + "…"
	}
	return s
}
