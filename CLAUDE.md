# patchright-go

Go library for Patchright - a patched Playwright driver that evades bot detection.

## Project overview

This is a fork of [playwright-go](https://github.com/playwright-community/playwright-go) modified to use the [Patchright](https://github.com/Kaliiiiiiiiii-Vinyzu/patchright) driver instead of standard Playwright. The Go client communicates with the Patchright Node.js server over stdio JSON pipes, which in turn controls Chromium via CDP.

### Architecture

```
Go client (patchright-go) → stdio JSON pipe → Patchright Node.js server → CDP → Chromium
```

The driver is assembled from two npm packages:
- `patchright` - the wrapper package with cli.js
- `patchright-core` - the actual patched Playwright driver

### Key differences from playwright-go

- Package name: `patchright` (not `playwright`)
- Downloads `patchright` + `patchright-core` npm packages instead of `playwright-core`
- All `Evaluate` methods pass `isolatedContext: true` by default (avoids Runtime.enable leak)
- Init script injection via route interception
- Chromium-only (Firefox/WebKit not supported by Patchright)
- Env vars use `PATCHRIGHT_` prefix (not `PLAYWRIGHT_`)
- Config can be passed as struct fields instead of env vars

## Development workflow

1. **Plan** - Create a plan for the change, get user review before coding
2. **Code** - Implement the change
3. **Review** - Review for bugs, performance issues (high-throughput, memory usage)
4. **Branch** - Create a new branch for the change
5. **Commit & Push** - Commit with clear messages, push to remote
6. **PR** - Create PR and merge to main after review

## Build & test

```bash
go build ./...
go vet ./...
go test -short ./...           # unit tests only
go test ./... -timeout 120s    # includes integration tests (downloads driver + browser)
```

## File structure

- `run.go` - Driver download, installation, and startup
- `playwright.go` - Main `Patchright` type definition
- `connection.go` - JSON pipe communication with the Node.js driver
- `transport.go` - stdio pipe transport
- `frame.go` - Frame evaluation methods (with `isolatedContext: true`)
- `worker.go` - Worker evaluation methods (with `isolatedContext: true`)
- `js_handle.go` - JSHandle evaluation methods (with `isolatedContext: true`)
- `page.go` - Page methods including init script route injection
- `route.go` - Route handling
- `browser_type.go` - Browser launch and context creation
- `generated-*.go` - Auto-generated types from Playwright protocol
- `cmd/patchright/` - CLI tool for driver/browser management

## Key env vars (all optional, struct fields preferred)

| Env var | Purpose |
|---------|---------|
| `PATCHRIGHT_DRIVER_PATH` | Override driver directory |
| `PATCHRIGHT_NODEJS_PATH` | Use preinstalled Node.js |
| `PATCHRIGHT_CLI_PATH` | Override cli.js path |
| `PATCHRIGHT_NPM_REGISTRY` | npm registry mirror |
| `NODE_MIRROR` | Node.js download mirror |

## Updating to new Patchright version

1. Update `patchrightCliVersion` in `run.go`
2. Check if `nodeVersion` needs updating
3. Regenerate types if Playwright protocol changed (run `scripts/generate-api.sh`)
4. Run all tests

## Credits

- [Patchright](https://github.com/Kaliiiiiiiiii-Vinyzu/patchright) by [Vinyzu](https://github.com/Kaliiiiiiiiii-Vinyzu) - the patched Playwright driver
- [patchright-python](https://github.com/Kaliiiiiiiiii-Vinyzu/patchright-python) by [Vinyzu](https://github.com/Kaliiiiiiiiii-Vinyzu) - Python language binding
- [patchright-nodejs](https://github.com/Kaliiiiiiiiii-Vinyzu/patchright-nodejs) by [Vinyzu](https://github.com/Kaliiiiiiiiii-Vinyzu) - Node.js language binding
- [patchright-dotnet](https://github.com/DevEnterpriseSoftware/patchright-dotnet) by [DevEnterpriseSoftware](https://github.com/DevEnterpriseSoftware) - .NET language binding (community)
- [playwright-go](https://github.com/playwright-community/playwright-go) by [Max Schmitt](https://github.com/mxschmitt) - the Go Playwright library this is forked from
- [Playwright](https://playwright.dev/) by Microsoft - the upstream browser automation framework
