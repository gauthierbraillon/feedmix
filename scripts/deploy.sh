#!/usr/bin/env bash
set -euo pipefail

VERSION="${VERSION:-$(git describe --tags --always --dirty 2>/dev/null || echo "dev")}"
BUILD_DIR="dist"

echo "=== Feedmix Deploy (${VERSION}) ==="

# Run CI first
./scripts/ci.sh || exit 1

# Clean and build
rm -rf "$BUILD_DIR" && mkdir -p "$BUILD_DIR"

for platform in linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64; do
    GOOS="${platform%/*}"
    GOARCH="${platform#*/}"
    out="$BUILD_DIR/feedmix-$VERSION-$GOOS-$GOARCH"
    [ "$GOOS" = "windows" ] && out="${out}.exe"
    echo "Building $GOOS/$GOARCH..."
    GOOS=$GOOS GOARCH=$GOARCH go build -ldflags "-X main.version=$VERSION" -o "$out" ./cmd/feedmix
done

cd "$BUILD_DIR" && sha256sum feedmix-* > checksums.txt && cd ..

echo -e "\nArtifacts in $BUILD_DIR/:"
ls -la "$BUILD_DIR/"
echo -e "\nTo release: git tag v$VERSION && git push origin v$VERSION"
