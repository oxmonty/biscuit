#!/usr/bin/env bash
# biscuit installer — a third channel alongside Homebrew and npm, for anyone
# who doesn't want either. Structure, PATH-detection, and shell-config logic
# adapted from opencode's install script (MIT): https://github.com/sst/opencode
set -euo pipefail

REPO="oxmonty/biscuit"
INSTALL_DIR="$HOME/.biscuit/bin"

MUTED='\033[0;2m'
RED='\033[0;31m'
ORANGE='\033[38;5;214m'
NC='\033[0m'

usage() {
    cat <<EOF
biscuit installer

Usage: install.sh [options]

Options:
    -h, --help                 Display this help message
    -v, --version <version>    Install a specific version (e.g. 0.1.0-alpha.4)
    -c, --channel <channel>    stable or next (default: next — no stable release yet)
        --no-modify-path       Don't modify shell config files (.zshrc, .bashrc, etc.)

Examples:
    curl -fsSL https://raw.githubusercontent.com/oxmonty/biscuit/main/install.sh | bash
    curl -fsSL https://raw.githubusercontent.com/oxmonty/biscuit/main/install.sh | bash -s -- --version 0.1.0-alpha.4
EOF
}

requested_version=""
channel="next"
no_modify_path=false

while [[ $# -gt 0 ]]; do
    case "$1" in
        -h|--help) usage; exit 0 ;;
        -v|--version)
            [[ -n "${2:-}" ]] || { echo -e "${RED}Error: --version requires an argument${NC}"; exit 1; }
            requested_version="$2"; shift 2 ;;
        -c|--channel)
            [[ -n "${2:-}" ]] || { echo -e "${RED}Error: --channel requires an argument${NC}"; exit 1; }
            channel="$2"; shift 2 ;;
        --no-modify-path) no_modify_path=true; shift ;;
        *) echo -e "${ORANGE}Warning: unknown option '$1'${NC}" >&2; shift ;;
    esac
done

mkdir -p "$INSTALL_DIR"

raw_os=$(uname -s)
case "$raw_os" in
    Darwin*) os="darwin" ;;
    Linux*)  os="linux" ;;
    MINGW*|MSYS*|CYGWIN*) os="windows" ;;
    *) echo -e "${RED}Error: unsupported OS '$raw_os'${NC}"; exit 1 ;;
esac

raw_arch=$(uname -m)
case "$raw_arch" in
    x86_64|amd64) arch="amd64" ;;
    aarch64|arm64) arch="arm64" ;;
    i386|i686) arch="386" ;;
    *) echo -e "${RED}Error: unsupported architecture '$raw_arch'${NC}"; exit 1 ;;
esac

archive_ext="tar.gz"
[[ "$os" == "windows" ]] && archive_ext="zip"

if [[ -z "$requested_version" ]]; then
    # /releases/latest 404s until the first stable release exists; /releases
    # is newest-first including prereleases, which is what --channel next wants.
    if [[ "$channel" == "stable" ]]; then
        body=$(curl -fsSL "https://api.github.com/repos/$REPO/releases/latest" || true)
        requested_version=$(echo "$body" | grep '"tag_name"' | head -1 | sed -E 's/.*"v?([^"]+)".*/\1/')
        if [[ -z "$requested_version" ]]; then
            echo -e "${RED}Error: no stable release published yet. Use --channel next.${NC}"
            exit 1
        fi
    else
        body=$(curl -fsSL "https://api.github.com/repos/$REPO/releases")
        requested_version=$(echo "$body" | grep '"tag_name"' | head -1 | sed -E 's/.*"v?([^"]+)".*/\1/')
    fi
fi

filename="biscuit_${requested_version}_${os}_${arch}.${archive_ext}"
url="https://github.com/$REPO/releases/download/v${requested_version}/${filename}"

echo -e "${MUTED}Downloading${NC} biscuit ${requested_version} ${MUTED}(${os}/${arch})${NC}"
tmpdir=$(mktemp -d)
trap 'rm -rf "$tmpdir"' EXIT
curl -fsSL "$url" -o "$tmpdir/$filename" || { echo -e "${RED}Error: download failed — $url${NC}"; exit 1; }

if [[ "$archive_ext" == "zip" ]]; then
    command -v unzip >/dev/null || { echo -e "${RED}Error: 'unzip' is required${NC}"; exit 1; }
    unzip -oq "$tmpdir/$filename" -d "$tmpdir"
else
    tar -xzf "$tmpdir/$filename" -C "$tmpdir"
fi

binary="biscuit"
[[ "$os" == "windows" ]] && binary="biscuit.exe"
mv "$tmpdir/$binary" "$INSTALL_DIR/$binary"
chmod +x "$INSTALL_DIR/$binary"

if [[ "$no_modify_path" != "true" ]] && [[ ":$PATH:" != *":$INSTALL_DIR:"* ]]; then
    current_shell=$(basename "${SHELL:-sh}")
    case "$current_shell" in
        fish) config_files="$HOME/.config/fish/config.fish" ;;
        zsh)  config_files="$HOME/.zshrc" ;;
        bash) config_files="$HOME/.bashrc $HOME/.bash_profile" ;;
        *)    config_files="$HOME/.profile" ;;
    esac

    config_file=""
    for f in $config_files; do [[ -f "$f" ]] && { config_file="$f"; break; }; done

    if [[ -n "$config_file" ]]; then
        line="export PATH=\"$INSTALL_DIR:\$PATH\""
        [[ "$current_shell" == "fish" ]] && line="fish_add_path $INSTALL_DIR"
        if ! grep -qF "$INSTALL_DIR" "$config_file" 2>/dev/null; then
            { echo ""; echo "# biscuit"; echo "$line"; } >> "$config_file"
        fi
        echo -e "${MUTED}Successfully added ${NC}biscuit ${MUTED}to \$PATH in ${NC}$config_file"
    else
        echo -e "${ORANGE}No shell config found — add this manually:${NC}"
        echo -e "  export PATH=\"$INSTALL_DIR:\$PATH\""
    fi
fi

echo -e ""
echo -e "${ORANGE} _     _                _ _   ${NC}"
echo -e "${ORANGE}| |   (_)              (_) |  ${NC}"
echo -e "${ORANGE}| |__  _ ___  ___ _   _ _| |_ ${NC}"
echo -e "${ORANGE}| '_ \\\\| / __|/ __| | | | | __|${NC}"
echo -e "${ORANGE}| |_) | \\\\__ \\\\ (__| |_| | | |_ ${NC}"
echo -e "${ORANGE}|_.__/|_|___/\\\\___|\\\\__,_|_|\\\\__|${NC}"
echo -e ""
echo -e "${MUTED}Generate a production-ready CLI repository from an OpenAPI 3.x spec${NC}"
echo -e ""
printf "%-20s ${MUTED}# a dir with an OpenAPI spec${NC}\n" "cd <project>"
printf "%-20s ${MUTED}# grade the spec${NC}\n" "biscuit doctor"
echo -e ""
echo -e "${MUTED}For more information visit ${NC}https://github.com/$REPO"
echo -e ""
