# patchright-go

> [!WARNING]
> **Unofficial community port.** This is not affiliated with or endorsed by the Patchright project. If an official Go binding is released, it takes priority over this package. We maintain this for our own use and may stop at any time — adopt accordingly.

Port of [playwright-go](https://github.com/playwright-community/playwright-go) made to work with the [Patchright](https://github.com/Kaliiiiiiiiii-Vinyzu/patchright) driver, with a couple of small UX tweaks (`NewStealthPage`, automatic UA patching). Early version — has not been tested in a production environment yet.

Thin Go client wrapping the Node.js Patchright server over stdio pipes; all browser automation runs through the patched Playwright engine, Go handles serialization and the public API.

## How it works

```
Your Go code
    ↓ function calls
patchright-go (thin Go client)
    ↓ JSON messages over stdin/stdout pipes
Node.js process running patchright-core (patched Playwright server)
    ↓ Chrome DevTools Protocol (CDP) over WebSocket
Chromium browser process(es)
```

When you call `patchright.Run()`, the library spawns a Node.js child process running the patched Playwright server. Every Go API call (click, evaluate, goto, etc.) is serialized to JSON, sent over the stdin pipe, and the response is read back from stdout. The Node.js process handles all actual browser automation via CDP.

The Go side is ~3 MB of heap at 10 concurrent browsers — it's purely serialization and API surface. The real memory cost is Chromium (~395 MB per browser, ~77 MB per tab).

Node.js is required because the entire Playwright engine (CDP protocol handling, browser lifecycle, routing, tracing) is written in JavaScript. All Patchright language bindings (Python, Node.js, .NET, Go) are thin clients talking to this same server.

## What Patchright patches

| Patch | Description |
|-------|-------------|
| Runtime.enable leak | Executes JavaScript in isolated ExecutionContexts instead of using Runtime.enable CDP command |
| Console.enable leak | Disables Console API to prevent console-based detection |
| Command flags | Adds `--disable-blink-features=AutomationControlled`, removes `--enable-automation` |
| Closed Shadow DOM | Enables interaction with closed Shadow DOM elements |
| Init script injection | Uses route interception instead of Runtime.enable for init scripts |

**Chromium-only** — Firefox and WebKit are not supported by Patchright.

## Installation

```bash
go get github.com/status403com/patchright-go
```

Install the browser:

```go
patchright.Install(&patchright.RunOptions{
    Browsers: []string{"chromium"},
})
```

Or via CLI:

```bash
go run github.com/status403com/patchright-go/cmd/patchright@latest install chromium
```

Google Chrome is recommended over Chromium for better anti-detection:

```bash
go run github.com/status403com/patchright-go/cmd/patchright@latest install chrome
```

## Quick start

```go
package main

import (
    "fmt"
    "log"

    patchright "github.com/status403com/patchright-go"
)

func main() {
    pw, err := patchright.Run()
    if err != nil {
        log.Fatal(err)
    }
    defer pw.Stop()

    // Prefer headful over headless — anti-bot solutions can still detect
    // headless browsers via deep fingerprinting even with Patchright patches
    browser, err := pw.Chromium.Launch(patchright.BrowserTypeLaunchOptions{
        Headless: patchright.Bool(false),
    })
    if err != nil {
        log.Fatal(err)
    }
    defer browser.Close()

    // NewStealthPage auto-patches the HeadlessChrome user agent
    page, err := browser.NewStealthPage()
    if err != nil {
        log.Fatal(err)
    }

    page.Goto("https://example.com")

    title, _ := page.Title()
    fmt.Println(title)
}
```

## Headless vs headful

**Always prefer headful mode (`Headless: false`) unless you have a specific reason to use headless.** Anti-bot solutions like PerimeterX, Akamai, and Datadome can still detect headless browsers through deep fingerprinting (WebGL renderer, `navigator.plugins`, screen/window dimension mismatches, etc.) even with all Patchright patches applied.

In testing, headful mode passes Walmart.ca's PerimeterX protection on product pages, while headless mode gets blocked — even with a patched user agent and `navigator.webdriver: false`.

```go
// Headful (recommended) — passes advanced anti-bot
browser, err := pw.Chromium.Launch(patchright.BrowserTypeLaunchOptions{
    Headless: patchright.Bool(false),
})

// Headless — use only when headful is not possible (CI, serverless, etc.)
browser, err := pw.Chromium.Launch()
```

If you're getting blocked despite using Patchright, switching to headful mode is the first thing to try.

## Stealth API

Headless Chromium sends `HeadlessChrome` in the default user agent, which is an instant detection signal. patchright-go provides stealth methods that automatically patch this:

```go
// NewStealthPage — creates a page with a patched user agent
// HeadlessChrome/149.0.7827.55 → Chrome/149.0.0.0
page, err := browser.NewStealthPage()

// NewStealthContext — creates a context with a patched user agent
// All pages created from this context share the patched UA
context, err := browser.NewStealthContext()
page1, _ := context.NewPage()
page2, _ := context.NewPage()

// PatchHeadlessUA — standalone helper to patch any UA string
fixedUA := patchright.PatchHeadlessUA(rawUA)
```

The patched UA applies to `navigator.userAgent` and all HTTP requests (fetch, XHR, navigation).

Alternatively, use Google Chrome which sends the correct UA natively:

```go
browser, err := pw.Chromium.Launch(patchright.BrowserTypeLaunchOptions{
    Channel: patchright.String("chrome"),
})
```

## Differences from playwright-go

| Feature | playwright-go | patchright-go |
|---------|--------------|---------------|
| Package name | `playwright` | `patchright` |
| Driver | `playwright-core` | `patchright` + `patchright-core` |
| Main type | `*Playwright` | `*Patchright` |
| Driver type | `*PlaywrightDriver` | `*PatchrightDriver` |
| Bot detection | Detected | Evades detection |
| Browsers | Chromium, Firefox, WebKit | Chromium only |
| JS evaluation | Standard context | Isolated context (default) |
| Env prefix | `PLAYWRIGHT_` | `PATCHRIGHT_` |
| Stealth UA | Not available | `NewStealthPage` / `NewStealthContext` |

## Migration from playwright-go

1. Change your import: `playwright` -> `patchright "github.com/status403com/patchright-go"`
2. Replace type references: `playwright.Playwright` -> `patchright.Patchright`
3. Replace driver type: `playwright.PlaywrightDriver` -> `patchright.PatchrightDriver`
4. Update env vars: `PLAYWRIGHT_*` -> `PATCHRIGHT_*`
5. Remove Firefox/WebKit usage (Chromium only)
6. Use `NewStealthPage` / `NewStealthContext` instead of `NewPage` / `NewContext` for anti-detection

## Configuration

All configuration can be set via `RunOptions` struct fields or environment variables. Struct fields take precedence.

```go
patchright.Run(&patchright.RunOptions{
    DriverDirectory: "/custom/driver/path",
    NodeJSPath:      "/usr/local/bin/node",
    NpmRegistry:     "https://registry.npmmirror.com",
    Browsers:        []string{"chromium"},
})
```

| RunOptions field | Env var | Description |
|-----------------|---------|-------------|
| `DriverDirectory` | `PATCHRIGHT_DRIVER_PATH` | Driver installation directory |
| `NodeJSPath` | `PATCHRIGHT_NODEJS_PATH` | Path to Node.js binary |
| `CLIPath` | `PATCHRIGHT_CLI_PATH` | Path to cli.js |
| `NpmRegistry` | `PATCHRIGHT_NPM_REGISTRY` | npm registry URL |
| `NodeMirror` | `NODE_MIRROR` | Node.js download mirror |

## Running multiple browsers

Patchright supports running many browser instances from a single Go process:

```go
pw, _ := patchright.Run()
defer pw.Stop()

var wg sync.WaitGroup
for i := 0; i < 50; i++ {
    wg.Add(1)
    go func() {
        defer wg.Done()
        browser, _ := pw.Chromium.Launch()
        defer browser.Close()
        page, _ := browser.NewStealthPage()
        page.Goto("https://example.com")
    }()
}
wg.Wait()
```

### Memory usage

The Go client itself uses ~2-3 MB regardless of browser count. Memory is dominated by Chromium processes. Tabs within a single browser share renderer processes and are significantly cheaper than separate browsers.

| RAM | Separate browsers | Tabs (single browser) |
|-----|------------------|-----------------------|
| 1 GB | ~2 | ~10 |
| 2 GB | ~4 | ~24 |
| 4 GB | ~9 | ~50 |
| 8 GB | ~20 | ~104 |
| 16 GB | ~41 | ~211 |

**Use tabs when possible** — a single browser with many tabs via `NewStealthContext` uses ~77 MB per tab, while separate browser instances use ~395 MB each. Use separate browsers only when you need isolated browser fingerprints or independent cookie jars beyond what contexts provide.

## API

The API is identical to [playwright-go](https://pkg.go.dev/github.com/mxschmitt/playwright-go) with the type renames listed above, plus the stealth methods. Refer to the playwright-go documentation for the full API reference and `docs/llm-guide.md` for a concise cheat sheet.

## Credits

- [Patchright](https://github.com/Kaliiiiiiiiii-Vinyzu/patchright) by [Vinyzu](https://github.com/Kaliiiiiiiiii-Vinyzu)
- [patchright-python](https://github.com/Kaliiiiiiiiii-Vinyzu/patchright-python) by [Vinyzu](https://github.com/Kaliiiiiiiiii-Vinyzu)
- [patchright-nodejs](https://github.com/Kaliiiiiiiiii-Vinyzu/patchright-nodejs) by [Vinyzu](https://github.com/Kaliiiiiiiiii-Vinyzu)
- [patchright-dotnet](https://github.com/DevEnterpriseSoftware/patchright-dotnet) by [DevEnterpriseSoftware](https://github.com/DevEnterpriseSoftware)
- [playwright-go](https://github.com/playwright-community/playwright-go) by [Max Schmitt](https://github.com/mxschmitt)
- [Playwright](https://playwright.dev/) by Microsoft

## License

Apache-2.0
