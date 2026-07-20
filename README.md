# biscuit

<p align="left">
  <a href="https://github.com/oxmonty/biscuit/actions/workflows/ci.yml?query=branch%3Amain"><img src="https://img.shields.io/github/actions/workflow/status/oxmonty/biscuit/ci.yml?branch=main&style=for-the-badge" alt="CI status"></a>
  <a href="https://github.com/oxmonty/biscuit/releases"><img src="https://img.shields.io/github/v/release/oxmonty/biscuit?include_prereleases&style=for-the-badge" alt="GitHub release"></a>
  <a href="https://somsubhra.github.io/github-release-stats/?username=oxmonty&repository=biscuit"><img src="https://img.shields.io/github/downloads/oxmonty/biscuit/v0.1.0-alpha.5/total?label=downloads&style=for-the-badge" alt="Downloads of the v0.1.0-alpha.5 release"></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/License-GPL--2.0-blue.svg?style=for-the-badge" alt="GPL-2.0 License"></a>
  <!-- Uncomment when the Discord server is up (fill in invite code and server id):
  <a href="https://discord.gg/INVITE_CODE"><img src="https://img.shields.io/discord/SERVER_ID?label=Discord&logo=discord&logoColor=white&color=5865F2&style=for-the-badge" alt="Discord"></a>
  -->
</p>

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
