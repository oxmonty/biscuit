# biscuit

A Go package and CLI that converts an OpenAPI 3.x spec into a complete, production-ready CLI repository (`{project}-cli`). Self-hostable alternative to the wound-down Stainless CLI generator.

`ROADMAP.md` is the design doc and source of truth — epics E1–E11, architecture, mapping heuristics, bench methodology, and open questions all live there. Read the relevant section before building anything.

## Commands

```sh
go build ./...              # build everything
go test ./...               # run tests
go run ./cmd/biscuit        # run the CLI locally
```

## Layout

- `cmd/biscuit/` — thin main; wires cobra commands
- `internal/cli/` — cobra command definitions
- `internal/version/` — version metadata, set by goreleaser ldflags
- `npm/biscuit-cli/` — npm shim package (`npx biscuit-cli`); platform packages are built at release time by `scripts/publish_npm.mjs`
- Planned (see ROADMAP.md project structure): `biscuit.go` public library API, `internal/{spec,lint,ir,mapping,render,bench}`, `templates/`, `testdata/`

## Principles (from ROADMAP.md)

- `biscuit.Generate(ctx, spec, cfg) → FilePlan` is a pure function; writing files is a separate `plan.Write(dir)` step.
- Generation is deterministic: sort every IR slice at mapping time; each output file's bytes depend only on the IR.
- The CLI is the first consumer of the public library API; everything else stays `internal/`.
- CI compiles generated output (`go build ./...` on golden repos) — the single most valuable check.

## Releases

Conventional Commits → release-please accumulates a release PR → merging it tags `vX.Y.Z` → goreleaser builds cross-platform binaries, publishes GitHub Release + Homebrew tap cask → `scripts/publish_npm.mjs` publishes `@oxmonty/biscuit-<platform>-<arch>` packages then the `biscuit-cli` shim. Never tag or publish manually.

One-time setup (not yet done — required before the first release works):

1. Create the `oxmonty/homebrew-tap` repo (public, empty is fine).
2. Add a `HOMEBREW_TAP_TOKEN` repo secret: fine-grained PAT (or GitHub App token) with contents write on the tap repo.
3. The `oxmonty` npm org exists (shared home for biscuit's platform packages, published skills, and future packages). The unscoped `biscuit-cli` shim is claimed by its first publish.
4. Add an `NPM_TOKEN` repo secret (granular automation token) for the bootstrap release, then configure npm trusted publishing (OIDC) for `biscuit-cli` and each `@oxmonty/biscuit-*` platform package against this repo's `release.yml` workflow and delete the token.
