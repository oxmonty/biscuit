# E3 — Mapping and config

_Code completed 2026-07-19 ([PR #7](https://github.com/oxmonty/biscuit/pull/7)); ships as `v0.1.0-alpha.5`. Append-only narrative — the spec lives in PRD.md, current state in ROADMAP.md._

## What shipped

The middle of the pipeline: the IR now derives a full command surface, and `biscuit generate --dry-run` shows it for any spec.

- `internal/mapping/tree.go`: resource/verb tree from paths (static segments carry nesting; tags only group root paths; `/api`/`/vN` mounts shed). Verbs from operationIds with stutter stripped against resource, tag, *and path-param* tokens plus naive singulars; shape verbs (`list`/`get`/`create`/`update`/`delete`) when the id is missing or reduces to a bare HTTP verb. POST-only leaves after a param become custom actions (`payment-intents confirm`) unless plural (`login-links` stays a sub-collection). Collisions rename deterministically with a diagnostic pointing at overrides.
- `internal/mapping/flags.go`: request bodies flatten into dot-notation flags with schema-adaptive depth — per-operation iterative deepening to the deepest expansion fitting a 64-flag budget, hard bound 8; cycles, arrays, and undiscriminated unions fall back to single json flags. Params map one-to-one (cookies deferred); multipart/form bodies flatten like JSON.
- `internal/mapping/union.go`: the ogen discriminator cascade (explicit → unique field → JSON type → enum value → opaque), annotating every oneOf flag as `ir.Union`.
- `internal/config`: strict `biscuit.yaml` loader — unknown keys rejected with line-precise errors, `version` gated, malformed config exits 2. Overrides (name, group, ignore, aliases, pagination) apply at mapping time; the in-spec `x-biscuit-*` mirror set feeds the same struct, sidecar winning field-wise.
- `biscuit init`: starter config seeded from doctor's gap analysis — commented override stubs keyed `"METHOD /path"` for id-less operations, self-validated back through the strict loader.
- `biscuit generate --dry-run`: tree + `--flags` + diagnostics + file plan, over the public `Generate(ctx, doc, cfg) → FilePlan` / `plan.Write(dir)` split. Generate runs doctor implicitly (`--strict` / `lint.min_grade` gates; one-line advisory summary unless `--quiet`).
- Doctor polish (E2 hand-off): counts folded into impact phrasing ("718 sites missing an example: mock-server responses and bench corpus weakened"), humane resolver diagnostics, TTY-only severity colors, `--format json`.
- stripe/openapi (6.3 MB, MIT) joined the ladder as the tree-derivation stress test.

## Evidence

- `go build ./... && go vet ./... && go test ./...` green; `golangci-lint run` (CI's linter) 0 issues.
- Whole ladder derives; stripe with **zero** mapping diagnostics after the heuristic fixes below.
- openai `chat completions create`: 46 static flags under the 64 budget — past Stainless's fixed two-level cap.
- Cascade on openai request bodies: 47 of 57 unions non-opaque (34 json-type, 6 enum-value, 4 unique-field, 3 discriminator).
- Bench (Apple M2, parse→tree): petstore 0.56 ms, train-travel 4.5 ms, openai 167 ms, stripe 378 ms — tree+flags added nothing measurable over E2's parse→IR.
- Demo flow live on pokeapi: `doctor` (10/100) → `init` scaffold → override renames `pokemon retrieve` to `pokemon catch` in `--dry-run`.

## Decisions made along the way

- **Config overrides: sidecar canonical + `x-biscuit-*` mirror set** (decision log). Fern and Speakeasy both converged on extensions-as-vocabulary with an overrides/overlay file as the non-owner carrier; Stainless is sidecar-only. No native Overlay support (pre-apply externally); an `extensions: true` toggle rejected — the namespace is opt-in by construction.
- **Depth policy: adaptive via iterative deepening** (decision log), resolving the fixed-cap-vs-adaptive open question with the 64/8 constants as ponytail-marked tunables.
- **POST on an instance path is `update`** — Stripe's whole API updates via `POST /v1/{resource}/{id}`, never PUT/PATCH.
- **oneOf flags stay single json values**; the cascade's verdict is an annotation, and union-of-variant-props expansion is the named upgrade path if the bench asks.
- **Public API stays minimal**: `Generate`/`FilePlan`/`Write` plus `Config` aliases. The IR (and the tree) remain internal; dry-run prints via internal packages until E4's render settles what the plan publicly carries.
- **`generate` without `--dry-run` is a usage error** until rendering ships — honest exit 2 over a silent no-op.

## Surprises

- **Stripe broke `spec.Load` on arrival**: libopenapi reports required-chain cycles as *build errors* ("infinite circular reference"), a path E2's pathological specs never exercised — they were wrongly blocking against the cycles-are-advisory policy. Fixed with a `required-cycle.yaml` regression spec.
- **Stripe's polymorphism lives in responses**, not request bodies — so flag-level unions on stripe are rare and the cascade's real workout came from openai's request schemas.
- **59 identical collisions on stripe** (create vs instance-POST) fell to zero from the one `update` heuristic — the diagnostics-as-signal loop working as designed.
- **OpenAI's chatkit operationIds genuinely end in `Method`** (`CancelChatSessionMethod`), leaving `list-method`-style verbs — spec noise, not a heuristic bug; exactly what overrides and doctor exist for.
- **The demo caught a flow bug**: discovery persists `spec.path` the moment doctor runs, which made the canonical doctor→init sequence refuse itself. `init` now regenerates a config that only caches `spec.path`.
- `/api/v2/` prefixes (pokeapi) nested everything under `api v2` until mount-point shedding learned to eat a leading `api` segment too.

## What this proved

The mapping heuristics hold on the two hardest real-world shapes we know (openai's polymorphic request bodies, stripe's deep nesting and POST-everything conventions) with overrides as the escape hatch, and the plan/write split is real — E4's render slots into `FilePlan.Files` with the command tree, flags, and unions already derived, sorted, and deterministic.

---

## Addendum — manual tier-1 bench preview vs openai-cli (2026-07-19)

A hand-run preview of E6's tier-1 (command-surface) comparison, against the Stainless parity target: [openai/openai-cli](https://github.com/openai/openai-cli) v1.4.0, built from source. Method: their `--help` trees vs our `generate --dry-run --flags` on the committed openai.yaml.

**Adapter finding (E6 groundwork).** openai-cli is *not* cobra — it's urfave/cli: colon-joined resource groups (`chat:completions:messages`), repeated array flags singularized (`--message`, `--tool`), and single-dash shorts (`-n`). The roadmap's "per-target adapter if it isn't stock cobra" caution is confirmed necessary; the tier-1 walker must normalize colon-vs-space nesting and singular-vs-plural flag names before diffing.

**Resource groups: 70/73 match (96%).**

| Bucket | Count | Note |
|---|---|---|
| Exact match | 36 | `chat:completions` ↔ `chat completions`, etc. |
| Match after stripping their `beta:` / `admin:` prefixes | 34 | those namespaces come from Stainless config, not the spec — sidecar `group` overrides reproduce them |
| Unmatched | 3 | `admin:organization:projects:*:roles` — spec drift: those endpoints don't exist in our committed snapshot (their 1.4.0 tracks a newer revision), so on shared spec content the match is total |

Our 122 groups also include ~50 intermediate parents (theirs flat-lists leaves only) plus endpoints from spec drift — their 1.4.0 tracks a newer spec revision than our committed snapshot.

**Verbs (`chat:completions`): 4/5 exact** (`create`/`list`/`update`/`delete`); theirs says `retrieve` where we derive `get`. Notably *our own* `batches` verb is `retrieve` (kept from the operationId) while chat's `getChatCompletion` reduced to a bare HTTP verb and took the shape name — a consistency question (normalize `get`→`retrieve`? respect the id?) that belongs in E6's corpus annotations (`expected: ours|theirs|either`).

**Flags (`chat completions create`): 100% coverage, both directions.** 29/36 of their long flags match ours byte-for-byte. Every delta is systematic, not a gap:

| Their shape | Our shape | Why |
|---|---|---|
| `--message`, `--tool`, `--function`, `--modality` | `--messages`, `--tools`, `--functions`, `--modalities` | they singularize repeated flags; we keep the spec's property name |
| `--audio`, `--prompt-cache-options`, `--web-search-options` (opaque JSON) | `--audio.format`, `--prompt-cache-options.ttl`, … `--web-search-options.user-location.approximate.city` | the adaptive depth policy expanding to depth 4 where their surface stops at the object — the deviation, visible |
| `-n` | `--n` | short-flag rendering |

**The preview caught a real bug.** Our first flag list had `top-logprobs` three times (`--top-logprobs`, `--body.top-logprobs`, `--body.top-logprobs-2`): allOf members redeclaring a property were merged by append, and the collision renamer dutifully suffixed the duplicates. Fixed (one name, one flag, later members win) with a regression test — the first concrete return on benching against Stainless output, before the harness even exists.
