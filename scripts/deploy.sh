#!/usr/bin/env bash
set -euo pipefail

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

step() { echo -e "${YELLOW}>>> $1${NC}"; }
success() { echo -e "${GREEN}✓ $1${NC}"; }
fail() { echo -e "${RED}✗ $1${NC}"; exit 1; }

echo "=== Feedmix Continuous Delivery ==="
echo ""

# Check for uncommitted changes
if [[ -n $(git status --porcelain) ]]; then
    fail "Uncommitted changes detected. Commit first with: git add -A && git commit -m \"type: description\""
fi

# Check current branch
BRANCH=$(git rev-parse --abbrev-ref HEAD)
if [[ "$BRANCH" != "main" ]]; then
    echo -e "${YELLOW}⚠ Warning: Not on main branch (current: $BRANCH)${NC}"
    read -p "Continue anyway? [y/N] " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        exit 1
    fi
fi

VERSION="${VERSION:-$(git describe --tags --always --dirty 2>/dev/null || echo "dev")}"
BUILD_DIR="dist"

step "Running CI pipeline..."
./scripts/ci.sh || fail "CI failed"
success "CI passed"
echo ""

step "Building release binaries..."
rm -rf "$BUILD_DIR" && mkdir -p "$BUILD_DIR"

for platform in linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64; do
    GOOS="${platform%/*}"
    GOARCH="${platform#*/}"
    out="$BUILD_DIR/feedmix-$VERSION-$GOOS-$GOARCH"
    [ "$GOOS" = "windows" ] && out="${out}.exe"
    GOOS=$GOOS GOARCH=$GOARCH go build -ldflags "-X main.version=$VERSION" -o "$out" ./cmd/feedmix
done

cd "$BUILD_DIR" && sha256sum feedmix-* > checksums.txt && cd ..
success "Built 5 platform binaries"
echo ""

step "Running E2E smoke tests..."
VERSION="$VERSION" ./tests/e2e-smoke.sh || fail "E2E smoke tests failed"
success "E2E smoke tests passed"
echo ""

# All validations passed - deploy automatically
step "Deploying to GitHub (automatic push)..."
CURRENT_COMMIT=$(git rev-parse HEAD)
REMOTE_COMMIT=$(git rev-parse origin/$BRANCH 2>/dev/null || echo "")

if [[ "$CURRENT_COMMIT" == "$REMOTE_COMMIT" ]]; then
    echo -e "${YELLOW}ℹ Current commit already pushed to origin/$BRANCH${NC}"
    echo -e "${YELLOW}ℹ Skipping push (nothing to deploy)${NC}"
else
    git push origin "$BRANCH" || fail "Git push failed"
    success "Pushed to origin/$BRANCH"
    echo ""
    echo -e "${GREEN}=== Deployment Triggered ===${NC}"
    echo ""
    echo "GitHub Actions will now:"
    echo "  1. Run full CI validation (tests, lint, security)"
    echo "  2. Build release binaries"
    echo "  3. Run E2E smoke tests on binaries"
    echo "  4. If smoke tests fail → Auto-rollback"
    echo ""
    echo -e "${YELLOW}Monitor deployment:${NC}"
    echo "  https://github.com/gauthierbraillon/feedmix/actions"
    echo ""
fi

echo -e "${GREEN}=== Continuous Delivery Complete ===${NC}"
echo ""
echo "To create a release tag:"
echo "  git tag v1.2.3 && git push origin v1.2.3"
echo ""
