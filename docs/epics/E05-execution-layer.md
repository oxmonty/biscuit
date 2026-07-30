# E5: Execution layer

Generated CLIs make correct, ergonomic API calls, proven by golden requests against a spec-generated mock in a released biscuit. → [Output control](../../PRD.md#output-control), [API semantics](../../PRD.md#api-semantics-handled-automatically), [Protocol scope](../../PRD.md#protocol-scope), [Additional design considerations](../../PRD.md#additional-design-considerations)

## Stories

Build order: 6 → 2 → 3 → 4 → 5 → 1 (the mock proves everything, so it lands last; the client core unblocks the rest).

- [ ] Generate a mock server from any spec (routes + schema-valid canned responses + request recording), rendered into generated repos as smoke-test code; biscuit's e2e swaps E4's echo server for the repo's own smoke machinery, and E6's ladder and parity bench reuse the same generator.
- [ ] Map `securitySchemes` to auth flags and env vars (`{BINARY}_{SCHEME}`, multiple keys supported); oauth2/openIdConnect run as pre-obtained bearer tokens from env with a doctor advisory.
- [ ] Ship `--format auto|json|jsonl|pretty|raw|yaml` and `--transform`/`--transform-error`/`--format-error` via gjson, plus `--output/-o` with non-clobbering defaults, `--include-headers`, and the binary-to-TTY guard (`explore` waits for E8).
- [ ] Implement `@file` argument handling with sniffing (text → string, binary → base64), explicit `@file://`/`@data://` prefixes, `\@` escape, and minimal multipart per-part mapping.
- [ ] Implement pagination — transparent walking bounded by `--max-pages`, detected via declared `pagination:` schemes plus a built-in convention library (survey-seeded, word-boundary vocabulary, both-sides corroboration) — and stream SSE responses as JSONL when piped (plain line streaming on TTY until E8).
- [ ] Add retries/backoff with `Retry-After`, the documented exit-code contract, `--debug` with secret redaction (`--debug-unsafe`), `--header` injection, and `servers[].variables` in the `--base-url` default.

## Acceptance criteria

- A generated CLI authenticates via securityScheme-derived flags and env vars, with multiple keys supported.
- `--format` renders json/jsonl/pretty/raw/yaml and `--transform` applies GJSON expressions to output (`--transform-error` to error bodies).
- `@file.ext` arguments sniff text vs binary (string vs base64), honoring `@file://`/`@data://` prefixes and the `\@` escape.
- Paginated endpoints (declared or built-in scheme match) walk all pages transparently; `--max-pages N` bounds the walk; unmatched operations never paginate.
- The client retries 429/5xx with jittered backoff honoring `Retry-After`; `--no-retries` disables; the mock proves retries fire and stop.
- Generated CLIs exit with the documented code contract (0 success, distinct codes for usage, auth, 4xx, 5xx, network), stated in the generated README.
- `--debug` prints full request/response with auth headers and secret-shaped fields redacted, and a generated repo's own smoke suite (spec-derived mock + golden requests) passes in its CI.

## Exit criteria

- Artifact: a released biscuit (next release-please cut) whose generated CLIs execute with auth, retries, output control, pagination, and SSE — and whose generated repos self-test against their own spec-derived mock.
- Regression: `go test ./...` — proves the acceptance criteria; its green run gates the tick (compile-the-output extends to running generated smoke suites).
- Demo: generate galaxy-cli, then against its own mock: env-var auth, a paginated `list` walking three mock pages, an `@file` upload, an SSE stream piped as JSONL, and a 429-retry visible in `--debug` with the auth header redacted.
- Benchmarks: `go test ./internal/render/ -bench BenchmarkRender -run XXX` → openai render within 5 s (E4 baseline 1.23 s).
