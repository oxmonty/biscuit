# E2: Spec ingestion and IR

a released biscuit parses any OpenAPI 3.x spec into a deterministic, immutable IR, and `biscuit doctor` grades it. `v0.1.0-alpha.4` → [Project structure](../../PRD.md#project-structure-the-generator), [Generation pipeline](../../PRD.md#generation-pipeline-and-concurrency-model), [Spec quality gate](../../PRD.md#spec-quality-gate-biscuit-doctor), [Spec discovery](../../PRD.md#spec-discovery)

Write-up: [docs/write-ups/E2-spec-ingestion-and-ir.md](../write-ups/E2-spec-ingestion-and-ir.md)

## Stories

- [x] Spike `pb33f/libopenapi` vs `speakeasy-api/openapi` in `spike/`, both parsing openai.yaml, scored against defined metrics (cycle-safe `$ref` resolution, 3.0/3.1 handling, parse time/memory, API ergonomics, governance/bus-factor); the winner and its linter sibling become the parser and doctor engine.
- [x] Parse and validate specs with the spike-chosen parser, resolving `$ref`s cycle-safely, with biscuit's own exit-code contract so scripts and pipelines get predictable failures.
- [x] Make `--spec` optional: discover the spec by well-known names (`openapi|swagger.{yaml,yml,json}`) in the current directory (flat scan — deeper enumeration ships with E8's discovery UX), then content-sniff its remaining yaml/json (first ~1 KB) for an `openapi:` root key; on multiple matches list candidates and prompt (plain stderr); persist the choice to `spec.path` in `biscuit.yaml` so discovery runs once — the config is the cache.
- [x] Define IR types with all collections sorted at mapping time, normalizing 3.0 and 3.1 (`nullable` vs `type` arrays, `example` vs `examples`) into one shape.
- [x] Integrate the spike-chosen linter (vacuum or Speakeasy's) as `biscuit doctor`: blocking correctness errors, advisory quality report with generation-impact notes, `--strict` / `lint.min_grade` gate.
- [x] Seed `testdata/specs` as a graded ladder: petstore (easy), a mid-size real-world 3.1 spec with oneOf/multi-auth/SSE (medium, e.g. Train Travel API), openai.yaml (hard), plus pathological cases including cyclic `$ref`s.
- [x] Add the generation benchmark (`gen_bench_test.go`) from day one.
