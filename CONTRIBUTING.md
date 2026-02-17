# Contributing to Feedmix

## Development Setup

```bash
git clone https://github.com/gauthierbraillon/feedmix
cd feedmix
go mod download
go test ./...
```

**Requirements**: Go 1.22+

## Workflow

This project uses ATDD (Acceptance Test Driven Development) — every change starts with a failing test.

```
RED → GREEN → REFACTOR → COMMIT → DEPLOY
```

1. **RED** — write a test that describes the new behaviour (it must fail)
2. **GREEN** — write the minimum code to make the test pass
3. **REFACTOR** — clean up without breaking tests
4. **COMMIT** — conventional commit message (see below)
5. **DEPLOY** — `./scripts/deploy.sh` (validates, builds, pushes)

### Fast feedback loop

```bash
./scripts/ci.sh      # vet + tests + race detector + security + build (<2 min)
```

### Full deployment

```bash
./scripts/deploy.sh  # ci.sh + release binaries + E2E + auto-push
```

## Commit Messages

Follow [Conventional Commits](https://www.conventionalcommits.org/):

| Type | Effect |
|------|--------|
| `feat:` | New minor release |
| `fix:` | New patch release |
| `feat!:` or `BREAKING CHANGE:` | New major release |
| `docs:`, `chore:`, `refactor:`, `test:` | CI only, no release |

## What to Mock

- **Mock**: external APIs (YouTube Data API, OAuth endpoints), filesystem, browser
- **Do not mock**: our own internal packages

## Pull Requests

- One feature or fix per PR
- All tests must pass (`go test -race ./...`)
- Conventional commit title on the PR

## Reporting Issues

Use the [bug report template](https://github.com/gauthierbraillon/feedmix/issues/new?template=bug.yml).

## License

By contributing you agree your changes will be licensed under the [MIT License](LICENSE).
