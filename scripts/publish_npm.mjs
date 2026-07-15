// Publishes npm packages from goreleaser output: one @biscuit-cli/<platform>-<arch>
// package per built binary, then the biscuit-cli shim whose optionalDependencies
// pin them all at the same version. Run from the repo root after `goreleaser release`:
//
//   node scripts/publish_npm.mjs <version>
import { execFileSync } from "node:child_process";
import fs from "node:fs";
import path from "node:path";

const version = process.argv[2];
if (!version) {
  console.error("usage: node scripts/publish_npm.mjs <version>");
  process.exit(1);
}

// goos/goarch -> node's process.platform/process.arch
const PLATFORM = { darwin: "darwin", linux: "linux", windows: "win32" };
const ARCH = { amd64: "x64", arm64: "arm64", 386: "ia32" };

const artifacts = JSON.parse(fs.readFileSync("dist/artifacts.json", "utf8"));
const binaries = artifacts.filter((a) => a.type === "Binary");
if (binaries.length === 0) {
  console.error("no binaries in dist/artifacts.json — run goreleaser first");
  process.exit(1);
}

// npm refuses to publish prereleases without an explicit dist-tag; without
// this, `latest` would also end up pointing at an alpha
const distTag = version.includes("-") ? version.split("-")[1].split(".")[0] : "latest";

function publish(dir) {
  execFileSync("npm", ["publish", "--access", "public", "--tag", distTag], {
    cwd: dir,
    stdio: "inherit",
  });
}

const optionalDependencies = {};
for (const b of binaries) {
  const platform = PLATFORM[b.goos];
  const arch = ARCH[b.goarch];
  if (!platform || !arch) continue;

  const name = `@oxmonty/biscuit-${platform}-${arch}`;
  const dir = path.join("dist", "npm", `${platform}-${arch}`);
  const binDir = path.join(dir, "bin");
  fs.mkdirSync(binDir, { recursive: true });

  const binName = platform === "win32" ? "biscuit.exe" : "biscuit";
  fs.copyFileSync(b.path, path.join(binDir, binName));
  fs.chmodSync(path.join(binDir, binName), 0o755);

  fs.writeFileSync(
    path.join(dir, "package.json"),
    JSON.stringify(
      {
        name,
        version,
        description: `biscuit binary for ${platform}-${arch}`,
        repository: { type: "git", url: "git+https://github.com/oxmonty/biscuit.git" },
        license: "GPL-2.0-or-later",
        os: [platform],
        cpu: [arch],
        files: ["bin/"],
      },
      null,
      2
    )
  );

  publish(dir);
  optionalDependencies[name] = version;
}

const shimSrc = "npm/biscuit-cli";
const shimDir = path.join("dist", "npm", "biscuit-cli");
fs.cpSync(shimSrc, shimDir, { recursive: true });
const pkg = JSON.parse(fs.readFileSync(path.join(shimDir, "package.json"), "utf8"));
pkg.version = version;
pkg.optionalDependencies = optionalDependencies;
fs.writeFileSync(path.join(shimDir, "package.json"), JSON.stringify(pkg, null, 2));

publish(shimDir);
console.log(`published biscuit-cli@${version} + ${Object.keys(optionalDependencies).length} platform packages`);
