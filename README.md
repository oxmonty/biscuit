# biscuit

Convert an OpenAPI 3.x spec into a complete, production-ready CLI repository. An open, self-hostable alternative to the Stainless CLI generator.

```sh
biscuit generate --spec openapi.yaml --config biscuit.yaml --out ./foo-cli
```

> **Status: pre-alpha walking skeleton.** The install paths below work; generation does not exist yet. See [ROADMAP.md](ROADMAP.md).

## Install

```sh
brew tap oxmonty/tap && brew install biscuit-cli   # Homebrew
npx biscuit-cli version                            # npm
```

Or grab a binary from [GitHub Releases](https://github.com/oxmonty/biscuit/releases).

## Usage

```sh
biscuit version
biscuit --help
```

## License

GPL-2.0-or-later, with a planned generated-output exception (generated code belongs entirely to you). Final license text pending.
