# Claude Instructions

## TDD + CD Workflow

Every change follows: RED → GREEN → REFACTOR → CI → COMMIT

```bash
# 1. Write failing test
go test ./path -run TestName -v  # Must fail

# 2. Make it pass
go test ./path -run TestName -v  # Must pass

# 3. Refactor
go test ./...  # All must pass

# 4. Run CI
./scripts/ci.sh

# 5. Commit
git add -A && git commit -m "type: description"
```

## Rules

- Tests document behavior, not implementation
- Mock only external dependencies (HTTP, filesystem)
- Black box CLI tests (build and run binary)
- Run `./scripts/ci.sh` before every commit

## Commands

| Action | Command |
|--------|---------|
| All tests | `go test ./...` |
| With race | `go test -race ./...` |
| Single test | `go test ./pkg/oauth -run TestFlow -v` |
| CI pipeline | `./scripts/ci.sh` |
| Build | `go build -o feedmix ./cmd/feedmix` |
| Release | `./scripts/deploy.sh` |

## CI Gates

All must pass: tests, race detector, contracts, golangci-lint, govulncheck, gosec
