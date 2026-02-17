#!/usr/bin/env bash
set -euo pipefail

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

step() { echo -e "${YELLOW}>>> $1${NC}"; }
success() { echo -e "${GREEN}✓ $1${NC}"; }
fail() { echo -e "${RED}✗ $1${NC}"; exit 1; }

echo "=== Feedmix CI ==="

step "Checking Go..."
command -v go &>/dev/null || fail "Go not installed"
success "Go $(go version | cut -d' ' -f3)"

step "Dependencies..."
go mod download
success "Done"

step "Vet..."
go vet ./... || fail "go vet failed"
success "Passed"

step "Tests (includes contracts)..."
go test -race -cover ./... || fail "Tests failed"
success "Passed"

step "Integration tests (Ubuntu)..."
go test -tags=integration -v ./pkg/oauth/... ./cmd/feedmix/... || fail "Integration tests failed"
success "Passed"

step "Security (govulncheck)..."
if command -v govulncheck &>/dev/null; then
    govulncheck ./... || fail "Vulnerabilities found"
    success "Passed"
else
    echo "  Skipped (install: go install golang.org/x/vuln/cmd/govulncheck@v1.1.4)"
fi

step "Build..."
VERSION=$(git describe --tags --always --dirty 2>/dev/null || echo "dev")
go build -ldflags="-X main.version=$VERSION" -o feedmix ./cmd/feedmix || fail "Build failed"
success "Built ./feedmix (version: $VERSION)"

step "Verify..."
./feedmix --version || fail "Binary failed"
success "Verified"

echo -e "\n${GREEN}=== CI Passed ===${NC}\n"
