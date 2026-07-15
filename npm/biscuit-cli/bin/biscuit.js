#!/usr/bin/env node
"use strict";
const { spawnSync } = require("node:child_process");
const path = require("node:path");

function binaryPath() {
  const pkg = `@oxmonty/biscuit-${process.platform}-${process.arch}`;
  const bin = process.platform === "win32" ? "biscuit.exe" : "biscuit";
  try {
    return path.join(path.dirname(require.resolve(`${pkg}/package.json`)), "bin", bin);
  } catch {
    console.error(
      `biscuit-cli: no binary for ${process.platform}-${process.arch}.\n` +
        `Expected optional dependency ${pkg} to be installed. pnpm and Yarn PnP\n` +
        `users may need to enable optional dependencies for this platform.`
    );
    process.exit(1);
  }
}

const result = spawnSync(binaryPath(), process.argv.slice(2), { stdio: "inherit" });
if (result.error) {
  console.error(`biscuit-cli: ${result.error.message}`);
  process.exit(1);
}
process.exit(result.status ?? 1);
