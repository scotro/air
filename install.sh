#!/bin/sh
set -e

# Air (AI Runner) installer
# Usage: curl -sSL https://raw.githubusercontent.com/scotro/air/main/install.sh | sh

REPO="scotro/air"
BINARY_NAME="air"

# Colors (disabled if not a terminal)
if [ -t 1 ]; then
    RED='\033[0;31m'
    GREEN='\033[0;32m'
    YELLOW='\033[0;33m'
    NC='\033[0m'
else
    RED=''
    GREEN=''
    YELLOW=''
    NC=''
fi

info() { printf "${GREEN}info${NC}: %s\n" "$1"; }
warn() { printf "${YELLOW}warn${NC}: %s\n" "$1"; }
error() { printf "${RED}error${NC}: %s\n" "$1" >&2; exit 1; }

# Detect OS
detect_os() {
    case "$(uname -s)" in
        Darwin) echo "darwin" ;;
        Linux) echo "linux" ;;
        MINGW*|MSYS*|CYGWIN*) echo "windows" ;;
        *) error "Unsupported operating system: $(uname -s)" ;;
    esac
}

# Detect architecture
detect_arch() {
    case "$(uname -m)" in
        x86_64|amd64) echo "amd64" ;;
        arm64|aarch64) echo "arm64" ;;
        *) error "Unsupported architecture: $(uname -m)" ;;
    esac
}

# Get latest version from GitHub
get_latest_version() {
    curl -sS "https://api.github.com/repos/${REPO}/releases/latest" | \
        grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/'
}

# Determine install directory
get_install_dir() {
    if [ "$(id -u)" = "0" ]; then
        echo "/usr/local/bin"
    elif [ -n "$AIR_INSTALL_DIR" ]; then
        echo "$AIR_INSTALL_DIR"
    else
        echo "$HOME/.local/bin"
    fi
}

main() {
    info "Installing ${BINARY_NAME}..."

    OS=$(detect_os)
    ARCH=$(detect_arch)
    VERSION=$(get_latest_version)

    if [ -z "$VERSION" ]; then
        error "Could not determine latest version. Check your internet connection."
    fi

    info "Latest version: ${VERSION}"
    info "Platform: ${OS}/${ARCH}"

    # Construct download URL
    VERSION_NUM="${VERSION#v}"
    if [ "$OS" = "windows" ]; then
        ARCHIVE="${BINARY_NAME}_${VERSION_NUM}_${OS}_${ARCH}.zip"
    else
        ARCHIVE="${BINARY_NAME}_${VERSION_NUM}_${OS}_${ARCH}.tar.gz"
    fi

    DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${VERSION}/${ARCHIVE}"
    CHECKSUM_URL="https://github.com/${REPO}/releases/download/${VERSION}/checksums.txt"

    # Create temp directory
    TMP_DIR=$(mktemp -d)
    trap 'rm -rf "$TMP_DIR"' EXIT

    info "Downloading ${ARCHIVE}..."
    curl -sL "$DOWNLOAD_URL" -o "$TMP_DIR/$ARCHIVE" || error "Failed to download archive"

    info "Downloading checksums..."
    curl -sL "$CHECKSUM_URL" -o "$TMP_DIR/checksums.txt" || error "Failed to download checksums"

    # Verify checksum
    info "Verifying checksum..."
    cd "$TMP_DIR"
    EXPECTED=$(grep "$ARCHIVE" checksums.txt | awk '{print $1}')
    if [ -z "$EXPECTED" ]; then
        error "Checksum not found for $ARCHIVE"
    fi

    if command -v sha256sum >/dev/null 2>&1; then
        ACTUAL=$(sha256sum "$ARCHIVE" | awk '{print $1}')
    elif command -v shasum >/dev/null 2>&1; then
        ACTUAL=$(shasum -a 256 "$ARCHIVE" | awk '{print $1}')
    else
        warn "sha256sum not found, skipping checksum verification"
        ACTUAL="$EXPECTED"
    fi

    if [ "$EXPECTED" != "$ACTUAL" ]; then
        error "Checksum verification failed!\nExpected: $EXPECTED\nActual: $ACTUAL"
    fi

    # Extract archive
    info "Extracting..."
    if [ "$OS" = "windows" ]; then
        unzip -q "$ARCHIVE"
    else
        tar -xzf "$ARCHIVE"
    fi

    # Install binary
    INSTALL_DIR=$(get_install_dir)

    if [ ! -d "$INSTALL_DIR" ]; then
        info "Creating ${INSTALL_DIR}..."
        mkdir -p "$INSTALL_DIR"
    fi

    info "Installing to ${INSTALL_DIR}/${BINARY_NAME}..."
    mv "$BINARY_NAME" "$INSTALL_DIR/$BINARY_NAME"
    chmod +x "$INSTALL_DIR/$BINARY_NAME"

    # Verify installation
    if [ -x "$INSTALL_DIR/$BINARY_NAME" ]; then
        info "Successfully installed ${BINARY_NAME}!"
        "$INSTALL_DIR/$BINARY_NAME" version
    else
        error "Installation failed"
    fi

    # PATH reminder
    case ":$PATH:" in
        *":$INSTALL_DIR:"*) ;;
        *)
            echo ""
            warn "${INSTALL_DIR} is not in your PATH"
            echo "Add this to your shell profile (~/.bashrc, ~/.zshrc, etc.):"
            echo ""
            echo "  export PATH=\"\$PATH:${INSTALL_DIR}\""
            echo ""
            ;;
    esac
}

main "$@"
