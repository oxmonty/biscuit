# biscuit

[![release](https://img.shields.io/github/v/release/oxmonty/biscuit?include_prereleases&sort=semver)](https://github.com/oxmonty/biscuit/releases)
[![downloads](https://img.shields.io/github/downloads/oxmonty/biscuit/total?label=binary%20downloads)](https://github.com/oxmonty/biscuit/releases)
[![npm](https://img.shields.io/npm/dm/biscuit-cli?label=npm%20downloads)](https://www.npmjs.com/package/biscuit-cli)
[![ci](https://github.com/oxmonty/biscuit/actions/workflows/ci.yml/badge.svg)](https://github.com/oxmonty/biscuit/actions/workflows/ci.yml)

Convert an OpenAPI 3.x spec into a complete, production-ready CLI repository. An open, self-hostable alternative to the Stainless CLI generator.

```sh
biscuit generate --spec openapi.yaml --config biscuit.yaml --out ./foo-cli
```

> **Status: pre-alpha walking skeleton.** The install paths below work; generation does not exist yet. See [ROADMAP.md](ROADMAP.md).

## Install

```sh
# stable channel
brew install oxmonty/tap/biscuit-cli
npm install -g biscuit-cli             # or one-off: npx biscuit-cli

# prerelease channel (recommended until v0.1 ships)
brew install oxmonty/tap/biscuit-cli@next
npm install -g biscuit-cli@next
```

Either way the installed command is `biscuit`.

The fully-qualified brew name matters: since Homebrew 6, it trusts just this cask; a bare `brew install biscuit-cli` after tapping requires `brew trust oxmonty/tap` first.

No package manager? Install directly:

```sh
curl -fsSL https://raw.githubusercontent.com/oxmonty/biscuit/main/install.sh | bash
```

Or grab a binary from [GitHub Releases](https://github.com/oxmonty/biscuit/releases).

## Usage

```sh
biscuit version
biscuit --help
```

## License

[GPL-2.0-or-later](LICENSE). A generated-output exception is planned (output produced by biscuit, including code derived from its templates, will belong entirely to you under any license you choose); final wording pending legal review.
