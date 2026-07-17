# E2 — Spec ingestion and IR

_Completed 2026-07-17 (code); ships as `v0.1.0-alpha.4`. Append-only narrative — the spec lives in PRD.md, current state in ROADMAP.md._

## What shipped

The front half of the generation pipeline: any OpenAPI 3.x spec loads into a deterministic, immutable IR, and `biscuit doctor` grades it.

- `internal/spec`: `Load()` via libopenapi — `BasePath` always set, resolver slog noise captured as diagnostics, `$ref`s inside vendor extensions advisory, cycles safe and reported. Blocking problems (unparseable spec, unresolvable `$ref`s, duplicate operationIds) return a typed `InvalidError`.
- The exit-code contract: `0` ok, `1` internal, `2` usage, `3` no spec, `4` spec invalid, `5` quality gate — every code proven live.
- `internal/ir` + `internal/mapping`: sorted-at-mapping-time IR; 3.0/3.1 normalized to one shape (`nullable` vs type arrays, `example` vs `examples`); component `$ref`s stay Ref nodes, which is what keeps cyclic specs finite; example/enum/default values render as canonical JSON.
- `biscuit doctor`: vacuum under a thin generation-relevant **biscuit ruleset** (style rules excluded so the grade means something), findings grouped per rule with generation impact + remediation, `--strict` / `lint.min_grade` gates.
- Spec discovery (MVP cut): flat cwd scan, well-known names then content-sniff, plain stderr prompt on multiple matches, choice persisted to `spec.path` — the config is the cache.
- `testdata/specs` graded ladder: petstore (3.0, easy), Train Travel API (3.1, medium), openai.yaml (3.1, 2.8 MB, hard), `pathological/` (cyclic, unresolvable, duplicate-opId).
- `gen_bench_test.go` benchmarking parse→IR; `biscuit.Load()` opens the public library API.
- Release plumbing folded prerelease channels into one: `biscuit-cli@next` cask + npm `next` dist-tag replace `@alpha`/`alpha`.

## Evidence

- `go build ./... && go vet ./... && go test ./...` green across `internal/{spec,ir,mapping,lint,cli}`.
- Ladder grades: petstore 94/100, train-travel 97/100, openai 10/100 (718 missing examples, 5 discriminator errors — the honest story).
- Live exit codes: bare `doctor` in a spec-less dir → 3; duplicate-opId spec → 4; `--strict` on petstore → 5; unknown flag → 2.
- Discovery round-trip: bare `biscuit doctor` printed `using spec openapi.yaml (recorded in biscuit.yaml)`; second run hit the cache silently.
- Bench (Apple M2): petstore 0.45 ms, train-travel 4.2 ms, openai.yaml 177 ms parse→IR.

## Decisions made along the way

- **Parser+doctor pair: libopenapi + vacuum** (spike kept in `spike/`). Scores on the hard spec:

  | Metric | libopenapi v0.38.7 | speakeasy v1.24.0 |
  |---|---|---|
  | openai.yaml parse+resolve | **99 ms / 41 MB** | 730 ms / 160 MB |
  | Cyclic `$ref`s | safe, 4 detected | safe, 2 detected, classified |
  | Native validation | none (vacuum's job) | caught dup opIds + real 3.1 type bugs |
  | 3.0→3.1 normalization | ours, in mapping | built-in `Upgrade()` |
  | Doctor sibling | vacuum: Spectral rulesets + report scoring | linter pkg, no grade |
  | Governance | single-author, mature | company-backed — **a biscuit competitor** |

  Performance and vacuum's scoring (what `min_grade` assumes) decided it; depending on a competitor's library sealed it.
- **Cycles are advisory, not blocking** — codegen breaks cycles with pointers; only unresolvable `$ref`s and duplicate opIds block.
- **Duplicate-opId check lives in `spec.Load`**, not deferred to vacuum: it's part of the exit-4 contract, one map pass.
- **Discovery scans cwd only** in this cut; the git-index/`WalkDir` enumeration ships with E8's UX.
- **`update` aliases `upgrade`** (synonyms in the wild); spec regeneration keeps no verb of its own — `biscuit generate` fetches a remote `spec.source` (fern/speakeasy precedent).
- **One prerelease channel, `next`** — alpha/beta/rc are version-string identifiers, never channels (npm `next` convention; VS Code stable+insiders; Flutter removed its dev channel). Renamed while the installed base was ~zero.

## Surprises

- libopenapi chases `$ref`s inside `x-*` vendor extensions (train-travel's `x-topics` points at a markdown file) and fails the build on them — extensions are opaque per spec. Classified advisory via `GetExtensionRefsSequenced()`.
- The 281-vs-296 operation-count discrepancy between parsers was webhooks: speakeasy's index folds webhook operations into `Operations`; neither parser was wrong.
- openai.yaml carries genuine 3.1 violations (`exclusiveMinimum: true`, boolean `$recursiveAnchor`) that speakeasy's strict validation flags and libopenapi ignores.
- vacuum's recommended ruleset graded openai.yaml 10/100 mostly on style noise (2 247 casing hits, 3 336 duplicate descriptions) — the biscuit ruleset exists because of this.
- Train Travel API is CC-BY-NC-SA licensed — flagged onto the OSS-solicitor release gate.
- npm's publish script derived the dist-tag from the semver identifier, so a future `-beta.1` would have silently minted a new `beta` channel — found while folding channels, fixed to constant `next`.

## What this proved

`spec → IR` is deterministic end to end (double-map equality on the hardest spec), the doctor speaks in generation impact rather than lint noise, and failures are contractual enough to script against. E3's mapping heuristics now have a sorted, normalized, cycle-finite structure to build the command tree on — and a benchmark watching the pipeline as render joins it.
