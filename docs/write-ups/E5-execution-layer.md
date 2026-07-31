# E5: Execution layer — write-up

2026-07-31

## What shipped

Generated CLIs stopped being request printers and became production API clients. For the end user of any generated CLI: credentials resolve from flags or env vars with actionable errors before a single network call (exit 3 names the exact flag and env var to set); transient failures retry with jittered backoff honoring `Retry-After`; output bends to the job (`--format json|jsonl|pretty|raw|yaml`, GJSON `--transform`, `--output` with non-clobbering filenames, `--include-headers`, a binary-to-TTY guard); payloads come from files (`@file` sniffing text vs binary, real multipart uploads); paginated lists walk every page transparently, bounded by `--max-pages`; SSE endpoints stream line-per-event JSONL that composes with `--transform`; and scripts branch on a documented 0–6 exit-code contract. `--debug` shows full wire traffic with auth headers, secret-shaped body keys, and query params redacted (`--debug-unsafe` to disable).

For the developer running biscuit: pagination needs zero config on common APIs — a built-in convention library (seeded from a sourced survey of Stripe, OpenAI, Slack, AWS, Google, GitHub, and the offset/page classics) auto-matches operations conservatively, requiring corroboration from both the request and response side; custom shapes declare named schemes in `biscuit.yaml`, Stainless-style. And every generated repo now tests itself: a spec-derived recording mock plus a smoke suite that drives every command in-process land in `internal/mock/`, running in the repo's own CI — the machinery E6's parity bench and E9's update PRs build on.

## Try it yourself

```sh
cd $(mktemp -d)
curl -fsSLo openapi.yaml https://raw.githubusercontent.com/oxmonty/biscuit/main/testdata/specs/petstore.yaml
biscuit generate                     # discovers the spec, writes ./swagger-petstore-cli
cd swagger-petstore-cli
go test ./...                        # the repo tests itself against its own spec-derived mock
go run ./cmd/swagger-petstore pets list --limit 3 --base-url https://httpbin.org/anything --format pretty
go run ./cmd/swagger-petstore pets list --debug --header "X-Api-Key: secret" \
  --base-url https://httpbin.org/anything 2>&1 | grep REDACTED
go run ./cmd/swagger-petstore pets show; echo "exit: $?"   # missing required flag → exit: 2
```

The first command after the build prints a syntax-highlighted response; the second proves secret redaction on the wire log; the third proves the exit-code contract. (Until v0.1.0-alpha.8 releases, run biscuit from the branch: `go run github.com/oxmonty/biscuit/cmd/biscuit@<sha> generate`.)

## Validation notes

- `go test ./...` — the whole suite (~90 s): four e2e functions driving real generated CLIs over the wire (auth, output control, `@file`, pagination walks incl. a clamping-server guard, SSE incl. JSON fallback, retries, the full exit-code contract, redaction), matcher/config units, golden repos (`-update` to refresh; the `-update` test writes bytes without compiling, so gates also `go build` + `go test` inside both golden repos directly), FilePlan hash pins, and compile-the-output running each big spec's own smoke suite.
- `go test ./internal/render/ -bench BenchmarkRender -run XXX` — budget: openai ≤ 5 s.
- `golangci-lint run` — 0 issues, generated output lints clean.

## Evidence

- 14 commits on `feature/execution-layer` (`31e7284`…`4f8b68b`), each story reviewed and gated before commit.
- Full `go test ./... -count=1` green at every story commit and at close; 13 regression assertions added at exit audit, each observed red once before being trusted (`1b84619`).
- Bench baseline (Apple M2, parse→render→FilePlan): petstore 104 ms, train-travel 119 ms, openai 1.25 s, stripe 2.53 s — the E5 numbers later epics regress against (E4 baseline: 73/94 ms, 1.23/2.45 s; the whole execution layer plus two new rendered files cost openai 20 ms).
- Smoke coverage: 993 generated invocations across the spec ladder (petstore 3, galaxy 10, museum 8, train-travel 7, pokeapi 98, openai 281, stripe 587), zero skips after the mapping fixes.
- Built-in pagination matches with zero config: openai 51, stripe 120, pokeapi 48, train-travel 3, galaxy 1; petstore 0 (negative control), museum 0 (spec documents no stop signal — correctly unmatched).
- Code review (multi-agent, eight angles + verification): 10 verified findings, 8 fixed in `4f8b68b`, 2 recorded as ponytail ceilings; 4 candidate findings refuted during verification.

## Decisions made along the way

- **Pagination is transparent walking, scheme-detected** (decision log): `--max-pages` bounds it; declared `biscuit.yaml` schemes plus a built-in convention library auto-match with word-exact, both-sides corroboration; ambiguity resolves to no pagination. Explicit `--all` opt-in was rejected as the default after verifying Stainless's own CLI walks transparently.
- **The mock renders into generated repos** (decision log): biscuit-internal-only was rejected — generated repos can't import biscuit, and the harness-only shape would have built the mock generator without its hardest consumer.
- **Declared schemes outrank built-ins as a tier**: the ambiguity veto applies within a tier, or declaring a custom scheme for a Stripe-shaped endpoint would have disabled its pagination.
- **Page-number arithmetic requires page-count corroboration**: Stripe's search endpoints pair a `page` param with an opaque cursor; a bare `total` no longer implies numeric pages (stepping to `page=2` would 400).
- **`ending_before` is not a forward cursor**: dropped from the built-in's params after review — feeding last-item ids into a backward param walks backward.
- **Smoke tests drive the command tree in-process** (`NewRootCmd` + `SetArgs`), not a built binary — full CLI fidelity at unit-test cost, which is what makes 993 invocations affordable in CI.
- **The custom-command hook is E9's** (backlogged this session): custom cobra commands today mean owning root.go; the emit-once `custom.RegisterCommands` hook ships with the update pipeline, where clobber-proof custom commands start mattering.

## What surprised

- **The smoke suite found two real generator bugs within minutes of existing**: openai declares seven path keys with literal query strings (`/responses?beta=true`), which survived into command names and produced double-`?` URLs; and a body mixing `properties` with an `anyOf` required-constraint fell through to an empty-named required flag, making one command uninvocable. Both root-caused and fixed with regression tests (`5776d23`).
- **Golden `-update` writes bytes without compiling** — a template bug that only breaks compilation passes the golden test. Caught when a duplicate map key (two apiKey schemes wiring to one param) compiled nowhere; every subsequent gate builds the golden repos directly.
- **The review's "plausible" unbounded-walk finding was real**: reverting the fix to observe the test red produced a live runaway — the generated CLI pinned a CPU for 2+ minutes against a clamping mock. The identical-page guard is not theoretical.
- **pflag's optional-flag values are a footgun**: `NoOptDefVal` silently breaks the natural `-o path` form (only `-o=path` parses), so `--output` took a literal `auto` sentinel instead.
- **A finder/lead wake-up race in the multi-agent review**: the finders' completion signals landed in the parent session instead of resuming the review lead — it needed two manual nudges to finish. Worth remembering when orchestrating background agent relays.

## Left open

- The ROADMAP epic checkbox awaits the released artifact: merge → release-please → `v0.1.0-alpha.8`, then tick with the version stamp (E4 precedent).
- oauth2/openIdConnect doctor advisory needs a custom vacuum rule — E13 (doctor deepening), recorded on the story.
- Mock-route collisions for stripped-query duplicate paths and multipart leaf-name collisions on nested schemas — recorded as `ponytail:` ceilings in `internal/render/model.go`, indistinguishable-on-the-wire / rare respectively.
- Cookie-idempotency fix has no dedicated test — needs a spec combining pagination + a header flag + cookie auth, which none of the ladder specs exhibit; add if one ever does.
- SSE on an interactive TTY streams plain lines until E8's TUI takes over.
