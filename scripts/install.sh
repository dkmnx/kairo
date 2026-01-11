#!/bin/sh
set -e

REPO="dkmnx/kairo"
BINARY_NAME="kairo"
INSTALL_DIR="${INSTALL_DIR:-$HOME/.local/bin}"

usage() {
    cat <<EOF
Install kairo CLI

Usage: $0 [OPTIONS]

OPTIONS
    -b, --bin-dir DIRECTORY    Install binary to DIRECTORY (default: $INSTALL_DIR)
    -v, --version VERSION      Install specific version (default: latest)
    -r, --repo REPO            Repository in format owner/repo (default: $REPO)
    -h, --help                 Show this help

EXAMPLES
    $0                        # Install latest version to ~/.local/bin
    $0 -b /usr/local/bin      # Install to /usr/local/bin
    $0 -v v1.0.0              # Install specific version

EOF
    exit 0
}

log() {
    echo "[kairo] $1"
}

error() {
    echo "[kairo] ERROR: $1" >&2
    exit 1
}

detect_os() {
    case "$(uname -s)" in
        Linux*)     echo "linux" ;;
        Darwin*)    echo "darwin" ;;
        CYGWIN*|MINGW*|MSYS*) echo "windows" ;;
        *)          error "Unsupported OS: $(uname -s)" ;;
    esac
}

detect_arch() {
    case "$(uname -m)" in
        x86_64)     echo "amd64" ;;
        arm64|aarch64) echo "arm64" ;;
        armv7l)     echo "arm7" ;;
        *)          error "Unsupported architecture: $(uname -m)" ;;
    esac
}

get_latest_version() {
    version=$(curl -s "https://api.github.com/repos/$REPO/releases/latest" | \
        grep '"tag_name"' | sed 's/.*": "\(.*\)".*/\1/')
    echo "$version"
}

download_and_install() {
    os="$1"
    arch="$2"
    version="$3"

    extension=""
    if [ "$os" = "windows" ]; then
        extension=".exe"
        archive_ext=".zip"
    else
        archive_ext=".tar.gz"
    fi

    filename="${BINARY_NAME}_${os}_${arch}${extension}${archive_ext}"
    url="https://github.com/$REPO/releases/download/$version/$filename"

    log "Downloading $url..."

    tmpdir=$(mktemp -d)
    trap 'rm -rf "$tmpdir"' EXIT

    archive_path="$tmpdir/$filename"
    version_no_prefix="${version#v}"
    checksum_path="$tmpdir/${BINARY_NAME}_${version_no_prefix}_checksums.txt"

    curl -fsSL -o "$archive_path" "$url" || error "Failed to download $url"

    log "Verifying checksum..."
    curl -fsSL -o "$checksum_path" "https://github.com/$REPO/releases/download/$version/${BINARY_NAME}_${version_no_prefix}_checksums.txt" || true

    if [ -f "$checksum_path" ]; then
        cd "$tmpdir"
        sha256sum -c "$checksum_path" --ignore-missing || error "Checksum verification failed"
    else
        log "Warning: Checksum file not found, skipping verification"
    fi

    log "Extracting archive..."
    if [ "$archive_ext" = ".zip" ]; then
        unzip -q "$archive_path" -d "$tmpdir" || error "Failed to extract archive"
    else
        tar -xzf "$archive_path" -C "$tmpdir" || error "Failed to extract archive"
    fi

    log "Installing to $INSTALL_DIR..."
    mkdir -p "$INSTALL_DIR"
    if [ -f "$tmpdir/LICENSE" ]; then
        log "Including LICENSE file"
    fi
    if [ -f "$tmpdir/README.md" ]; then
        log "Including README.md file"
    fi

    mv "$tmpdir/$BINARY_NAME${extension}" "$INSTALL_DIR/$BINARY_NAME" || error "Failed to move binary"
    chmod +x "$INSTALL_DIR/$BINARY_NAME"

    log "Installed $BINARY_NAME $version to $INSTALL_DIR/$BINARY_NAME"
    log ""
    log "Add to PATH by running:"
    log "  export PATH=\"$INSTALL_DIR:\$PATH\""
}

parse_args() {
    while [ $# -gt 0 ]; do
        case "$1" in
            -b|--bin-dir)
                INSTALL_DIR="$2"
                shift 2
                ;;
            -v|--version)
                VERSION="$2"
                shift 2
                ;;
            -r|--repo)
                REPO="$2"
                shift 2
                ;;
            -h|--help)
                usage
                ;;
            *)
                error "Unknown option: $1"
                ;;
        esac
    done
}

main() {
    if [ ! -t 1 ]; then
        :
    fi

    parse_args "$@"

    os=$(detect_os)
    arch=$(detect_arch)

    if [ -z "$VERSION" ]; then
        log "Fetching latest version..."
        VERSION=$(get_latest_version)
    fi

    log "Installing $BINARY_NAME $VERSION for $os/$arch..."
    download_and_install "$os" "$arch" "$VERSION"
}

main "$@"
