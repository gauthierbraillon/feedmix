#!/usr/bin/env bash
# CI script for feedmix
# Runs all checks required before merging/deploying

set -euo pipefail

echo "=== Feedmix CI Pipeline ==="
echo ""

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

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

# Step 1: Verify Go is installed
step "Checking Go installation..."
if ! command -v go &> /dev/null; then
    fail "Go is not installed"
fi
success "Go $(go version | cut -d' ' -f3) installed"

# Step 2: Download dependencies
step "Downloading dependencies..."
go mod download
success "Dependencies downloaded"

# Step 3: Run go vet
step "Running go vet..."
if ! go vet ./...; then
    fail "go vet found issues"
fi
success "go vet passed"

# Step 4: Run tests
step "Running tests..."
if ! go test -race -v ./... 2>&1; then
    fail "Tests failed"
fi
success "All tests passed"

# Step 5: Run contract tests specifically
step "Running contract tests..."
if ! go test -v ./pkg/contracts/... 2>&1; then
    fail "Contract tests failed"
fi
success "Contract tests passed"

# Step 6: Build binary
step "Building binary..."
if ! go build -o feedmix ./cmd/feedmix; then
    fail "Build failed"
fi
success "Binary built: ./feedmix"

# Step 7: Verify binary runs
step "Verifying binary..."
if ! ./feedmix --version; then
    fail "Binary verification failed"
fi
success "Binary verified"

echo ""
echo -e "${GREEN}=== CI Pipeline Complete ===${NC}"
echo ""
