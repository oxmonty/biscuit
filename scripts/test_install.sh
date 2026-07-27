#!/bin/sh
# Offline tests for install.sh: --binary install path, checksum verification,
# and the "bad --version" error path. Run: sh scripts/test_install.sh
set -eu

SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
ROOT_DIR=$(CDPATH= cd -- "$SCRIPT_DIR/.." && pwd)
INSTALL_SH="$ROOT_DIR/install.sh"

pass=0
fail=0

ok() { printf '  ok   - %s\n' "$1"; pass=$((pass + 1)); }
bad() { printf '  FAIL - %s\n' "$1"; fail=$((fail + 1)); }

# mirrors install.sh's own os/arch mapping so fixtures match the filename it computes
detect_os() {
    case "$(uname -s)" in
        Darwin*) echo darwin ;;
        Linux*) echo linux ;;
        MINGW*|MSYS*|CYGWIN*) echo windows ;;
        *) echo unsupported ;;
    esac
}

detect_arch() {
    case "$(uname -m)" in
        x86_64|amd64) echo amd64 ;;
        aarch64|arm64) echo arm64 ;;
        i386|i686) echo 386 ;;
        *) echo unsupported ;;
    esac
}

test_binary_install() {
    echo "test: --binary installs a local file, executable, PATH snippet runs"
    tmp_home=$(mktemp -d)
    dummy_bin="$tmp_home/dummy-biscuit"
    printf '#!/bin/sh\necho dummy biscuit\n' > "$dummy_bin"
    chmod +x "$dummy_bin"
    : > "$tmp_home/.bashrc"

    status=0
    HOME="$tmp_home" SHELL=/bin/bash bash "$INSTALL_SH" --binary "$dummy_bin" \
        > "$tmp_home/out.log" 2>&1 || status=$?

    installed="$tmp_home/.biscuit/bin/biscuit"
    if [ "$status" -eq 0 ] && [ -x "$installed" ]; then
        ok "binary installed and executable at $installed"
    else
        bad "binary not installed correctly (status=$status)"
        cat "$tmp_home/out.log"
    fi

    if grep -qF "$tmp_home/.biscuit/bin" "$tmp_home/.bashrc" 2>/dev/null; then
        ok "PATH snippet appended to .bashrc without error"
    else
        bad "PATH snippet was not appended to .bashrc"
        cat "$tmp_home/out.log"
    fi

    rm -rf "$tmp_home"
}

# builds a fake `curl` on PATH that serves local fixture files for the
# archive and checksums.txt downloads, and 404s everything else
make_fake_curl() {
    fakebin_dir="$1"
    archive_src="$2"
    checksums_src="$3"
    filename="$4"

    mkdir -p "$fakebin_dir"
    cat > "$fakebin_dir/curl" <<EOF
#!/bin/sh
out=""
prev=""
url=""
for a in "\$@"; do
    if [ "\$prev" = "-o" ]; then out="\$a"; fi
    prev="\$a"
    url="\$a"
done
case "\$url" in
    */${filename})
        if [ -n "${archive_src}" ] && [ -f "${archive_src}" ]; then cp "${archive_src}" "\$out"; printf '200'; else printf '404'; fi
        ;;
    */checksums.txt)
        if [ -n "${checksums_src}" ] && [ -f "${checksums_src}" ]; then cp "${checksums_src}" "\$out"; printf '200'; else printf '404'; fi
        ;;
    *) printf '404' ;;
esac
EOF
    chmod +x "$fakebin_dir/curl"
}

test_checksum_mismatch() {
    echo "test: tampered archive is rejected by checksum verification"
    tmp_home=$(mktemp -d)
    work="$tmp_home/work"
    mkdir -p "$work/root"

    os_name=$(detect_os)
    arch_name=$(detect_arch)
    version="9.9.9-test"
    filename="biscuit_${version}_${os_name}_${arch_name}.tar.gz"

    printf '#!/bin/sh\necho hi\n' > "$work/root/biscuit"
    chmod +x "$work/root/biscuit"
    (cd "$work/root" && tar -czf "$work/$filename" biscuit)

    # checksums.txt intentionally references a hash that does not match the archive above
    bogus_hash=$(printf 'not-the-real-content' | (sha256sum 2>/dev/null || shasum -a 256) | awk '{print $1}')
    printf '%s  %s\n' "$bogus_hash" "$filename" > "$work/checksums.txt"

    fakebin="$tmp_home/fakebin"
    make_fake_curl "$fakebin" "$work/$filename" "$work/checksums.txt" "$filename"

    status=0
    PATH="$fakebin:$PATH" HOME="$tmp_home" bash "$INSTALL_SH" --version "$version" --no-modify-path \
        > "$tmp_home/out.log" 2>&1 || status=$?

    if [ "$status" -ne 0 ] && grep -qi "checksum" "$tmp_home/out.log"; then
        ok "checksum mismatch rejected with a clear error (status=$status)"
    else
        bad "checksum mismatch was not detected (status=$status)"
        cat "$tmp_home/out.log"
    fi

    rm -rf "$tmp_home"
}

test_bad_version() {
    echo "test: a nonexistent --version fails with a clear message, not a curl/extract error"
    tmp_home=$(mktemp -d)
    fakebin="$tmp_home/fakebin"
    make_fake_curl "$fakebin" "" "" "biscuit_missing"

    status=0
    PATH="$fakebin:$PATH" HOME="$tmp_home" bash "$INSTALL_SH" --version "0.0.0-missing" --no-modify-path \
        > "$tmp_home/out.log" 2>&1 || status=$?

    if [ "$status" -ne 0 ] && grep -qi "not found" "$tmp_home/out.log"; then
        ok "bad version rejected with a clear 'not found' message"
    else
        bad "bad version did not produce the expected error (status=$status)"
        cat "$tmp_home/out.log"
    fi

    rm -rf "$tmp_home"
}

test_binary_install
test_checksum_mismatch
test_bad_version

echo ""
echo "passed: $pass, failed: $fail"
[ "$fail" -eq 0 ]
