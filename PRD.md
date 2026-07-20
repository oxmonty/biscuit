# biscuit — PRD

Design spec for [ROADMAP.md](ROADMAP.md) — every epic links into a section here. This file is low-churn specification; ticks and sequencing live in the roadmap.

---

## Workflow

The core loop — everything else in this doc serves it:

```
openapi.yaml + biscuit.yaml
  → doctor (lint gate) → parse (libopenapi) → IR (sorted, immutable)
  → map (heuristics + overrides) → render (FilePlan)
  → plan.Write(dir) → {project}-cli repo → its own release pipeline
```

### Spec discovery

`--spec` is optional: run `generate`/`doctor` from a project dir and biscuit finds the spec.

Discovery order: well-known names first (`openapi|swagger.{yaml,yml,json}`), then content-sniff remaining yaml/json files (first ~1 KB) for an `openapi:` root key. In a git repo, enumerate via `git ls-files --cached --others --exclude-standard` — the index beats walking, and gitignore acts as a first-pass filter, not a hard exclusion; when git finds nothing (or outside a repo), fall back to `filepath.WalkDir` pruning `.git`/`node_modules`/`vendor`-style dirs. The fallback deliberately ignores .gitignore, so pipeline-generated gitignored specs are still found. The MVP cut (E2) enumerates the current directory only — one flat `ReadDir`, no recursion; the git-index and `WalkDir` machinery arrives with E8's discovery UX.

UX: a spinner on stderr once discovery exceeds ~150 ms (TTY only — the fast case stays flicker-free, pipes stay clean). On multiple matches, an interactive TTY gets a selector — plain numbered stderr prompt in the MVP cut, upgraded to a Bubble Tea countdown selector alongside the chat TUI — that defaults to the best-ranked candidate (conventional name at shallowest depth); non-TTY picks that default outright and prints what it chose.

The result persists to `spec.path` in `biscuit.yaml` so discovery runs once — the config is the cache, no hidden state. The same scanner serves the hosted site later ([Future](#future-hosted-generation-api)).

### Spec quality gate (`biscuit doctor`)

Spec quality is the biggest determinant of generated-CLI quality, and the failure mode is silent: a spec missing `operationId`s doesn't break generation, it produces commands named `post-v1-users-id-activate`. Diagnose before generating — via **vacuum** (`daveshanley/vacuum`), the Go-native OpenAPI linter from the same author as libopenapi, built _on_ libopenapi (shares our parse tree, no re-parse), Spectral-ruleset-compatible, embeddable as a library, with report scoring. Don't roll our own rules engine — write a thin **biscuit ruleset** of generation-relevant rules on top.

Two severity categories, two policies:

- **Blocking (correctness)** — spec fails OAS schema validation, unresolvable `$ref`s, duplicate `operationId`s. Generation cannot produce a sane repo; hard fail with the doctor report.
- **Advisory (quality)** — missing `operationId`s (path-derived command names), untagged operations (flat command tree), missing descriptions (empty `--help` — guts the agent-usability story), no schema examples (weakens the auto-synthesized bench corpus _and_ mock responses), unnamed inline schemas. Degrades output but **never blocks by default**: most users don't control the spec they generate from, so a threshold refusal punishes the person who can't fix it.

The biscuit-specific layer: findings translate to **generation impact + remediation**, not generic lint output — "12 operations missing operationId → commands will be path-derived; fix in spec, or map names in `biscuit.yaml`" — spec fix if you own it, config override if you don't. `doctor` can emit a starter `biscuit.yaml` patching the gaps — this is what `biscuit init` scaffolds from, and `adopt`'s first move on Stainless-refugee specs.

Gating for spec owners:

```yaml
# biscuit.yaml
lint:
  min_grade: 85        # or: biscuit generate --strict
  ruleset: .biscuit-rules.yaml   # optional custom vacuum ruleset
```

`biscuit doctor --spec openapi.yaml` runs standalone; `generate` runs it implicitly (blocking errors fail, advisories print unless `--quiet`).

**Repair split** — remediation lives on doctor, never on generate (`Generate` is pure; a generation that edits its own input would break determinism and the update pipeline's clean diffs). Config-side repair is `biscuit init`: gap analysis emitted as overrides, for specs the user doesn't own. Spec-side repair is `doctor --fix`: deterministic edits only — operationIds and tags written exactly as generate would derive them, so the spec converges on the CLI it already produces — as an opt-in, reviewable diff with the grade printed before and after. Fixes needing invention (descriptions, examples) are semantic authoring and belong to the spec-acquisition skill's LLM loop ([Future](#future-multi-api-toolsets-and-spec-acquisition)), gated behind this same doctor.

### Regeneration safety

- Local, **deterministic** generation (clean diffs are non-negotiable for the update pipeline).
- Every generated file carries a header marker; regeneration touches only marked files.
- Reserved `internal/custom/` in output — emitted once, never overwritten. Overwrite protection is not contract protection: `internal/custom/` gets a defined stable surface it may depend on (the generated client's exported API), and compile-the-output CI includes a repo with a representative custom file so a regeneration that breaks the contract fails in biscuit's CI, not the user's.
- `biscuit.yaml` sidecar config for overrides (resource names, aliases, hidden endpoints, pagination hints, distribution), schema-validated on load: unknown keys rejected with precise errors, a `version` key for forward migration — the config drives codegen, so a malformed config must fail loudly, never mis-generate. The sidecar is canonical — it also carries what extensions can't (lint, spec source, distribution) — and a small in-spec `x-biscuit-*` set (name, group, ignore, pagination hints) feeds the same override struct, sidecar winning on conflict: the field's converged shape (Fern's `x-fern-*`, Speakeasy's `x-speakeasy-*` — extensions as vocabulary, a file the non-owner controls as carrier; Stainless alone is sidecar-only). OpenAPI Overlays stay external — pre-apply them with overlay tooling before feeding biscuit.

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

`.biscuit-state.yml` (committed) records spec hash + provenance ("generated from spec @ abc123") — same role as Stainless's `.stats.yml` in openai-cli. Locally the same loop is just `biscuit generate`: with a remote `spec.source` configured it fetches before regenerating — no separate regeneration verb, matching the field ([`fern generate`](https://buildwithfern.com/learn/cli-api-reference/cli-reference/commands), [`speakeasy run`](https://www.speakeasy.com/docs/speakeasy-reference/cli/run) with the source in workflow config, [`stl builds`](https://www.stainless.com/docs/getting-started/quickstart-cli); nobody ships an update-from-source command). `update` is instead an alias of `upgrade`, the tool self-bump ([Distribution](#distribution)).

**Push topology (v2)** — for users owning both repos: spec repo fires `repository_dispatch` at the CLI repo on merge; same downstream job, instant instead of polled. Ship as a documented snippet, not core machinery.

Implementation traps:

- **Pin the biscuit version in the workflow.** Spec-update PRs and biscuit-upgrade PRs must be separate species, or tool-diff and spec-diff become indistinguishable in one PR. The biscuit-upgrade PR flow (bump the pinned version, regenerate, open its own PR) is what produces the second species.
- **`GITHUB_TOKEN` gotcha**: PRs opened by the default token don't trigger the repo's own CI — parity/build checks on update PRs silently never run. Generated workflow defaults to a GitHub App installation token (or PAT), with a comment explaining why (`peter-evans/create-pull-request` documents this trap).
- Determinism (above) is what makes these PRs reviewable — this pipeline is the reason it's non-negotiable.
- **Doctor on update**: the update PR includes the doctor delta — a spec update that _worsens_ quality (dropped descriptions, new untagged operations) is called out in the PR body before it degrades the CLI.

### Future: hosted generation API

Out of scope for now — local generation first — but kept cheap through library shape: `biscuit.Generate(ctx, specBytes, cfg) → FilePlan` is a **pure function** (no disk I/O, no side effects); writing is a separate `plan.Write(dir)` step. The future service is then a ~50-line HTTP handler returning the plan as a tarball, and CLI / GitHub Action / service are all thin clients of one function. The plan/write split also yields `--dry-run` for free. Costs nothing today; it's API discipline.

The hosted site rides the same shape: auth through GitHub, point at a repository, scan it for the spec with the [Spec discovery](#spec-discovery) scanner, and create the `{binary}-cli` repo for the user.

### Future: multi-API toolsets and spec acquisition

Considered, unscheduled; scope after E7 proves single-API `mcp serve`.

- **Multi-API toolsets (MCP gateway)** — one generated binary spanning several APIs (`{binary} stripe payment-intents confirm`, `{binary} datadog monitors list`), exposed as one curated MCP server: the open, self-hostable counter to Speakeasy's account-gated [Gram](https://www.speakeasy.com/blog/release-gram-beta) (hosted MCP servers with [curated toolsets](https://www.speakeasy.com/docs/mcp/build/toolsets/create-default-toolset), positioned as an [MCP gateway](https://speakeasy.com/product/gram) against Composio/Arcade/TrueFoundry). Convictions recorded up front: model it as multi-source config — an `apis:` list in `biscuit.yaml` with per-API spec/auth/allowlist — never a synthetic merged OpenAPI document (colliding component names, conflicting `servers` and auth blocks); **curation is the product, aggregation is the plumbing** — three real APIs is 500+ operations, past any MCP client's tool-count/context budget, so allowlisted toolsets carry the value (E8's client spike supplies the tool-count evidence); per-API auth namespacing (`STRIPE_API_KEY`, `DATADOG_API_KEY`) extends the securitySchemes mapping. Rides existing rails: per-API IRs merge under one extra namespace layer in the command tree, and `operations: ignore` is the curation seed.
- **Spec-acquisition skill** — a Claude Code skill (`.claude/skills/`, the setup-publishing delivery mechanism) that takes users from "no spec" to a graded one, in strict preference order: **fetch** (official published specs — Stripe, Datadog, cloud vendors — and registries like APIs.guru), **derive** (from server code or traffic captures), **author from intent** only as the grounded last resort. The loop closes through shipped machinery: draft → `doctor --format json` → fix findings → repeat until `lint.min_grade` passes, exit codes as the gate. Authored specs stay quarantined behind doctor plus (once the mock server lands) golden-request checks — an invented spec that grades well can still describe an API that doesn't exist, so fetch/derive always outrank author.

## Surfaces

Every place biscuit manifests, and what's on each:

|Surface|Primary user|What's on it|
|---|---|---|
|**biscuit CLI**|dev generating a CLI|`generate`, `doctor`, `bench`, `init`, `upgrade` (alias `update`), `adopt` (E11); reads `biscuit.yaml`|
|**biscuit library**|tooling authors, future hosted API|`biscuit.Generate(ctx, spec, cfg) → FilePlan`, `plan.Write(dir)` — pure, no side effects|
|**Generated CLI**|end users of the target API|resource/verb command tree, `--format`/`--transform`, `@file`, pagination, SSE, auth, completions, man pages, `upgrade` (alias `update`)|
|**Generated MCP server**|agents / MCP clients|`{binary} mcp serve` — one tool per operation, stdio + Streamable HTTP|
|**Chat TUI**|humans, interactively|one Bubble Tea interface, three entry points: `mcp chat`, `{binary} chat`, interactive-TTY SSE|
|**GitHub Action**|CI|update pipeline: fetch spec → regenerate → PR with `.biscuit-state.yml` provenance|

**Surface-specific (deliberate non-parity):** `adopt` and `bench` exist only on the biscuit CLI (generation-time concerns, meaningless at runtime); the chat TUI only opens on an interactive TTY (piped contexts always get JSONL — scripts and agents depend on it); the library exposes plan/write but not the cobra command layer (the CLI is its first consumer, not its twin).

---

## Validation strategy: reverse-engineering Stainless

Reference target: [openai/openai-cli](https://github.com/openai/openai-cli) (Stainless-generated, Go) built from [openai/openai-openapi](https://github.com/openai/openai-openapi/blob/main/openapi.yaml).

**Key insight:** Stainless CLIs are thin wrappers over their generated Go SDK, and Stainless applies a private config beyond the spec (resource grouping, naming, pagination semantics). Byte-level match % is therefore unachievable and the wrong goal. Parity is measured in three tiers instead:

1. **Command-surface parity** (primary, cheap, objective) — walk both binaries' `--help` trees; diff commands, flags, and argument types.
2. **Behavioral parity** (the metric that matters) — golden tests: run identical commands against a spec-generated mock server; diff the HTTP requests produced.
3. **Structural similarity** (tertiary) — file-tree and per-file similarity score.

Shipped as: `biscuit bench --against ./openai-cli --spec openapi.yaml --corpus cases.yaml --report report.md` → parity report. The published parity number is the project's credibility line ("verified against Stainless output").

The Stainless generator itself is closed source; its public generated repos are effectively the spec. The bench harness is the methodology. openai-cli is frozen since the Stainless wind-down — a stable target, but one that drifts from the live openai-openapi spec — so the published number always carries the paired spec/CLI SHAs and a date.

### Test ladder

Easy (petstore, ~20 ops) and medium (mid-size 3.1 spec with oneOf/multi-auth/SSE) rungs are **integration tests** — generate → `go build` → golden requests against a spec-generated mock, asserted against committed expectations — running on every commit with readable failures. Hard (openai-cli) is the **parity bench**, run on PRs touching `internal/mapping` or `templates/`. Same tier-2 machinery throughout; one `internal/bench` package serves the ladder, the parity bench, `biscuit adopt`'s analysis phase, and the smoke tests templated into generated repos.

### Bench mechanics

**Version pairing (step 0):** read openai-cli's `.stats.yml` for spec provenance, check out the matching spec SHA, record both SHAs in the report header; build both binaries from source.

**Tier 1 — help-tree walk:** recurse `binary <path...> --help` on both binaries, parse commands and flag tables into a common JSON model (`{path, flags[{name,type,repeatable}], args}`), then set arithmetic: command recall/precision, per-shared-command flag recall/precision, rolled into an F1-style score. Allowlist `completion`/`help`/`version` out of scope. Verify openai-cli's help output actually parses before committing to the metric; if it isn't stock cobra, tier 1 grows a per-target adapter.

**Tier 2 — golden requests against a mock:** both CLIs accept `--base-url`; point them at a spec-generated `httptest` server that records requests verbatim and returns minimal schema-valid 200s. Corpus = auto-synthesized invocations (required params filled from schema examples — scales to hundreds of endpoints free) + curated `cases.yaml` for what synthesis can't exercise (`@file`, stdin+flags merge, dot-notation, repeated arrays, pagination, SSE). Canonicalize before diffing: sort JSON keys and query params, normalize numbers, strip a header denylist (`User-Agent`, `X-Stainless-*`, timing-derived). Same corpus reused for response handling: identical canned payloads → diff stdout under `--format json` + exit codes; 4xx/5xx cases; canned SSE streams → diff emitted JSONL. Gotchas: run with piped stdio (forces non-TTY paths); mock responds fast with 200s so retries never fire.

**Tier 3:** file-tree Jaccard + averaged per-file similarity — will read low even at high tier-1/2 (their SDK dependency vs our inline client); its job is honesty ("we match behavior, not bytes"), not a target.

**Report & gating:** publish all three scores separately (headline weighting ≈ 40% surface / 50% behavior / 10% structure) as a per-tier bar chart in biscuit's README — SVG rendered by the bench harness itself, no Python dependency; `--min-parity` gates CI so the number only ratchets up. Corpus cases carry `expected: ours|theirs|either` so deliberate divergences from Stainless quirks don't tank the score — that annotation list _is_ the README's documented-deviations section.

### Bench metrics (cross-generator)

The parity tiers above are *relative* — scored against Stainless output. The README's lead chart is *absolute*: biscuit vs [Fern's CLI generator](https://buildwithfern.com/cli) vs Speakeasy's tooling, each generator's output scored against the same spec + spec-generated mock, so no target is privileged. Six metrics, all computable by the tier-1/tier-2 machinery with only the target binary swapped:

1. **Operation coverage** — % of the spec's operations invocable as commands (help-tree walk vs the IR's operation list).
2. **Request fidelity** — % of corpus invocations whose emitted HTTP (method, path, query, headers, body — canonicalized) matches what the spec prescribes, measured on the recording mock.
3. **Response handling** — % of canned-response cases rendered correctly: stdout under `--format json`, exit codes on 4xx/5xx, pagination walks, SSE-as-JSONL.
4. **Help completeness** (the agent-usability proxy) — % of commands carrying a summary, description, and typed flags; what an agent gets from `--help` alone.
5. **Footprint** — cold-start latency to `--help` and installed binary size; native Go vs runtime-dependent output shows up here.
6. **Generation cost** — wall-clock spec → buildable repo, and build success rate across the test ladder.

Reference targets: Fern publishes its own CLI generator output at [fern-api/petstore-cli](https://github.com/fern-api/petstore-cli) (Rust — irrelevant to tiers 1–2, which are black-box by design) on the same petstore spec as our easy rung. Speakeasy ships no CLI generator ([their examples](https://github.com/speakeasy-api/examples) cover SDKs, Terraform providers, tests, and MCP servers) — charted as zero and footnoted; the honest Speakeasy comparison is MCP-server output vs ours once E7 exists. Customer CLIs were surveyed (2026-07) for generated alternatives: every public CLI from Fern/Speakeasy customers (auth0-cli, twilio-cli, Kong/deck, airbyte abctl, vercel, cloudflare wrangler) is hand-written — no `.fernignore`/`.speakeasy` markers — and a global `.fernignore` code search surfaces only SDKs plus petstore-cli itself. Fern's CLI generator is too new to have public customer output; petstore-cli is the reference until that changes. Account-gated generation of further comparison repos (e.g. Fern on train-travel) is a spike task. Scores are dated and SHA-paired like the parity number.

**Live-API smoke tier (optional):** generate a CLI from a real provider's spec (openai.yaml), run a **read-only** corpus (GET-only allowlist, never mutating calls against a live account) with the token from an env var, against the same commands on the vendor CLI. Compares status codes and response shape, not bytes — live responses aren't deterministic. Credibility evidence ("both made identical real calls"), never a CI gate; the mock stays primary.

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

**Deviation from Stainless:** Stainless caps dot notation at two levels; biscuit's depth is schema-adaptive. Per operation, iterative deepening picks the deepest expansion (hard bound 8) whose total flag count fits a 64-flag budget — small schemas expand fully, exploding ones cap early, and subtrees cut by depth, budget, or a `$ref` cycle become single json-typed flags (the inline layer). The bench harness measures the resulting surface against Stainless's. oneOf/allOf flag mapping borrows ogen's discriminator-inference cascade (explicit discriminator → field name → field type → field enum value); the verdict rides on the flag as its union annotation for help, completions, and the execution layer.

Flag flattening is cycle-safe: recursive schemas (self-referencing trees, pervasive in 3.1 JSON Schema) get cycle detection and a hard depth bound, falling back to the inline-JSON/YAML layer past it. Each hard 3.1 construct (`type` arrays incl. null, `if/then/else`, `prefixItems`, enum-heavy schemas) carries an explicit handled-or-deferred decision in the mapping layer rather than an implicit crash. Multipart/form-data uploads map per-part (encoding + Content-Type), beyond bare `@file`.

### Output control

- `--format auto|json|jsonl|pretty|raw|yaml|explore` — syntax-highlighted JSON default, color auto-disabled when piped; `explore` = interactive TUI pager. JSONL is first-class (agents parse line-per-item).
- `--transform` / `--transform-error` — GJSON expressions (`tidwall/gjson`), plus `--format-error`.
- File-download endpoints: `--output/-o`, smart non-clobbering default filenames, pipe/redirection support.
- `--include-headers` to surface rate-limit/pagination/tracing headers (Speakeasy pattern).

### API semantics handled automatically

- **Pagination**: explicit `--all` / `--max-pages N` opt-in (Speakeasy's safer default against accidental thousand-page fetches) vs transparent walking — see Open questions; streaming endpoints wired to paging tools.
- **Auth**: mapped from `securitySchemes` → flags + env vars (multiple keys supported, e.g. standard + admin).
- `--base-url` (with `servers[].variables` templated into its default), `--debug` (full HTTP request/response logging, redacted — see Additional design considerations).
- Runtime controls: `--timeout`, `--no-retries`, `--retry-max-elapsed-time`, arbitrary `--header` injection.

### Discoverability

- `--help` on every command (agent-usable with zero docs — Stainless's Claude Code / Spotify demo validates this).
- Machine-readable help: the command tree and flag schemas dump as structured JSON (Fern ships this; cheap for biscuit since the IR is the schema, and it feeds the bench help-tree walker for free).
- Man pages generated automatically.
- Shell completions: Bash, Zsh, fish, **and PowerShell**; Windows via `--flag=value`.

### MCP subcommand

`{binary} mcp` — a human CLI, an MCP server, and a chat client unified in one binary. Positioning for accuracy: Stainless offered MCP server generation only as a separate free product, and Fern ships CLI + MCP bundled in one binary (early access, account-gated, since May 2026) — so the bundling itself is table stakes; biscuit's differentiator is being the **open, self-hostable** version of it.

Every operation's JSON schema is already a tool schema: `operationId` → tool name, description from spec, execution through the same client.

```
{binary} mcp serve --transport stdio|http
{binary} mcp chat  --provider anthropic|openai --api-key ...
{binary} mcp config
```

- **Transports**: per the MCP spec there are exactly two — **stdio** (client spawns the process; how `npx foo-cli mcp serve` gets wired into Claude Desktop etc.) and **Streamable HTTP** (remote endpoint; uses SSE internally for server→client streaming). The old standalone HTTP+SSE transport is deprecated/absorbed into Streamable HTTP — not worth implementing separately. gRPC is not an MCP transport. `--transport stdio|http` is therefore complete. The generated server declares a pinned MCP protocol revision rather than tracking "latest".
- **Client attachment** — MCP is the boundary where rich clients (Claude Code, Warp, pi, Cursor, VS Code) integrate; the two transports map to two attachment modes. *stdio*: the client owns the process — `claude mcp add acme -- acme mcp serve` makes Claude Code spawn the binary per session and speak JSON-RPC over stdin/stdout: handshake, `tools/list` (one tool per operation, names from `operationId`, descriptions from the spec), then `tools/call` mid-conversation, each call executing through the same HTTP client the CLI commands use; auth comes from env (`--env ACME_API_KEY=…` or inherited). *Streamable HTTP*: the server runs independently — `acme mcp serve --transport http` somewhere long-lived, attached with `claude mcp add --transport http acme http://host:8080/mcp`; this is how an already-running instance is linked rather than spawned. For API owners, the generated README documents the **sidecar deployment** of this mode: run the binary next to the API backend and reverse-proxy `your-api.com/mcp` to it (the go-sdk's Streamable HTTP server is an `http.Handler`, so the mount path is free) — a docs recipe, not tooling. This only serves internal/trusted networks, since the sidecar authenticates upstream with a single env-var key; making `/mcp` publicly consumable (OAuth on the HTTP transport, per-user credentials, multi-tenancy) is gateway-product territory, deferred to [multi-API toolsets](#future-multi-api-toolsets-and-spec-acquisition). Third-party consumers never need any of this — they wire over stdio. Generated repos also ship a project-scope `.mcp.json` committed at the repo root, so anyone opening the repo in Claude Code gets the CLI's tools wired automatically — zero-command onboarding for the whole team.
- `serve`: the official `modelcontextprotocol/go-sdk` (stable v1, tracks current spec revisions; `mark3labs/mcp-go` is the fallback). stdio first.
- `chat`: in-process agent loop where the model's tools are the API's endpoints. **Do not port pi's TUI** (TypeScript, differential renderer — a project in itself), and don't compete with rich MCP clients at chat UX either: the built-in surface only needs to be *serviceable* — anyone wanting pi's, Warp's, or Claude's experience points that client at `mcp serve` and gets it, improving as those clients improve, at zero maintenance cost to biscuit. Steal pi's UX decisions (layout, streaming, tool-call display); build minimal on **Bubble Tea + Lipgloss + Glamour**. Providers via `anthropic-sdk-go` + `openai-go` behind a tiny two-provider interface — no LiteLLM clone. SSE-streaming endpoints (below) render token-by-token in the TUI. A rich owned UI (hosted playground territory) would be a TypeScript companion npm package on the E10 rails — optional, never the built-in.
- npm distribution (below) makes `npx foo-cli mcp serve` the natural MCP client wiring.

### Protocol scope

**In scope: anything describable in OpenAPI 3.x — including SSE.** SSE endpoints are ordinary OpenAPI operations with a `text/event-stream` response content type (this is how OpenAI's own spec describes its streaming endpoints — already in the stress-test spec). Streaming output is **TTY-aware**: stdout is a pipe → plain JSONL, one event per line, composing with `--transform` (scripts and agents depend on this); stdout is an interactive terminal → open the same Bubble Tea chat-style TUI used by `mcp chat`, rendering tokens as they arrive. One TUI, multiple entry points. Additionally, biscuit detects **chat-shaped endpoints** at generation time (SSE response + messages-array request schema is the heuristic; confirmable/overridable in `biscuit.yaml`) and emits a top-level `{binary} chat` convenience command — a stateful REPL against the API's own chat endpoint, sharing all TUI machinery. Given LLM-style APIs, this is a must-have — and a concrete beat-Stainless point, since their CLI handled SSE poorly and their platform was flagged for it.

**Spec version scope:** libopenapi parses through OpenAPI 3.2; generation semantics target 3.0/3.1, normalized into one IR shape (`nullable` vs `type` arrays, `example` vs `examples`, `exclusiveMinimum` semantics). 3.2-only constructs and 3.1 `webhooks` are diagnosed by `doctor` and deferred from generation, not silently dropped.

**Future work (considered, not scheduled):**

- **gRPC service → CLI** — requires a protobuf frontend (`.proto`/descriptor sets instead of OpenAPI), a second client stack (HTTP/2, proto wire format), and its own flag mapping: a sibling product, not a feature. The IR-centric design leaves the door open — a proto frontend as `internal/spec/proto` feeding the same mapping/render pipeline. Prior art to respect if attempted: `grpcurl` (ad-hoc calls), buf (codegen, breaking-change detection). Note: gRPC services exposing REST via gRPC-gateway transcoding ship an OpenAPI spec — biscuit handles those today with zero work.

**Out of scope:**

- **WebSockets** — not describable in OpenAPI (AsyncAPI territory); different frontend, different product.

---

## API design

_N/A for the MVP — biscuit exposes no service API._ The library contract plays this role: `biscuit.Generate(ctx, spec, cfg) → FilePlan` and `plan.Write(dir)` are the stable exported surface (see [Surfaces](#surfaces)); everything else is `internal/` with no compatibility promise. This section gets filled in when the [hosted generation API](#future-hosted-generation-api) graduates from Future — spec-first, with biscuit dogfooding itself on its own OpenAPI spec.

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
│   ├── spec/                   # libopenapi ingestion, validation, $ref resolution, discovery
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
├── examples/                   # committed generated output (petstore-cli + one real-world spec)
└── .github/workflows/
```

Principles: IR between spec and templates (never render straight from spec); CLI is the first consumer of the public library API; template tree mirrors output tree; **CI compiles the generated output** (`go build ./...`) — the single most valuable check. Biscuit's own failures are contractual, so scripts and the update pipeline branch on them predictably. Exit codes: `0` success · `1` internal error · `2` usage error · `3` no spec found · `4` spec invalid (blocking correctness: schema validation, unresolvable `$ref`s, duplicate operationIds) · `5` quality gate failed (`--strict` / `lint.min_grade`).

### Generation pipeline and concurrency model

Design for parallelism from day one; enable it only when the benchmark says so (even the OpenAI spec renders in low single-digit seconds sequentially; parsing via libopenapi dominates and isn't ours to parallelize; sqlc and ogen generate essentially sequentially).

Phases and their concurrency boundaries:

1. **Parse** — sequential (library-bound).
2. **Map spec → IR** — sequential _by design_: global name dedup and collision resolution over **sorted** inputs is what guarantees byte-identical output regardless of scheduling. Cheap phase anyway.
3. **Render** — IR is immutable from here; fan out per render unit (`errgroup` + worker pool). gofmt/goimports rides in the same worker.
4. **Write** — trivially parallel; paths are disjoint.

**The invariant that removes locking: exactly one render unit per output file.** Per-operation files parallelize embarrassingly; whole-spec aggregates (root command registry, client, README) are each one unit receiving the complete IR. No two workers ever touch the same path → nothing to synchronize. A template that seems to need multi-thread contributions is a signal to restructure it as one aggregate unit, not to add a mutex.

Determinism rules: sort every IR slice at mapping time; never let map iteration or goroutine completion order influence content; each file's bytes depend only on the IR (`renderFile(ir, unit) → []byte` is pure). Golden output is also formatter-dependent: pin the Go toolchain (and thus gofmt) version in CI so golden diffs mean template changes, not Go upgrades. Then parallel render is bit-identical to sequential by construction, and golden tests catch any violation instantly. Ship a `gen_bench_test.go` from day one (ogen has one to copy); flipping on the errgroup later is a ~15-line, zero-output-risk change.

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
├── Makefile            # sectioned help headed by name/description from the spec's info
└── README.md           # generated
```

Emit gh-style patterns: **factory** (commands constructed from a Factory carrying HTTP client/config/IO — testable) and **iostreams** (central TTY detection, color, pager).

---

## Distribution

Of generated CLIs — all of this is templated into the output repo.

- **GoReleaser** on release-please PR merge: macOS (arm64/amd64), Linux (arm64/amd64/386), Windows (arm64/amd64), published to GitHub Releases.
- **Homebrew** tap, formula auto-updated. Tap token + macOS signing/notarization secrets in a `main`-scoped GitHub environment (Stainless's documented hardening). Homebrew 6 requires third-party taps to be trusted before first install (`brew tap --trust`, or a fully-qualified `brew install org/tap/name` which bypasses the prompt) — the generated README documents the trust step, and the friction is what makes the homebrew/core submission (E12) worth pursuing.
- **npm**: per-platform `optionalDependencies` pattern (esbuild/Biome/Turborepo), _not_ postinstall downloads — works with `--ignore-scripts`, proxies, lockfiles. Main package's `bin` is a ~20-line shim resolving `@scope/cli-${platform}-${arch}` with `require.resolve` fallback error (pnpm/Yarn PnP quirk). Publish order: platform packages → main. npm trusted publishing via OIDC: a brand-new package can't OIDC on its first publish (bootstrap locally), trusted-publisher configs must explicitly allow publishing, and `npm trust` configures multiple packages in one command (npm ≥ 11.5.1, Node ≥ 22.14). npm has no caveats surface and postinstall echoes are banned by this same pattern (and hidden by npm ≥ 7 anyway) — so install guidance must never live only in brew caveats: the binary (`--help`, `upgrade`) and README are the surfaces every channel shares.
- **Upgrade**: every generated CLI — and biscuit itself, where the mechanics are proven first — ships a channel-aware `{binary} upgrade`. Channel-aware in both senses: the *install* channel (Homebrew cellar path, npm global shim, bare binary) decides the mechanism — exec the package manager's own upgrade; self-download-and-swap only for bare binaries, so brew/npm always own the files they installed — and the *release* channel decides the target: a prerelease install upgrades within the `next` dist-tag / `@next` cask (alpha.3 → alpha.4), a stable install never crosses onto prereleases. There are exactly two channels — `stable` and `next` — for any prerelease maturity: alpha/beta/rc live in the version string, never in the channel list ([npm's `next` convention](https://docs.npmjs.com/cli/dist-tag/); VS Code ships stable+insiders; Flutter [*removed* its dev channel](https://github.com/flutter/flutter/issues/94962) — channels multiply cost, identifiers are free). Graduation is the one asymmetry: when a stable release is semver-newer than the newest prerelease, a `next` install upgrades to it and switches channels (cask swap / dist-tag change, announced) — `next` means early builds, not a permanent side-channel. Crossing the other way is explicit-only: `--channel stable|next` switches the tracked channel package-manager-natively, and `--version vX.Y.Z[-alpha.N]` pins an exact release (rollback included) by fetching that GitHub-release binary and self-swapping — which takes ownership from brew/npm, so it confirms before doing it (`deno upgrade --version` / `flutter channel` precedent). `update` is an alias of `upgrade` — the wild treats them as synonyms (rustup says update, deno and flutter say upgrade) and guessing wrong shouldn't regenerate anything. Once `upgrade` ships, cask caveats collapse to `Get started: {binary} --help` / `Upgrade: {binary} upgrade` on both channels — the binary knows its own channel, so install-manager text stops duplicating channel-specific commands. Spec regeneration has no verb of its own: it's `biscuit generate`, which fetches a remote `spec.source` first ([Update pipeline](#update-pipeline)).
- **`install.sh`**: a third channel needing no package manager at all — `curl -fsSL .../install.sh | bash` installs to `~/.{binary}/bin` and self-adds to PATH (bash/zsh/fish config detection). Proven on biscuit's own (`install.sh` at repo root), structure and PATH-detection logic adapted from [opencode's installer](https://github.com/sst/opencode) (MIT). `--channel stable|next` and `--version` mirror `upgrade`'s flags; defaults to `next` until a stable release exists (`/releases/latest` 404s pre-v1, so channel `next` reads `/releases` — newest first, prereleases included). No runtime dependency (no musl/AVX2-baseline detection needed — generated binaries are `CGO_ENABLED=0`, fully static, unlike opencode's Bun-based runtime).
- **Quickstart in help** (E4): `root.Long` carries the spec-derived description plus quickstart lines and docs link, plain text — no color, no logo, no TTY detection. Cobra's own template puts it above `Usage:` on `{binary} help`, `--help`, `-h`, *and* bare invocation for free, since a root command with no `Run` falls back to `cmd.Help()`. One help surface, everywhere, always plain; deliberately not a TTY-gated splash — help must read identically in terminals, pipes, and agent transcripts.
- Opt-in via config:

```yaml
# biscuit.yaml
distribution:
  homebrew: true
  npm:
    package: "biscuit-cli"   # npm name ≠ command name; bin stays `biscuit`
  install_script: true        # curl -fsSL .../install.sh | bash
```

**Naming note:** biscuit itself ships as `biscuit-cli` on both registries — bare `biscuit` is squatted on npm and is the Biscuit browser cask in homebrew/cask — with the `bin`/cask `binaries` field keeping the command `biscuit`. The abandoned-npm-package dispute route gets revisited alongside the homebrew/core submission (E12). Generated CLIs hitting the same collision follow the same pattern; never ship a misspelling (`biscut` — donates the typo funnel to a stranger's package, reads as an error, only fixes one registry). Also aware of: biscuit-auth (security tokens) — different space, coexistence fine, check before printing stickers.

---

## CI/CD

How biscuit itself is built, gated, versioned, and released — proven in E1 before any feature exists, then templated into generated repos in E4.

**Quality gates (every PR, all required):** lint (`golangci-lint`), unit tests, golden-output tests (`testdata/golden`, `-update` flag, pinned Go toolchain), the easy/medium integration rungs, **compile-the-output** (`go build ./...` on every generated golden repo, including one with a representative `internal/custom/` file — the gate that matters most), and the generation benchmark tracked for regressions. Post-E6: `biscuit bench` parity vs openai-cli runs on PRs touching mapping or templates, with the number surfaced in the PR.

**Versioning:** semver via release-please from conventional commits. Breaking changes to the CLI surface _of generated repos_ (removed/renamed commands or flags after a spec or template change) classify as major — the spec-diff semver rules (see [Additional design considerations](#additional-design-considerations)) apply to biscuit's own template changes too.

```
merge to main → release-please PR accumulates changelog
merge release PR → tag → goreleaser (darwin/linux/windows, arm64+amd64)
              → GitHub Release → Homebrew tap formula bump
              → npm publish: platform packages, then biscuit-cli (OIDC trusted publishing)
```

**Release speed:** the release job carries its own cross-compile build cache (`release-go-` key with a `restore-keys` prefix fallback) — the ci workflow's cache collides on the same go.sum key and only ever holds linux/amd64 objects, so without the dedicated key every release rebuilds all targets cold. Warm releases recompile only changed dependencies; within the job, goreleaser already parallelizes across cores. Templated into generated repos' release workflow by default — same 7-target build, same win. goreleaser Pro's `--split` per-platform matrix stays off the default path (a paid dependency generated repos shouldn't inherit); revisit only if warm-cache releases are still too slow.

**Secrets & signing:** publish credentials in a `main`-scoped GitHub environment readable only by the release workflow (Stainless's documented hardening); npm via OIDC, no long-lived tokens; Homebrew tap token scoped to the tap repo; macOS signing/notarization keys same environment. Generated repos inherit this exact posture from templates.

**Pinning contract:** generated repos' workflows pin their biscuit version; biscuit upgrades arrive as separate PRs from spec updates (see [Update pipeline](#update-pipeline)) — so biscuit's own release cadence never contaminates spec-diff review downstream.

---

## Additional design considerations

- **Retries & rate limits (execution layer)**: Stainless SDKs auto-retry with exponential backoff and the CLI inherited it; biscuit's own client must ship retry policy (429/5xx, honor `Retry-After`, jittered backoff, `--max-retries`). Absence here would be an immediate parity regression. Good example of this utility but in typescript can be [found here](https://github.com/lamanIbrahimli/async-retry-with-backoff) but is missing isRetriable, etc.
- **Exit-code contract**: documented, stable mapping (0 success; distinct codes for usage error, auth failure, 4xx, 5xx, network). Scripts and agents branch on these; make it part of the generated README.
- **Request preview**: a `--dry-run` on generated CLIs printing the composed HTTP request without sending it (Fern ships this) — cheap given the client design, invaluable for debugging and for authoring bench corpus cases.
- **No telemetry, ever — measurement is passive**: biscuit and generated CLIs phone home to nothing — no usage pings, no crash reporting, no update checks beyond the explicit `upgrade` command. A hard differentiator against account-gated generators, and table stakes for the self-hosted positioning. Adoption is measured registry-side instead, where it's free and consentless: GitHub release-asset `download_count` (covers brew casks and `install.sh`, which both fetch from Releases), the npm downloads API, aggregated on a schedule into a README badge. Trends over absolutes — npm numbers include mirrors and bots. Generated CLIs are structurally out of the question regardless: their API vendor already sees every call server-side, and templating trackers into other people's products is a trust breach no data justifies. If binary telemetry is ever revisited it follows [Go's transparent model](https://go.dev/blog/gotelemetry) — opt-in, counters only, no IDs, collected data public — and honors [`DO_NOT_TRACK`](https://donottrack.sh/).
- **Secret redaction in `--debug`**: openai-cli merely _warns_ that debug logs may contain sensitive payloads. Biscuit redacts auth headers and known secret-shaped fields by default (`--debug-unsafe` to disable). Cheap, differentiating, security-reviewer-friendly.
- **Auth UX beyond env vars**: `{binary} auth login|whoami|status|logout` storing keys in the OS keychain (gh-style, `zalando/go-keyring`), plus named profiles/environments (`--profile staging`) in a config file. Written resolution precedence as a contract: **flags → env vars → OS keychain → config file** (Speakeasy's disclosed order). Env vars remain the CI path; keychain is the human path. Stainless CLIs were env-var-only. Unscheduled — tracked on the roadmap's Future line.
- **Binary output guard + header surfacing**: block raw binary writes to an interactive TTY (direct users to `--output-file`/`--output-b64`/pipe), and support `--include-headers` to surface rate-limit/pagination/tracing headers (both Speakeasy patterns).
- **Spec-diff-driven semver**: the update pipeline classifies spec changes (added endpoint → minor; removed/renamed endpoint or flag → major; description-only → patch) and feeds release-please accordingly — buf-style breaking-change detection applied to CLI surface. Novel in this category; protects users' scripts from silent breakage.
- **Generated smoke tests**: the emitted repo includes its own test suite — spec-derived mock server + golden request tests — so `{project}-cli`'s CI validates every update PR without hitting the real API. (Same machinery as the bench harness, repackaged into output.)
- **No telemetry**: generated CLIs phone home to no one, stated explicitly in README. Trust signal, and a contrast with hosted-generator lineage — Fern's "local" generation still verifies your org against their servers; biscuit never makes a network call you didn't ask for.

---

## Competitive landscape

Stainless announced (May 2026) it is joining Anthropic and **winding down its hosted products** — the entire forward roadmap is dead inventory, and every Stainless CLI customer needs a migration path. The field that remains:

- **Speakeasy** — the funded incumbent, and a direct CLI competitor, not just an SDK tool: it generates Go/Cobra CLIs (wrapping its generated Go SDK, GoReleaser included) and, separately, MCP servers from OpenAPI. The openness gap is the wedge: account-gated SaaS (free tier ~one SDK/CLI up to 50 methods, paid per-language), and while its generator CLI runs air-gapped it still needs a license — not self-hostable.
- **Fern** — shipped a CLI generator in early access (May 2026) whose output is a single statically-linked Rust binary that is also an MCP server over stdio or Streamable HTTP, distributed via npm and GitHub Releases (Homebrew "coming soon"). Its "local generation" runs in Docker but still requires a `FERN_TOKEN` org-verification call — account-gated, not open.
- **restish** (`rest-sh/restish`, with its archived ancestor `danielgtaylor/openapi-cli-generator`) — the closest OSS prior art: an actively-maintained generic runtime CLI auto-configured per-API from OpenAPI 3, rather than a generated per-API repo. Different model (one shared binary vs an owned, releasable repo), but its author's ["Mapping OpenAPI to the CLI"](https://dev.to/danielgtaylor/mapping-openapi-to-the-cli-37pb) writeup is free design input for E3.
- **Not substitutes** (surface on the same searches, solve a different problem): Hey API, Microsoft Kiota, and OpenAPI Generator emit SDKs/clients/server stubs, not a releasable end-user CLI repo; oclif is a CLI framework with no OpenAPI input.

Biscuit's position in that field: the only **open, self-hostable** generator producing a repo the user owns — no account, no token, no network call, GPL with output exception.

**Stainless promised, never shipped:**

- npm distribution ("working on support for more package managers, like npm" — CLI launch post). Biscuit ships it (as does Fern).
- Rust and Swift SDK targets (per third-party analysis); C# still beta at wind-down.

**Stainless documented limitations, now frozen forever (= biscuit's differentiation list):**

- Deep-nesting arguments requiring JSON/YAML fallback (→ depth-policy deviation, above).
- REST/OpenAPI only: no WebSockets, SSE, or gRPC; generation gaps on advanced OpenAPI 3.1 JSON Schema (→ biscuit: libopenapi gives full 3.1; SSE in scope per Protocol scope).
- Hosted-only, dashboard-centric: no self-hosted option, config not diffable/version-controlled (→ biscuit is local-first with config in the repo).
- Custom code injection limited to specific files (→ `internal/custom/` + marker-header contract is more generous).

**Migration as a product:** biscuit's parity-matched output isn't just validation methodology — it's a migration tool. Pitch line: _"Point biscuit at your existing Stainless-generated CLI repo and spec, and keep shipping releases."_ Candidate command: `biscuit adopt --repo ./foo-cli --spec openapi.yaml` (run bench, propose config that maximizes parity, take over the release pipeline).

---

## Tech stack

- **Go** (matching Stainless's rationale: native binaries, no runtime, instant startup, cross-compilation)
- [`pb33f/libopenapi`](https://github.com/pb33f/libopenapi) (spec — see Open questions), [`daveshanley/vacuum`](https://github.com/daveshanley/vacuum) (spec lint/doctor), [`spf13/cobra`](https://github.com/spf13/cobra) (commands), [`tidwall/gjson`](https://github.com/tidwall/gjson) (transforms), [`charmbracelet/bubbletea|lipgloss|glamour`](https://github.com/charmbracelet/bubbletea) (chat TUI + explore), [`modelcontextprotocol/go-sdk`](https://github.com/modelcontextprotocol/go-sdk) (MCP), [`zalando/go-keyring`](https://github.com/zalando/go-keyring) (auth), [GoReleaser](https://github.com/goreleaser/goreleaser) + [release-please](https://github.com/googleapis/release-please) (release), `text/template` + `embed.FS` (codegen)

---

## Reference codebases

|Project|Lesson|
|---|---|
|**[sqlc](https://github.com/sqlc-dev/sqlc)**|Config-driven codegen hygiene; endtoend golden testdata; compile-the-output CI. Closest constraint match — check it first on every structure decision.|
|**[ogen](https://github.com/ogen-go/ogen)**|OpenAPI→IR rigor; Optional/Nullable semantics; oneOf discriminator inference; `x-` extension overrides. Also the compatibility benchmark: "if ogen parses it, biscuit must." Its `_testdata` specs are free test cases.|
|**[kubebuilder](https://github.com/kubernetes-sigs/kubebuilder)**|Regeneration-safe scaffolding (file markers, machine- vs human-owned regions); plugin/versioning for migrating existing projects.|
|**[gh (cli/cli)](https://github.com/cli/cli)**|What output should feel like: factory pattern, iostreams, `pkg/cmd/<resource>/<verb>`.|
|**[goreleaser](https://github.com/goreleaser/goreleaser)**|Pipe-per-stage pipeline architecture; YAML config ergonomics.|
|**[buf](https://github.com/bufbuild/buf)**|Product analogue in protobuf-land; breaking-change engine ≈ the bench harness.|
|**[openai-cli](https://github.com/openai/openai-cli)**|Living spec of Stainless output — bench target _and_ template reference.|
|**[restish](https://github.com/danielgtaylor/restish)**|OSS prior art on OpenAPI→CLI mapping (runtime model); its author's mapping writeup feeds the E3 heuristics.|
|**[speakeasy-api/openapi](https://github.com/speakeasy-api/openapi)**|Genuinely OSS Go library: OpenAPI 3.0/3.1/3.2 parsing, 60+-rule linter, Overlays, Arazzo. Parser alternative to libopenapi; Overlays = a third answer to the config-override question. Speakeasy's CLI docs also document borrowable patterns (auth precedence, stutter removal, `--all` pagination, TTY binary guard).|
|**[vacuum](https://github.com/daveshanley/vacuum)**|Go-native OpenAPI linter on libopenapi (shared parse tree); Spectral-compatible rulesets; report scoring. The engine under `biscuit doctor` — study its custom-ruleset API before writing the biscuit ruleset.|

---

## License

**GPLv2-or-later** (VLC's posture) with an **explicit generated-output exception**: output produced by biscuit, including code derived from biscuit's templates, is not covered by the GPL and belongs entirely to the user under any license they choose (GCC Runtime Library Exception / Bison precedent — without this clause every generated repo is arguably a GPL derivative and adoption dies). SPDX already codifies the shape as `GPL-2.0-with-bison-exception` — adapt that text rather than drafting from scratch, and keep Bison's load-bearing carve-out: the exception must not extend to building a competing generator from biscuit's templates, or it gives away the core. "Or later" is load-bearing, not stylistic: Apache 2.0 dependencies (libopenapi et al.) are incompatible with GPLv2-only but compatible via GPLv3. Register the "biscuit" trademark separately — it's license-independent leverage. Known trades, accepted: GPL doesn't cover SaaS-wrapping (AGPL territory) and some corporate legal teams avoid GPL tooling. Effectively irreversible once outside contributions arrive; the output-exception wording is worth an hour with an OSS solicitor before v0.1 (tracked as a v0.1 release gate on the roadmap).

---

## Open questions

Live questions, plus settled ones with their answers — this is the project's decision log.

- **Resolved (2026-07):** dot-notation depth policy — schema-size-adaptive over a fixed cap: per-operation iterative deepening to the deepest expansion fitting a 64-flag budget, hard depth bound 8, cut subtrees fall back to inline JSON. Evidence: openai's `chat completions create` derives 46 static flags (past Stainless's two-level cap) while stripe's form bodies cap early; full parse→tree on stripe runs 378 ms with no budget blowup. Budget/bound stay constants until the bench shows real specs need tuning.
- Pagination mode: transparent walking vs explicit `--all`/`--max-pages` (Speakeasy's safer default) — decide before E5.
- **Resolved (2026-07):** parser+doctor pair — `pb33f/libopenapi` + vacuum, over `speakeasy-api/openapi` + its linter. E2 spike (kept in `spike/` for the epic write-up): on openai.yaml (2.8 MB) libopenapi parses+resolves in 99 ms / 41 MB heap vs speakeasy's 730 ms / 160 MB; both are cycle-safe on pathological `$ref`s. Speakeasy's built-in validation is stronger (caught duplicate operationIds and eight real 3.1 type bugs in openai.yaml) and `Upgrade()` normalizes 3.0→3.1 for free — but vacuum covers the validation layer with Spectral-compatible rulesets plus the report scoring that `lint.min_grade` assumes, and Speakeasy is a direct competitor: building biscuit's foundation on their library is a strategic dependency risk. Mitigations owed by `internal/spec`: always set `BasePath`, route libopenapi's slog noise into our diagnostics, treat `$ref` failures inside vendor extensions as advisory (libopenapi chases them; they're opaque per spec), and count webhook operations explicitly (path ops and webhooks are separate collections).
- **Resolved (2026-07):** config overrides — `biscuit.yaml` sidecar canonical, plus a small in-spec `x-biscuit-*` set (name, group, ignore, pagination hints) feeding the same override struct, sidecar winning on conflict; no native Overlay support (pre-apply externally). Evidence: Fern and Speakeasy both converged on extensions-as-vocabulary with an overrides/overlay file as the non-owner carrier ([Fern overrides](https://buildwithfern.com/learn/api-definitions/openapi/overrides), [Speakeasy extensions](https://www.speakeasy.com/docs/speakeasy-reference/extensions) + [overlays](https://www.speakeasy.com/docs/overlays)); Stainless is sidecar-only. Extension reading is known territory since E2's vendor-extension `$ref` handling. An `extensions: true` toggle was considered and rejected — the `x-biscuit-*` namespace is opt-in by construction.
- Doctor default advisory set: which vacuum rules map to generation impact vs noise (tune on the test-ladder specs) — owned by E13's tuning story; the working rule is that anything unphraseable as generation impact leaves the ruleset.
- **Resolved (2026-07):** chat TUI substrate — minimal Go/Bubble Tea REPL built in, rich UX delegated to external MCP clients (Claude Code, Warp, pi, Cursor) over `mcp serve`; a pi TypeScript port was considered and rejected. MCP is the integration boundary: rich clients already exist, improve without biscuit's involvement, and reach every generated CLI through the protocol — porting pi would mean competing with pi at chat UX in every generated repo, breaking the single-static-binary spine for the built-in surface. A rich owned UI, if ever wanted, is a TS companion npm package (E10 rails), optional. E8's spike validates the load-bearing assumption instead: real MCP clients must drive a generated `mcp serve` well (tool discovery, streaming, auth) — if they can't, the port question reopens with evidence.
- **Resolved (2026-07):** linux/386 stays in the goreleaser targets — biscuit's and generated CLIs' both. Evidence: gh, goreleaser, and openai-cli (the parity target) all ship it today; only sqlc dropped it, and deno/bun never had it ([Debian 13 ships no i386 kernel](https://www.debian.org/releases/trixie/release-notes/issues.html); [deno's 32-bit request sits unaddressed](https://github.com/denoland/deno/issues/22456)). The debate: 32-bit x86 distros are effectively dead (Debian 13 ships no i386 kernel), but the target is pure-Go (no cgo risk), costs seconds under the release build cache, and dropping it would put generated output structurally behind the Stainless output it's benchmarked against. Revisit trigger: openai-cli dropping the target.
- **Resolved (2026-07):** registry names for biscuit itself — ship as `biscuit-cli` on npm and the Homebrew tap, with `bin`/cask `binaries` keeping the command `biscuit`. Evidence: bare `biscuit` is npm-squatted and a browser cask in homebrew/cask; `v0.1.0-alpha.3` published under these names. Dispute/core-submission revisited in E12.
