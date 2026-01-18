#!/usr/bin/env bash
# Deploy script for feedmix
# Builds release binaries for multiple platforms

set -euo pipefail

VERSION="${VERSION:-$(git describe --tags --always --dirty 2>/dev/null || echo "dev")}"
BUILD_DIR="dist"

echo "=== Feedmix Deploy Pipeline ==="
echo "Version: $VERSION"
echo ""

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

step() {
    echo -e "${YELLOW}>>> $1${NC}"
}

success() {
    echo -e "${GREEN}✓ $1${NC}"
}

fail() {
    echo -e "${RED}✗ $1${NC}"
    exit 1
}

# Step 1: Run CI first
step "Running CI pipeline..."
if ! ./scripts/ci.sh; then
    fail "CI pipeline failed - cannot deploy"
fi
success "CI pipeline passed"

# Step 2: Clean build directory
step "Cleaning build directory..."
rm -rf "$BUILD_DIR"
mkdir -p "$BUILD_DIR"
success "Build directory cleaned"

# Step 3: Build for multiple platforms
PLATFORMS=(
    "linux/amd64"
    "linux/arm64"
    "darwin/amd64"
    "darwin/arm64"
    "windows/amd64"
)

step "Building release binaries..."
for platform in "${PLATFORMS[@]}"; do
    GOOS="${platform%/*}"
    GOARCH="${platform#*/}"
    output="$BUILD_DIR/feedmix-$VERSION-$GOOS-$GOARCH"

    if [ "$GOOS" = "windows" ]; then
        output="${output}.exe"
    fi

    echo "  Building for $GOOS/$GOARCH..."
    GOOS=$GOOS GOARCH=$GOARCH go build \
        -ldflags "-X main.version=$VERSION" \
        -o "$output" \
        ./cmd/feedmix
done
success "Release binaries built"

# Step 4: Generate checksums
step "Generating checksums..."
cd "$BUILD_DIR"
sha256sum feedmix-* > checksums.txt
cd ..
success "Checksums generated"

# Step 5: Show artifacts
step "Build artifacts:"
ls -la "$BUILD_DIR/"

echo ""
echo -e "${GREEN}=== Deploy Pipeline Complete ===${NC}"
echo ""
echo "Artifacts are in: $BUILD_DIR/"
echo ""
echo "To publish a release:"
echo "  1. Tag the release: git tag v$VERSION"
echo "  2. Push the tag: git push origin v$VERSION"
echo "  3. Upload artifacts to GitHub releases"
echo ""
