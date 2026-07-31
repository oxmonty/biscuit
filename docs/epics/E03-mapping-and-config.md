# E3: Mapping and config

a released `biscuit generate --dry-run` prints the derived command surface for any spec, overridable via `biscuit.yaml`. `v0.1.0-alpha.5` → [Command grammar](../../PRD.md#command-grammar), [Argument parsing](../../PRD.md#argument-parsing)

Write-up: [docs/write-ups/E3-mapping-and-config.md](../write-ups/E3-mapping-and-config.md)

## Stories

- [x] Derive the resource/verb tree from tags and paths, including nested sub-resources and stutter removal.
- [x] Add [stripe/openapi](https://github.com/stripe/openapi) to `testdata/specs` as the tree-derivation stress test: a large real-world 3.x spec with deeply nested resources and polymorphic `oneOf` on nearly every object, a distinct shape from openai.yaml.
- [x] Implement flag flattening with the schema-adaptive dot-notation depth policy, cycle detection, and a hard depth bound.
- [x] Implement the oneOf discriminator-inference cascade.
- [x] Load and apply `biscuit.yaml` overrides (names, aliases, hidden endpoints, pagination hints), validated against a schema: unknown keys rejected with precise errors, `version` key for forward migration — plus the in-spec `x-biscuit-*` mirror set (name, group, ignore, pagination hints) feeding the same override struct, sidecar winning on conflict.
- [x] Ship `biscuit init`: scaffold a starter `biscuit.yaml` seeded from `doctor`'s gap analysis.
- [x] Ship `biscuit generate --dry-run` printing the derived resource/verb tree and the FilePlan — free from the plan/write split, and E3's demo.
- [x] Polish doctor output: humane one-line resolver diagnostics (no raw rolodex dumps), finding counts folded into the impact phrasing ("718 sites weaken the mock corpus"), severity colors on TTY, and `doctor --format json` for CI pipelines.
