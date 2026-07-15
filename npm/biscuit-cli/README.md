# biscuit-cli

Convert an OpenAPI 3.x spec into a complete, production-ready CLI repository. An open, self-hostable alternative to the Stainless CLI generator.

```sh
npm install -g biscuit-cli   # or one-off: npx biscuit-cli
```

**The installed command is `biscuit`:**

```sh
biscuit --help
biscuit generate --spec openapi.yaml --config biscuit.yaml --out ./foo-cli
```

This package is a thin launcher; the platform binary comes from an `@oxmonty/biscuit-*` optional dependency. Also available via Homebrew: `brew install oxmonty/tap/biscuit-cli`.

Docs, source, and roadmap: [github.com/oxmonty/biscuit](https://github.com/oxmonty/biscuit)
