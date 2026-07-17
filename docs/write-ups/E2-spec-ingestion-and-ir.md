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

---

## Addendum — release day (2026-07-17, afternoon)

Appended after `v0.1.0-alpha.4` shipped; the sections above were written pre-release.

- **Released and verified end to end**: 8 assets on the GitHub Release; `biscuit-cli@next` cask installs and runs `doctor`; npm `next` dist-tag resolves via both `npm i -g` and cold `npx`; the shim pulled exactly one platform package. Stale `alpha` dist-tags removed manually.
- **CI failed post-merge on errcheck** (golangci-lint defaults; the repo has no config). Nine unchecked writer returns fixed with the `_, _ =` idiom; running CI's exact linter binary locally caught three sites CI's own log had truncated.
- **Release took ~30 min** — vacuum drags in `modernc.org/sqlite`, cross-compiled 7× on a cold cache every release because the ci and release workflows collide on the same setup-go cache key. Fixed with a dedicated `release-go-` cache + `restore-keys`; templated into E4's scope for generated repos. goreleaser Pro's `--split` deliberately not adopted (paid).
- **`update` became an alias of `upgrade`** (synonyms in the wild); spec regeneration keeps no verb — `biscuit generate` fetches a remote `spec.source` (fern/speakeasy precedent). Channel-aware `upgrade` with `--channel`/`--version` scoped into E4.
- **Prerelease channels folded to one**: `biscuit-cli@next` cask + npm `next` dist-tag replace `@alpha`/`alpha`; alpha/beta/rc live in the version string (npm `next` convention; VS Code stable+insiders; Flutter removed its dev channel). The publish script had been deriving the dist-tag from the semver identifier — a future `-beta.1` would have silently minted a `beta` channel; found and fixed during the fold.
- **linux/386 kept** after a staffed keep/drop debate: gh, goreleaser, and openai-cli (the parity target) all ship it; pure-Go so no cgo risk; seconds under the new cache. Revisit trigger: openai-cli dropping it.
- **Doctor output**: blank vacuum severities now label `info`, findings rank errors-first, and a summary footer explains the advisory-only policy (`112 errors … generation not blocked`). Deeper polish (humane resolver diagnostics, counts in impact lines, TTY colors, `--format json`) scoped as an E3 story.
- **Ladder grew**: museum (MIT, 3.1), galaxy (MIT, 3.1.1 — multi-auth/file-upload/webhooks and a real `Planet → Satellite` cycle), pokeapi (BSD-3, 98 GET-only ops — E3's mapping-scale rung and the doctor→init override-rescue demo). Callbacks remain uncovered anywhere; a hand-written pathological case is the cheap fix when needed.
- **Install-surface lesson**: npm has no caveats channel and postinstall echoes are banned by our own `--ignore-scripts` design — so install guidance must never live only in brew caveats; the binary and README are the surfaces every channel shares. Cask caveats collapse to `biscuit upgrade` pointers once E4 ships it.
