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

Publishing setup (done 2026-07): Homebrew publishes to `oxmonty/homebrew-tap` via the org-level `HOMEBREW_TAP_TOKEN` secret (fine-grained PAT, contents-write on the tap only). npm publishes via trusted publishing (OIDC) — each package's trusted publisher is this repo's `release.yaml` workflow; there is no npm token anywhere. A brand-new npm package can't use OIDC on its first publish: publish it once locally (`npm login` + `scripts/publish_npm.mjs`), then add the trusted publisher.
