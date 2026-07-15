# biscuit

- Repo: https://github.com/oxmonty/biscuit.git

> A Go package and CLI that converts an OpenAPI 3.x spec into a complete, production-ready CLI repository (`{project}-cli`) — an open, self-hostable alternative to the acquired Stainless CLI generator.

```
biscuit generate --spec openapi.yaml --config biscuit.yaml --out ./foo-cli
```

Usable two ways:

- **Library**: `import "github.com/oxmonty/biscuit"` → `biscuit.Generate(ctx, spec, cfg)` returns a file plan
- **CLI**: `biscuit generate | doctor | bench | init | update`

---

## Roadmap

- [ ] **E1: Walking skeleton** — biscuit itself installs via Homebrew and npm and runs end-to-end, doing almost nothing yet. → [CI/CD](https://claude.ai/chat/a2569c67-14bf-4787-b753-1be6f48407a9#cicd), [Distribution](https://claude.ai/chat/a2569c67-14bf-4787-b753-1be6f48407a9#distribution) `v0.1.0-alpha.1`
    - [ ] Scaffold the generator repo: module layout, cobra root, `biscuit version` and `--help` + CLAUDE.md file
    - [ ] Wire CI and releases: release-please + goreleaser cross-platform builds to GitHub Releases.
    - [ ] Publish the Homebrew tap so `brew install` works.
    - [ ] Publish `biscuit-cli` to npm (shim + platform optionalDependencies) so `npx biscuit-cli` works.
    - [ ] _(Same mechanics later templated into generated CLIs in E10 — this epic proves them on biscuit itself.)_
- [ ] **E2: Spec ingestion and IR** — an OpenAPI 3.x spec parses into a deterministic, immutable IR, with quality diagnosed before generation. → [Project structure](https://claude.ai/chat/a2569c67-14bf-4787-b753-1be6f48407a9#project-structure-the-generator), [Generation pipeline](https://claude.ai/chat/a2569c67-14bf-4787-b753-1be6f48407a9#generation-pipeline-and-concurrency-model), [Spec quality gate](https://claude.ai/chat/a2569c67-14bf-4787-b753-1be6f48407a9#spec-quality-gate-biscuit-doctor)
    - [ ] Parse and validate specs with `pb33f/libopenapi`, resolving `$ref`s.
    - [ ] Define IR types with all collections sorted at mapping time.
    - [ ] Integrate vacuum as `biscuit doctor`: blocking correctness errors, advisory quality report with generation-impact notes, `--strict` / `lint.min_grade` gate.
    - [ ] Seed `testdata/specs` as a graded ladder: petstore (easy), a mid-size real-world spec with oneOf/multi-auth/SSE (medium, e.g. Train Travel API), openai.yaml (hard), plus pathological cases.
    - [ ] Add the generation benchmark (`gen_bench_test.go`) from day one.
- [ ] **E3: Mapping and config** — spec constructs map to a CLI command surface, overridable via `biscuit.yaml`. → [Argument parsing](https://claude.ai/chat/a2569c67-14bf-4787-b753-1be6f48407a9#argument-parsing), [Command grammar](https://claude.ai/chat/a2569c67-14bf-4787-b753-1be6f48407a9#command-grammar)
    - [ ] Derive the resource/verb tree from tags and paths, including nested sub-resources and stutter removal.
    - [ ] Implement flag flattening with the schema-adaptive dot-notation depth policy.
    - [ ] Implement the oneOf discriminator-inference cascade.
    - [ ] Load and apply `biscuit.yaml` overrides (names, aliases, hidden endpoints, pagination hints).
- [ ] **E4: Execution layer** — generated CLIs make correct, ergonomic API calls. → [Output control](https://claude.ai/chat/a2569c67-14bf-4787-b753-1be6f48407a9#output-control), [API semantics](https://claude.ai/chat/a2569c67-14bf-4787-b753-1be6f48407a9#api-semantics-handled-automatically), [Protocol scope](https://claude.ai/chat/a2569c67-14bf-4787-b753-1be6f48407a9#protocol-scope), [Additional design considerations](https://claude.ai/chat/a2569c67-14bf-4787-b753-1be6f48407a9#additional-design-considerations)
    - [ ] Map `securitySchemes` to auth flags and env vars.
    - [ ] Ship `--format` (incl. jsonl) and `--transform`/`--transform-error` via gjson.
    - [ ] Implement `@file` argument handling with sniffing and explicit prefixes.
    - [ ] Implement pagination (`--all`/`--max-pages` or transparent — see Open questions).
    - [ ] Stream SSE responses as JSONL when piped.
    - [ ] Add retries/backoff with `Retry-After`, the exit-code contract, and `--debug` with secret redaction.
- [ ] **E5: Repo scaffolding** — `biscuit generate` emits a complete repo that builds and releases. → [Generated repo structure](https://claude.ai/chat/a2569c67-14bf-4787-b753-1be6f48407a9#generated-repo-structure), [Distribution](https://claude.ai/chat/a2569c67-14bf-4787-b753-1be6f48407a9#distribution), [Regeneration safety](https://claude.ai/chat/a2569c67-14bf-4787-b753-1be6f48407a9#regeneration-safety)
    - [ ] Render the full template tree with generated-file markers and `internal/custom/`.
    - [ ] Emit goreleaser, release-please, and Homebrew tap configuration (proven in E1).
    - [ ] Generate README, shell completions (bash/zsh/fish/PowerShell), and man pages.
    - [ ] Add compile-the-output CI (`go build ./...` on generated repos) and generated smoke tests.
- [ ] **E6: Bench harness and test ladder** — parity vs openai-cli is measured and published, atop a graded integration suite. → [Validation strategy](https://claude.ai/chat/a2569c67-14bf-4787-b753-1be6f48407a9#validation-strategy-reverse-engineering-stainless) w bar chart pitting against openai-cli for results across: TBD (use mathplotlib)
    - [ ] Generate a mock server from any spec (routes + schema-valid canned responses + request recording).
    - [ ] Ship the easy/medium integration rungs: generate → build → golden requests vs mock, on every commit.
    - [ ] Implement help-tree diffing of command surfaces.
    - [ ] Run golden-request comparison against openai-cli as the hard rung (PRs touching mapping/templates).
    - [ ] Ship `biscuit bench --against <repo>` emitting the parity report.

---

_MVP line — E1–E6 ship as v0.1: an installable biscuit, `biscuit generate`, and a published parity number._

- [ ] **E7: MCP serve** — every generated CLI is an MCP server. → [MCP subcommand](https://claude.ai/chat/a2569c67-14bf-4787-b753-1be6f48407a9#mcp-subcommand)
    - [ ] Map operations to MCP tools and serve over stdio, then Streamable HTTP.
- [ ] **E8: Chat TUI** — one Bubble Tea interface backs `mcp chat`, `{binary} chat`, and interactive SSE. → [Protocol scope](https://claude.ai/chat/a2569c67-14bf-4787-b753-1be6f48407a9#protocol-scope), [MCP subcommand](https://claude.ai/chat/a2569c67-14bf-4787-b753-1be6f48407a9#mcp-subcommand)
    - [ ] Build the TUI with streaming and tool-call display (pi UX, Go implementation).
    - [ ] Add Anthropic and OpenAI providers behind a two-provider interface.
    - [ ] Detect chat-shaped endpoints and emit the `{binary} chat` REPL.
    - [ ] Route interactive-TTY SSE responses into the TUI.
    - [ ] Recreate UX from [pi](https://github.com/earendil-works/pi)
- [ ] **E9: Update pipeline** — spec changes open reviewable PRs on the CLI repo automatically. → [Update pipeline](https://claude.ai/chat/a2569c67-14bf-4787-b753-1be6f48407a9#update-pipeline)
    - [ ] Ship the pull-topology workflow with `.biscuit-state.yml` and App-token PRs.
    - [ ] Classify spec diffs into semver bumps feeding release-please.
    - [ ] Document the push topology (`repository_dispatch`) as a snippet.
- [ ] **E10: npm distribution for generated CLIs** — generated CLIs install via `npm`/`npx`. → [Distribution](https://claude.ai/chat/a2569c67-14bf-4787-b753-1be6f48407a9#distribution)
    - [ ] Template the shim, per-platform packages, and ordered OIDC publish job.
- [ ] **E11: Adoption** — Stainless-generated repos migrate to biscuit in one command. → [Stainless gaps and migration opportunity](https://claude.ai/chat/a2569c67-14bf-4787-b753-1be6f48407a9#stainless-gaps-and-migration-opportunity)
    - [ ] Ship `biscuit adopt --repo --spec` proposing a parity-maximizing config and taking over the release pipeline.
    - [ ] Add star history graph to README.md

**Future (considered, unscheduled)**: hosted generation API ([here](https://claude.ai/chat/a2569c67-14bf-4787-b753-1be6f48407a9#future-hosted-generation-api)); gRPC/proto frontend ([here](https://claude.ai/chat/a2569c67-14bf-4787-b753-1be6f48407a9#protocol-scope)).

---

## Workflow

The core loop — everything else in this doc serves it:

```
openapi.yaml + biscuit.yaml
  → doctor (lint gate) → parse (libopenapi) → IR (sorted, immutable)
  → map (heuristics + overrides) → render (FilePlan)
  → plan.Write(dir) → {project}-cli repo → its own release pipeline
```

### Spec quality gate (`biscuit doctor`)

Spec quality is the biggest determinant of generated-CLI quality, and the failure mode is silent: a spec missing `operationId`s doesn't break generation, it produces commands named `post-v1-users-id-activate`. Diagnose before generating — via **vacuum** (`daveshanley/vacuum`), the Go-native OpenAPI linter from the same author as libopenapi, built _on_ libopenapi (shares our parse tree, no re-parse), Spectral-ruleset-compatible, embeddable as a library, with report scoring. Don't roll our own rules engine — write a thin **biscuit ruleset** of generation-relevant rules on top.

Two severity categories, two policies:

- **Blocking (correctness)** — spec fails OAS schema validation, unresolvable `$ref`s, duplicate `operationId`s. Generation cannot produce a sane repo; hard fail with the doctor report.
- **Advisory (quality)** — missing `operationId`s (path-derived command names), untagged operations (flat command tree), missing descriptions (empty `--help` — guts the agent-usability story), no schema examples (weakens the auto-synthesized bench corpus _and_ mock responses), unnamed inline schemas. Degrades output but **never blocks by default**: most users don't control the spec they generate from, so a threshold refusal punishes the person who can't fix it.

The biscuit-specific layer: findings translate to **generation impact + remediation**, not generic lint output — "12 operations missing operationId → commands will be path-derived; fix in spec, or map names in `biscuit.yaml`" — spec fix if you own it, config override if you don't. `doctor` can emit a starter `biscuit.yaml` patching the gaps, which is also `adopt`'s first move on Stainless-refugee specs.

Gating for spec owners:

```yaml
# biscuit.yaml
lint:
  min_grade: 85        # or: biscuit generate --strict
  ruleset: .biscuit-rules.yaml   # optional custom vacuum ruleset
```

`biscuit doctor --spec openapi.yaml` runs standalone; `generate` runs it implicitly (blocking errors fail, advisories print unless `--quiet`).

### Regeneration safety

- Local, **deterministic** generation (clean diffs are non-negotiable for the update pipeline).
- Every generated file carries a header marker; regeneration touches only marked files.
- Reserved `internal/custom/` in output — emitted once, never overwritten.
- `biscuit.yaml` sidecar config for overrides (resource names, aliases, hidden endpoints, pagination hints, distribution). Open question: also honor in-spec `x-biscuit-*` extensions (ogen's model) or standard OpenAPI Overlays — sidecar suits users who don't control the spec; extensions suit spec owners.

### Update pipeline

Spec change → automatic PR on `{project}-cli`. Config points at where the spec lives:

```yaml
# biscuit.yaml
spec:
  source: github            # or: url
  repo: openai/openai-openapi
  path: openapi.yaml
  ref: main                 # or a release tag pattern
```

**Pull topology (v1)** — a scheduled workflow in the CLI repo owns everything; requires zero cooperation from the spec repo (most users won't control the spec they generate from):

```
cron (~6h) + workflow_dispatch
  → fetch spec from spec.source
  → checksum vs committed .biscuit-state.yml   # short-circuit if unchanged
  → biscuit generate (pinned biscuit version)
  → git diff → open PR (spec SHA/version in title + body)
```

`.biscuit-state.yml` (committed) records spec hash + provenance ("generated from spec @ abc123") — same role as Stainless's `.stats.yml` in openai-cli.

**Push topology (v2)** — for users owning both repos: spec repo fires `repository_dispatch` at the CLI repo on merge; same downstream job, instant instead of polled. Ship as a documented snippet, not core machinery.

Implementation traps:

- **Pin the biscuit version in the workflow.** Spec-update PRs and biscuit-upgrade PRs must be separate species, or tool-diff and spec-diff become indistinguishable in one PR.
- **`GITHUB_TOKEN` gotcha**: PRs opened by the default token don't trigger the repo's own CI — parity/build checks on update PRs silently never run. Generated workflow defaults to a GitHub App installation token (or PAT), with a comment explaining why (`peter-evans/create-pull-request` documents this trap).
- Determinism (above) is what makes these PRs reviewable — this pipeline is the reason it's non-negotiable.
- **Doctor on update**: the update PR includes the doctor delta — a spec update that _worsens_ quality (dropped descriptions, new untagged operations) is called out in the PR body before it degrades the CLI.

### Future: hosted generation API

Out of scope for now — local generation first — but kept cheap through library shape: `biscuit.Generate(ctx, specBytes, cfg) → FilePlan` is a **pure function** (no disk I/O, no side effects); writing is a separate `plan.Write(dir)` step. The future service is then a ~50-line HTTP handler returning the plan as a tarball, and CLI / GitHub Action / service are all thin clients of one function. The plan/write split also yields `--dry-run` for free. Costs nothing today; it's API discipline.

---

## Surfaces

Every place biscuit manifests, and what's on each:

|Surface|Primary user|What's on it|
|---|---|---|
|**biscuit CLI**|dev generating a CLI|`generate`, `doctor`, `bench`, `init`, `update`, `adopt` (E11); reads `biscuit.yaml`|
|**biscuit library**|tooling authors, future hosted API|`biscuit.Generate(ctx, spec, cfg) → FilePlan`, `plan.Write(dir)` — pure, no side effects|
|**Generated CLI**|end users of the target API|resource/verb command tree, `--format`/`--transform`, `@file`, pagination, SSE, auth, completions, man pages|
|**Generated MCP server**|agents / MCP clients|`{binary} mcp serve` — one tool per operation, stdio + Streamable HTTP|
|**Chat TUI**|humans, interactively|one Bubble Tea interface, three entry points: `mcp chat`, `{binary} chat`, interactive-TTY SSE|
|**GitHub Action**|CI|update pipeline: fetch spec → regenerate → PR with `.biscuit-state.yml` provenance|

**Surface-specific (deliberate non-parity):** `adopt` and `bench` exist only on the biscuit CLI (generation-time concerns, meaningless at runtime); the chat TUI only opens on an interactive TTY (piped contexts always get JSONL — scripts and agents depend on it); the library exposes plan/write but not the cobra command layer (the CLI is its first consumer, not its twin).

Future: Have a hosted site which will auth through GitHub to create `{binary}-cli` repo for user after pointing to a repository of their choice (we scan for openapi spec then use that)

---

## Validation strategy: reverse-engineering Stainless

Reference target: [openai/openai-cli](https://github.com/openai/openai-cli) (Stainless-generated, Go) built from [openai/openai-openapi](https://github.com/openai/openai-openapi/blob/main/openapi.yaml).

**Key insight:** Stainless CLIs are thin wrappers over their generated Go SDK, and Stainless applies a private config beyond the spec (resource grouping, naming, pagination semantics). Byte-level match % is therefore unachievable and the wrong goal. Parity is measured in three tiers instead:

1. **Command-surface parity** (primary, cheap, objective) — walk both binaries' `--help` trees; diff commands, flags, and argument types.
2. **Behavioral parity** (the metric that matters) — golden tests: run identical commands against a spec-generated mock server; diff the HTTP requests produced.
3. **Structural similarity** (tertiary) — file-tree and per-file similarity score.

Shipped as: `biscuit bench --against ./openai-cli --spec openapi.yaml --corpus cases.yaml --report report.md` → parity report. The published parity number is the project's credibility line ("verified against Stainless output").

The Stainless generator itself is closed source; its public generated repos are effectively the spec. The bench harness is the methodology.

### Test ladder

Easy (petstore, ~20 ops) and medium (mid-size spec with oneOf/multi-auth/SSE) rungs are **integration tests** — generate → `go build` → golden requests against a spec-generated mock, asserted against committed expectations — running on every commit with readable failures. Hard (openai-cli) is the **parity bench**, run on PRs touching `internal/mapping` or `templates/`. Same tier-2 machinery throughout; one `internal/bench` package serves the ladder, the parity bench, `biscuit adopt`'s analysis phase, and the smoke tests templated into generated repos.

### Bench mechanics

**Version pairing (step 0):** read openai-cli's `.stats.yml` for spec provenance, check out the matching spec SHA, record both SHAs in the report header; build both binaries from source.

**Tier 1 — help-tree walk:** recurse `binary <path...> --help` on both cobra binaries, parse commands and flag tables into a common JSON model (`{path, flags[{name,type,repeatable}], args}`), then set arithmetic: command recall/precision, per-shared-command flag recall/precision, rolled into an F1-style score. Allowlist `completion`/`help`/`version` out of scope.

**Tier 2 — golden requests against a mock:** both CLIs accept `--base-url`; point them at a spec-generated `httptest` server that records requests verbatim and returns minimal schema-valid 200s. Corpus = auto-synthesized invocations (required params filled from schema examples — scales to hundreds of endpoints free) + curated `cases.yaml` for what synthesis can't exercise (`@file`, stdin+flags merge, dot-notation, repeated arrays, pagination, SSE). Canonicalize before diffing: sort JSON keys and query params, normalize numbers, strip a header denylist (`User-Agent`, `X-Stainless-*`, timing-derived). Same corpus reused for response handling: identical canned payloads → diff stdout under `--format json` + exit codes; 4xx/5xx cases; canned SSE streams → diff emitted JSONL. Gotchas: run with piped stdio (forces non-TTY paths); mock responds fast with 200s so retries never fire.

**Tier 3:** file-tree Jaccard + averaged per-file similarity — will read low even at high tier-1/2 (their SDK dependency vs our inline client); its job is honesty ("we match behavior, not bytes"), not a target.

**Report & gating:** publish all three scores separately (headline weighting ≈ 40% surface / 50% behavior / 10% structure); `--min-parity` gates CI so the number only ratchets up. Corpus cases carry `expected: ours|theirs|either` so deliberate divergences from Stainless quirks don't tank the score — that annotation list _is_ the README's documented-deviations section.

---

## Features of generated CLIs

### Command grammar

```
{binary} [resource [sub-resource...]] method-name --flag value
```

Resource tree derived from tags/paths (must handle nested sub-resources, e.g. `/orgs/{org}/repos/{repo}/issues` → `orgs repos issues list`). Stutter removal: a "users" tag with a "list-users" operation maps to `users list`, not `users list-users` (Speakeasy's disclosed heuristic). Kebab-case commands and flags. Meaningful exit codes.

### Argument parsing

Layered design — Stainless's disclosed approach, adopted with one deviation. Constraint driving the design: tab completion and `--help` require **statically defined flags**, ruling out infinitely nested paths (`--messages.0.content.0.text`).

|Input shape|Syntax|
|---|---|
|Simple values|`--name "Alice"`|
|Nested objects|Dot notation: `--name.first "Abraham" --address.city "Springfield"`|
|Deep / polymorphic|Inline YAML or JSON per flag: `--name 'first: Abraham, last: Lincoln'`|
|Whole body|`--body '{"name":"Alice"}'` (Speakeasy pattern)|
|Arrays|Repeated flags: `--tag admin --tag reviewer`|
|Full payloads|stdin (JSON or YAML heredoc)|
|Templating|stdin + flags combined; **flags override stdin values**|
|Files|`@file.ext` with filetype sniffing (text → string, binary → base64); explicit `@file://` / `@data://` prefixes; `\@` escape|

**Deviation to explore:** Stainless caps dot notation at two levels. Biscuit should statically expand deeper where the schema is small and cap only where flag count explodes — a measurable "better than Stainless" claim via the bench harness. oneOf/allOf flag mapping borrows ogen's discriminator-inference cascade (explicit discriminator → field name → field type → field enum value).

### Output control

- `--format auto|json|jsonl|pretty|raw|yaml|explore` — syntax-highlighted JSON default, color auto-disabled when piped; `explore` = interactive TUI pager. JSONL is first-class (agents parse line-per-item).
- `--transform` / `--transform-error` — GJSON expressions (`tidwall/gjson`), plus `--format-error`.
- File-download endpoints: `--output/-o`, smart non-clobbering default filenames, pipe/redirection support.
- `--include-headers` to surface rate-limit/pagination/tracing headers (Speakeasy pattern).

### API semantics handled automatically

- **Pagination**: explicit `--all` / `--max-pages N` opt-in (Speakeasy's safer default against accidental thousand-page fetches) vs transparent walking — see Open questions; streaming endpoints wired to paging tools.
- **Auth**: mapped from `securitySchemes` → flags + env vars (multiple keys supported, e.g. standard + admin).
- `--base-url`, `--debug` (full HTTP request/response logging, redacted — see Additional design considerations).
- Runtime controls: `--timeout`, `--no-retries`, `--retry-max-elapsed-time`, arbitrary `--header` injection.

### Discoverability

- `--help` on every command (agent-usable with zero docs — Stainless's Claude Code / Spotify demo validates this).
- Man pages generated automatically.
- Shell completions: Bash, Zsh, fish, **and PowerShell**; Windows via `--flag=value`.

### MCP subcommand

`{binary} mcp` — a human CLI, an MCP server, and a chat client unified in one binary. Nuance for accuracy: Stainless _did_ offer MCP server generation from OpenAPI specs as a separate free product. What they never shipped is MCP **bundled into the CLI binary** — that unification is the differentiator.

Every operation's JSON schema is already a tool schema: `operationId` → tool name, description from spec, execution through the same client.

```
{binary} mcp serve --transport stdio|http
{binary} mcp chat  --provider anthropic|openai --api-key ...
{binary} mcp config
```

- **Transports**: per the MCP spec there are exactly two — **stdio** (client spawns the process; how `npx foo-cli mcp serve` gets wired into Claude Desktop etc.) and **Streamable HTTP** (remote endpoint; uses SSE internally for server→client streaming). The old standalone HTTP+SSE transport is deprecated/absorbed into Streamable HTTP — not worth implementing separately. gRPC is not an MCP transport. `--transport stdio|http` is therefore complete.
- `serve`: `modelcontextprotocol/go-sdk` (or `mark3labs/mcp-go`, more mature). stdio first.
- `chat`: in-process agent loop where the model's tools are the API's endpoints. **Do not port pi's TUI** (TypeScript, differential renderer — a project in itself). Steal pi's UX decisions (layout, streaming, tool-call display); build on **Bubble Tea + Lipgloss + Glamour**. Providers via `anthropic-sdk-go` + `openai-go` behind a tiny two-provider interface — no LiteLLM clone. SSE-streaming endpoints (below) render token-by-token in the TUI.
- npm distribution (below) makes `npx foo-cli mcp serve` the natural MCP client wiring.

### Protocol scope

**In scope: anything describable in OpenAPI 3.x — including SSE.** SSE endpoints are ordinary OpenAPI operations with a `text/event-stream` response content type (this is how OpenAI's own spec describes its streaming endpoints — already in the stress-test spec). Streaming output is **TTY-aware**: stdout is a pipe → plain JSONL, one event per line, composing with `--transform` (scripts and agents depend on this); stdout is an interactive terminal → open the same Bubble Tea chat-style TUI used by `mcp chat`, rendering tokens as they arrive. One TUI, multiple entry points. Additionally, biscuit detects **chat-shaped endpoints** at generation time (SSE response + messages-array request schema is the heuristic; confirmable/overridable in `biscuit.yaml`) and emits a top-level `{binary} chat` convenience command — a stateful REPL against the API's own chat endpoint, sharing all TUI machinery. Given LLM-style APIs, this is a must-have — and a concrete beat-Stainless point, since their CLI handled SSE poorly and their platform was flagged for it.

**Future work (considered, not scheduled):**

- **gRPC service → CLI** — requires a protobuf frontend (`.proto`/descriptor sets instead of OpenAPI), a second client stack (HTTP/2, proto wire format), and its own flag mapping: a sibling product, not a feature. The IR-centric design leaves the door open — a proto frontend as `internal/spec/proto` feeding the same mapping/render pipeline. Prior art to respect if attempted: `grpcurl` (ad-hoc calls), buf (codegen, breaking-change detection). Note: gRPC services exposing REST via gRPC-gateway transcoding ship an OpenAPI spec — biscuit handles those today with zero work.

**Out of scope:**

- **WebSockets** — not describable in OpenAPI (AsyncAPI territory); different frontend, different product.

---

## API design

_N/A for the MVP — biscuit exposes no service API._ The library contract plays this role: `biscuit.Generate(ctx, spec, cfg) → FilePlan` and `plan.Write(dir)` are the stable exported surface (see [Surfaces](https://claude.ai/chat/a2569c67-14bf-4787-b753-1be6f48407a9#surfaces)); everything else is `internal/` with no compatibility promise. This section gets filled in when the [hosted generation API](https://claude.ai/chat/a2569c67-14bf-4787-b753-1be6f48407a9#future-hosted-generation-api) graduates from Future — spec-first, with biscuit dogfooding itself on its own OpenAPI spec.

---

## Project structure (the generator)

```
biscuit/
├── biscuit.go                  # public library API: Load(), Generate(), Config
├── options.go
├── cmd/
│   └── biscuit/
│       └── main.go             # thin main; wires cobra commands
├── internal/
│   ├── cli/                    # cobra command defs: generate|doctor|bench|init|update
│   ├── spec/                   # libopenapi ingestion, validation, $ref resolution
│   ├── lint/                   # vacuum integration + biscuit ruleset + impact mapping
│   ├── ir/                     # intermediate representation types
│   ├── mapping/                # spec→IR heuristics + biscuit.yaml overrides
│   ├── render/                 # template execution, file planning, gofmt pass
│   ├── bench/                  # help-tree diff + golden-request harness + mock server
│   └── version/
├── templates/                  # embed.FS — mirrors the emitted repo tree
│   └── repo/
├── testdata/
│   ├── specs/                  # petstore (easy), mid-size (medium), openai.yaml (hard), pathological/
│   └── golden/                 # full golden output repos per spec, -update flag
├── examples/
└── .github/workflows/
```

Principles: IR between spec and templates (never render straight from spec); CLI is the first consumer of the public library API; template tree mirrors output tree; **CI compiles the generated output** (`go build ./...`) — the single most valuable check.

### Generation pipeline and concurrency model

Design for parallelism from day one; enable it only when the benchmark says so (even the OpenAI spec renders in low single-digit seconds sequentially; parsing via libopenapi dominates and isn't ours to parallelize; sqlc and ogen generate essentially sequentially).

Phases and their concurrency boundaries:

1. **Parse** — sequential (library-bound).
2. **Map spec → IR** — sequential _by design_: global name dedup and collision resolution over **sorted** inputs is what guarantees byte-identical output regardless of scheduling. Cheap phase anyway.
3. **Render** — IR is immutable from here; fan out per render unit (`errgroup` + worker pool). gofmt/goimports rides in the same worker.
4. **Write** — trivially parallel; paths are disjoint.

**The invariant that removes locking: exactly one render unit per output file.** Per-operation files parallelize embarrassingly; whole-spec aggregates (root command registry, client, README) are each one unit receiving the complete IR. No two workers ever touch the same path → nothing to synchronize. A template that seems to need multi-thread contributions is a signal to restructure it as one aggregate unit, not to add a mutex.

Determinism rules: sort every IR slice at mapping time; never let map iteration or goroutine completion order influence content; each file's bytes depend only on the IR (`renderFile(ir, unit) → []byte` is pure). Then parallel render is bit-identical to sequential by construction, and golden tests catch any violation instantly. Ship a `gen_bench_test.go` from day one (ogen has one to copy); flipping on the errgroup later is a ~15-line, zero-output-risk change.

---

## Generated repo structure

Mirrors openai-cli's shape — which is what makes the parity metric meaningful.

```
foo-cli/
├── cmd/foo/main.go
├── internal/           # client, iostreams, custom/ (never regenerated)
├── pkg/cmd/            # resource/verb command tree
├── npm/                # shim + platform packages (opt-in)
├── .github/workflows/  # release-please + goreleaser + npm publish
├── .goreleaser.yml
├── .biscuit-state.yml  # spec provenance
└── README.md           # generated
```

Emit gh-style patterns: **factory** (commands constructed from a Factory carrying HTTP client/config/IO — testable) and **iostreams** (central TTY detection, color, pager).

---

## Distribution

Of generated CLIs — all of this is templated into the output repo.

- **GoReleaser** on release-please PR merge: macOS (arm64/amd64), Linux (arm64/amd64/386), Windows (arm64/amd64), published to GitHub Releases.
- **Homebrew** tap, formula auto-updated. Tap token + macOS signing/notarization secrets in a `main`-scoped GitHub environment (Stainless's documented hardening).
- **npm** (leapfrog — Stainless only had this on roadmap): per-platform `optionalDependencies` pattern (esbuild/Biome/Turborepo), _not_ postinstall downloads — works with `--ignore-scripts`, proxies, lockfiles. Main package's `bin` is a ~20-line shim resolving `@scope/cli-${platform}-${arch}` with `require.resolve` fallback error (pnpm/Yarn PnP quirk). Publish order: platform packages → main. npm trusted publishing via OIDC.
- Opt-in via config:

```yaml
# biscuit.yaml
distribution:
  homebrew: true
  npm:
    package: "biscuit-cli"   # npm name ≠ command name; bin stays `biscuit`
```

**Naming note:** `biscuit` is taken on npm. Do **not** ship a misspelling (`biscut` — donates the typo funnel to a stranger's package, reads as an error, only fixes one registry). Use `biscuit-cli` or a scope; the `bin` field keeps the command `biscuit`. Check the abandoned-package dispute route in parallel. Also aware of: biscuit-auth (security tokens) — different space, coexistence fine, check before printing stickers.

---

## CI/CD

How biscuit itself is built, gated, versioned, and released — proven in E1 before any feature exists, then templated into generated repos in E5.

**Quality gates (every PR, all required):** lint (`golangci-lint`), unit tests, golden-output tests (`testdata/golden`, `-update` flag), the easy/medium integration rungs, **compile-the-output** (`go build ./...` on every generated golden repo — the gate that matters most), and the generation benchmark tracked for regressions. Post-E6: `biscuit bench` parity vs openai-cli runs on PRs touching mapping or templates, with the number surfaced in the PR.

**Versioning:** semver via release-please from conventional commits. Breaking changes to the CLI surface _of generated repos_ (removed/renamed commands or flags after a spec or template change) classify as major — the spec-diff semver rules (see [Additional design considerations](https://claude.ai/chat/a2569c67-14bf-4787-b753-1be6f48407a9#additional-design-considerations)) apply to biscuit's own template changes too.

```
merge to main → release-please PR accumulates changelog
merge release PR → tag → goreleaser (darwin/linux/windows, arm64+amd64)
              → GitHub Release → Homebrew tap formula bump
              → npm publish: platform packages, then biscuit-cli (OIDC trusted publishing)
```

**Secrets & signing:** publish credentials in a `main`-scoped GitHub environment readable only by the release workflow (Stainless's documented hardening); npm via OIDC, no long-lived tokens; Homebrew tap token scoped to the tap repo; macOS signing/notarization keys same environment. Generated repos inherit this exact posture from templates.

**Pinning contract:** generated repos' workflows pin their biscuit version; biscuit upgrades arrive as separate PRs from spec updates (see [Update pipeline](https://claude.ai/chat/a2569c67-14bf-4787-b753-1be6f48407a9#update-pipeline)) — so biscuit's own release cadence never contaminates spec-diff review downstream.

---

## Additional design considerations

- **Retries & rate limits (execution layer)**: Stainless SDKs auto-retry with exponential backoff and the CLI inherited it; biscuit's own client must ship retry policy (429/5xx, honor `Retry-After`, jittered backoff, `--max-retries`). Absence here would be an immediate parity regression. Good example of this utility but in typescript can be [found here](https://github.com/lamanIbrahimli/async-retry-with-backoff) but is missing isRetriable, etc.
- **Exit-code contract**: documented, stable mapping (0 success; distinct codes for usage error, auth failure, 4xx, 5xx, network). Scripts and agents branch on these; make it part of the generated README.
- **Secret redaction in `--debug`**: openai-cli merely _warns_ that debug logs may contain sensitive payloads. Biscuit redacts auth headers and known secret-shaped fields by default (`--debug-unsafe` to disable). Cheap, differentiating, security-reviewer-friendly.
- **Auth UX beyond env vars**: `{binary} auth login|whoami|status|logout` storing keys in the OS keychain (gh-style, `zalando/go-keyring`), plus named profiles/environments (`--profile staging`) in a config file. Written resolution precedence as a contract: **flags → env vars → OS keychain → config file** (Speakeasy's disclosed order). Env vars remain the CI path; keychain is the human path. Stainless CLIs were env-var-only.
- **Binary output guard + header surfacing**: block raw binary writes to an interactive TTY (direct users to `--output-file`/`--output-b64`/pipe), and support `--include-headers` to surface rate-limit/pagination/tracing headers (both Speakeasy patterns).
- **Spec-diff-driven semver**: the update pipeline classifies spec changes (added endpoint → minor; removed/renamed endpoint or flag → major; description-only → patch) and feeds release-please accordingly — buf-style breaking-change detection applied to CLI surface. Novel in this category; protects users' scripts from silent breakage.
- **Generated smoke tests**: the emitted repo includes its own test suite — spec-derived mock server + golden request tests — so `{project}-cli`'s CI validates every update PR without hitting the real API. (Same machinery as the bench harness, repackaged into output.)
- **No telemetry**: generated CLIs phone home to no one, stated explicitly in README. Trust signal, and a contrast with hosted-generator lineage.

---

## Stainless gaps and migration opportunity

Stainless announced (May 2026) it is joining Anthropic and **winding down its hosted products** — the entire forward roadmap is dead inventory, and every Stainless CLI customer needs a migration path. Speakeasy remains as the incumbent, with the same openness gap: its generator is account-gated SaaS (free tier: one SDK, 50 methods), not self-hostable.

**Promised, never shipped:**

- npm distribution ("working on support for more package managers, like npm" — CLI launch post). Biscuit ships it.
- Rust and Swift SDK targets (per third-party analysis); C# still beta at wind-down.

**Documented limitations, now frozen forever (= biscuit's differentiation list):**

- Deep-nesting arguments requiring JSON/YAML fallback (→ depth-policy deviation, above).
- REST/OpenAPI only: no WebSockets, SSE, or gRPC; generation gaps on advanced OpenAPI 3.1 JSON Schema (→ biscuit: libopenapi gives full 3.1; SSE in scope per Protocol scope).
- Hosted-only, dashboard-centric: no self-hosted option, config not diffable/version-controlled (→ biscuit is local-first with config in the repo — post-wind-down, the only option for orphaned users).
- Custom code injection limited to specific files (→ `internal/custom/` + marker-header contract is more generous).

**Migration as a product:** biscuit's parity-matched output isn't just validation methodology — it's a migration tool. Pitch line: _"Point biscuit at your existing Stainless-generated CLI repo and spec, and keep shipping releases."_ Candidate command: `biscuit adopt --repo ./foo-cli --spec openapi.yaml` (run bench, propose config that maximizes parity, take over the release pipeline).

---

## Tech stack

- **Go** (matching Stainless's rationale: native binaries, no runtime, instant startup, cross-compilation)
- `pb33f/libopenapi` (spec — see Open questions), `daveshanley/vacuum` (spec lint/doctor), `spf13/cobra` (commands), `tidwall/gjson` (transforms), `charmbracelet/bubbletea|lipgloss|glamour` (chat TUI + explore), MCP Go SDK, `zalando/go-keyring` (auth), GoReleaser + release-please (release), `text/template` + `embed.FS` (codegen)

---

## Reference codebases

|Project|Lesson|
|---|---|
|**sqlc**|Config-driven codegen hygiene; endtoend golden testdata; compile-the-output CI. Closest constraint match — check it first on every structure decision.|
|**ogen**|OpenAPI→IR rigor; Optional/Nullable semantics; oneOf discriminator inference; `x-` extension overrides. Also the compatibility benchmark: "if ogen parses it, biscuit must." Its `_testdata` specs are free test cases.|
|**kubebuilder**|Regeneration-safe scaffolding (file markers, machine- vs human-owned regions); plugin/versioning for migrating existing projects.|
|**gh (cli/cli)**|What output should feel like: factory pattern, iostreams, `pkg/cmd/<resource>/<verb>`.|
|**goreleaser**|Pipe-per-stage pipeline architecture; YAML config ergonomics.|
|**buf**|Product analogue in protobuf-land; breaking-change engine ≈ the bench harness.|
|**openai-cli**|Living spec of Stainless output — bench target _and_ template reference.|
|**speakeasy-api/openapi**|Genuinely OSS Go library: OpenAPI 3.0/3.1 parsing, 60+-rule linter, Overlays, Arazzo. Parser alternative to libopenapi; Overlays = a third answer to the config-override question. Speakeasy's CLI docs also document borrowable patterns (auth precedence, stutter removal, `--all` pagination, TTY binary guard).|
|**vacuum**|Go-native OpenAPI linter on libopenapi (shared parse tree); Spectral-compatible rulesets; report scoring. The engine under `biscuit doctor` — study its custom-ruleset API before writing the biscuit ruleset.|

---

## License

**GPLv2-or-later** (VLC's posture) with an **explicit generated-output exception**: output produced by biscuit, including code derived from biscuit's templates, is not covered by the GPL and belongs entirely to the user under any license they choose (GCC Runtime Library Exception / Bison precedent — without this clause every generated repo is arguably a GPL derivative and adoption dies). "Or later" is load-bearing, not stylistic: Apache 2.0 dependencies (libopenapi et al.) are incompatible with GPLv2-only but compatible via GPLv3. Register the "biscuit" trademark separately — it's license-independent leverage. Known trades, accepted: GPL doesn't cover SaaS-wrapping (AGPL territory) and some corporate legal teams avoid GPL tooling. Effectively irreversible once outside contributions arrive; the output-exception wording is worth an hour with an OSS solicitor before v0.1.

---

## Open questions

- Dot-notation depth policy: fixed cap vs schema-size-adaptive expansion (bench-measurable).
- Pagination mode: transparent walking vs explicit `--all`/`--max-pages` (Speakeasy's safer default) — decide before E4.
- Parser: `pb33f/libopenapi` vs `speakeasy-api/openapi` (spike both on openai.yaml in E2; each brings its linter sibling — vacuum vs Speakeasy's 60+-rule linter — so the choice is parser+doctor as a pair).
- Config overrides: `biscuit.yaml` vs `x-biscuit-*` extensions vs standard OpenAPI Overlays.
- Doctor default advisory set: which vacuum rules map to generation impact vs noise (tune on the test-ladder specs).
- npm package name: `biscuit-cli` vs scoped — check availability; dispute route for bare `biscuit` in parallel.
- Homebrew formula name availability for bare `biscuit`.