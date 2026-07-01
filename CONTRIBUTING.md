# Contributing to patchright-go

## Development workflow

1. **Plan** — open an issue or discuss the change before writing code
2. **Code** — implement on a feature branch
3. **Test** — all existing tests must pass, add tests for new functionality
4. **Review** — check for bugs, concurrency issues, and memory efficiency
5. **PR** — create a pull request against `main`

## Setup

```bash
git clone https://github.com/status403com/patchright-go.git
cd patchright-go
go build ./...
go run ./cmd/patchright install chromium
```

## Running tests

```bash
# Unit tests only
go test -short ./...

# Full suite including integration tests (requires installed browser)
go test ./... -timeout 120s

# With race detector
go test -race ./...
```

## Code style

- Follow standard Go conventions (`gofmt`, `go vet`)
- Keep functions under 50 lines, files under 800 lines
- Handle all errors explicitly
- Use `sync.Once`, `atomic.Bool`, or mutexes for shared state — never bare bools
- Prefer immutable patterns where possible

## What to contribute

- Bug fixes with test cases
- Performance improvements (especially for high-throughput scenarios)
- Anti-detection improvements
- Documentation updates
- New examples

## What not to change

- `generated-*.go` files are auto-generated from the Playwright protocol — modify the generator, not the output (except for Patchright-specific additions clearly marked with comments)
- Don't add Firefox or WebKit support — Patchright is Chromium-only

## Updating to a new Patchright version

1. Update `patchrightCliVersion` in `run.go`
2. Check if `nodeVersion` needs updating
3. Run `scripts/generate-api.sh` if the protocol changed
4. Run all tests
5. Test `NewStealthPage` against a protected site

## License

By contributing, you agree that your contributions will be licensed under Apache-2.0.
