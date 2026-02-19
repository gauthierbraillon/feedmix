#!/usr/bin/env bash
set -euo pipefail

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

step() { echo -e "${YELLOW}>>> $1${NC}"; }
success() { echo -e "${GREEN}✓ $1${NC}"; }
fail() { echo -e "${RED}✗ $1${NC}"; exit 1; }

echo "=== Feedmix E2E Smoke Tests ==="

# Determine which binary to test
BINARY=""

# Option 1: Use binary from GitHub release (only for tag pushes)
if [[ "${GITHUB_REF:-}" == refs/tags/* ]]; then
    VERSION="${GITHUB_REF#refs/tags/}"
    PLATFORM="$(uname -s | tr '[:upper:]' '[:lower:]')"
    ARCH="$(uname -m)"

    # Convert arch names (x86_64 -> amd64, aarch64 -> arm64)
    case "$ARCH" in
        x86_64) ARCH="amd64" ;;
        aarch64) ARCH="arm64" ;;
    esac

    BINARY_NAME="feedmix-${VERSION}-${PLATFORM}-${ARCH}"
    if [ "$PLATFORM" = "windows" ]; then
        BINARY_NAME="${BINARY_NAME}.exe"
    fi

    step "Downloading release binary: $BINARY_NAME"

    # Download from GitHub release
    DOWNLOAD_URL="https://github.com/gauthierbraillon/feedmix/releases/download/${VERSION}/${BINARY_NAME}"
    if command -v curl &>/dev/null; then
        curl -sSL "$DOWNLOAD_URL" -o "$BINARY_NAME" || fail "Failed to download binary"
    elif command -v wget &>/dev/null; then
        wget -q "$DOWNLOAD_URL" -O "$BINARY_NAME" || fail "Failed to download binary"
    else
        fail "Neither curl nor wget available for download"
    fi

    chmod +x "$BINARY_NAME"
    BINARY="./$BINARY_NAME"
    success "Downloaded $BINARY_NAME"

    # Cleanup on exit
    trap "rm -f $BINARY_NAME" EXIT
fi

# Option 2: Use locally built binary (dist/ directory from deploy.sh)
if [ -z "$BINARY" ] && [ -d "dist" ]; then
    step "Using locally built binary from dist/"

    PLATFORM="$(uname -s | tr '[:upper:]' '[:lower:]')"
    ARCH="$(uname -m)"

    case "$ARCH" in
        x86_64) ARCH="amd64" ;;
        aarch64) ARCH="arm64" ;;
    esac

    VERSION="${VERSION:-dev}"
    BINARY_NAME="feedmix-${VERSION}-${PLATFORM}-${ARCH}"
    if [ "$PLATFORM" = "windows" ]; then
        BINARY_NAME="${BINARY_NAME}.exe"
    fi

    BINARY="dist/$BINARY_NAME"

    if [ ! -f "$BINARY" ]; then
        fail "Binary not found: $BINARY"
    fi

    chmod +x "$BINARY"
    success "Using $BINARY"
fi

# Option 3: Use feedmix in current directory (from ci.sh)
if [ -z "$BINARY" ] && [ -f "./feedmix" ]; then
    step "Using ./feedmix from current directory"
    BINARY="./feedmix"
    success "Using ./feedmix"
fi

# Option 4: Use feedmix in PATH (installed via go install)
if [ -z "$BINARY" ] && command -v feedmix &>/dev/null; then
    step "Using feedmix from PATH"
    BINARY="feedmix"
    success "Using feedmix from PATH"
fi

if [ -z "$BINARY" ]; then
    fail "No feedmix binary found. Build with './scripts/ci.sh' or './scripts/deploy.sh'"
fi

echo ""
step "Test 1: Version command"
VERSION_OUTPUT=$($BINARY --version) || fail "feedmix --version failed"
if [[ ! "$VERSION_OUTPUT" =~ "feedmix version" ]]; then
    fail "Version output doesn't contain 'feedmix version': $VERSION_OUTPUT"
fi
success "Version: $VERSION_OUTPUT"

step "Test 2: Config command"
CONFIG_OUTPUT=$($BINARY config) || fail "feedmix config failed"
if [[ ! "$CONFIG_OUTPUT" =~ "config" ]] && [[ ! "$CONFIG_OUTPUT" =~ ".config" ]]; then
    fail "Config output doesn't contain config path: $CONFIG_OUTPUT"
fi
success "Config: $CONFIG_OUTPUT"

step "Test 3: Feed help command"
FEED_HELP_OUTPUT=$($BINARY feed --help 2>&1) || true
if [[ ! "$FEED_HELP_OUTPUT" =~ "feed" ]] && [[ ! "$FEED_HELP_OUTPUT" =~ "View" ]]; then
    fail "Feed help output unexpected: $FEED_HELP_OUTPUT"
fi
success "Feed help displayed"

step "Test 4: Invalid command returns error"
if $BINARY nonexistent-command &>/dev/null; then
    fail "Invalid command should return error"
fi
success "Invalid command correctly rejected"

echo ""
echo -e "${GREEN}=== All E2E Smoke Tests Passed ===${NC}"
echo ""
