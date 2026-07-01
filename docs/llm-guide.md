# patchright-go LLM Quick Reference

Concise reference for using patchright-go with LLMs. Import as:
```go
import patchright "github.com/status403com/patchright-go"
```

## Install driver + browser (one-time)

```go
patchright.Install(&patchright.RunOptions{
    Browsers: []string{"chromium"},
})
```

## Start and stop

```go
pw, err := patchright.Run()
if err != nil { log.Fatal(err) }
defer pw.Stop()
```

## Launch browser

**Always prefer headful mode.** Anti-bot solutions can detect headless browsers
through deep fingerprinting (WebGL, navigator.plugins, screen dimensions) even
with Patchright patches. If you're getting blocked, switch to headful first.

```go
// Headful (recommended) — passes advanced anti-bot like PerimeterX
browser, err := pw.Chromium.Launch(patchright.BrowserTypeLaunchOptions{
    Headless: patchright.Bool(false),
})

// Headless — only when headful is not possible (CI, serverless)
browser, err := pw.Chromium.Launch()

// Use Google Chrome channel (requires Chrome installed on system)
browser, err := pw.Chromium.Launch(patchright.BrowserTypeLaunchOptions{
    Channel:  patchright.String("chrome"),
    Headless: patchright.Bool(false),
})

defer browser.Close()
```

## Create page and navigate

```go
// Standard page (no UA patching)
page, err := browser.NewPage()

// Stealth page (recommended) — auto-patches HeadlessChrome UA to real Chrome format
// Affects navigator.userAgent AND all HTTP requests (fetch, XHR, navigation)
page, err := browser.NewStealthPage()

_, err = page.Goto("https://example.com")
title, err := page.Title()
```

## Stealth context (multiple pages sharing patched UA)

```go
context, err := browser.NewStealthContext()
page1, err := context.NewPage()
page2, err := context.NewPage()
// Both pages share the patched UA
```

## Create context with options

```go
context, err := browser.NewContext(patchright.BrowserNewContextOptions{
    UserAgent: patchright.String("custom-ua"),
    Viewport: &patchright.Size{Width: 1920, Height: 1080},
})
page, err := context.NewPage()
```

## Evaluate JavaScript

```go
// Return value
result, err := page.Evaluate("() => document.title")

// With argument
result, err := page.Evaluate("(x) => x * 2", 21)

// Return handle
handle, err := page.EvaluateHandle("() => document")
```

All evaluate methods automatically use isolated execution contexts (Patchright's anti-detection feature).

## Anti-detection best practices

**IMPORTANT**: Headless Chromium sends `HeadlessChrome` in the default user agent,
which is an instant detection signal. Use `NewStealthPage` / `NewStealthContext`
to automatically fix this.

### Easiest approach — stealth methods (recommended)

```go
browser, err := pw.Chromium.Launch()

// Auto-patches UA: HeadlessChrome → Chrome, 149.0.7827.55 → 149.0.0.0
// Applied to navigator.userAgent AND all HTTP requests (fetch, XHR, etc)
page, err := browser.NewStealthPage()
```

### Manual approach — set UserAgent yourself

```go
context, err := browser.NewContext(patchright.BrowserNewContextOptions{
    UserAgent: patchright.String("Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/149.0.0.0 Safari/537.36"),
    Viewport:  &patchright.Size{Width: 1920, Height: 1080},
    Locale:    patchright.String("en-US"),
})
page, err := context.NewPage()
```

### Use the PatchHeadlessUA helper directly

```go
rawUA := "Mozilla/5.0 ... HeadlessChrome/149.0.7827.55 ..."
fixedUA := patchright.PatchHeadlessUA(rawUA)
// → "Mozilla/5.0 ... Chrome/149.0.0.0 ..."
```

### Google Chrome channel (requires Chrome installed on system)

```go
// First: go run ./cmd/patchright install chrome
browser, err := pw.Chromium.Launch(patchright.BrowserTypeLaunchOptions{
    Channel: patchright.String("chrome"),
})
// Chrome channel sends correct UA natively — no patching needed
```

### Key rules
1. **Use headful mode** (`Headless: false`) — headless browsers are detectable via deep fingerprinting (WebGL, navigator.plugins, screen dimensions) even with Patchright. If you're getting blocked, this is the most likely fix
2. **Use `NewStealthPage`/`NewStealthContext`** for automatic UA patching — simplest option
3. **Real Chrome UA format** uses major version only: `Chrome/149.0.0.0`, never `Chrome/149.0.7827.55`
4. **Set viewport** to a common resolution (1920x1080, 1366x768, etc)
5. **Set locale** to match target site region
6. **Wait properly** - use `WaitUntilStateDomcontentloaded` instead of `WaitUntilStateNetworkidle` for sites with heavy analytics

### Troubleshooting blocks
If you're getting blocked despite using Patchright:
1. Switch to headful mode (`Headless: false`) — this is the #1 fix
2. Use `NewStealthPage` instead of `NewPage`
3. Try `Channel: "chrome"` with Google Chrome installed
4. Set a realistic viewport and locale
5. Add delays between actions to mimic human behavior

## Click, type, fill

```go
page.Click("button#submit")
page.Fill("input[name=email]", "test@example.com")
page.Type("input[name=search]", "query")
page.Press("input", "Enter")
```

## Wait for elements

```go
page.WaitForSelector("div.loaded")
page.WaitForLoadState(patchright.LoadStateNetworkidle)
```

## Locators (preferred)

```go
loc := page.Locator("button.submit")
loc.Click()
text, err := loc.TextContent()
loc.Fill("value")
```

## Screenshots

```go
// Full page
page.Screenshot(patchright.PageScreenshotOptions{
    Path:     patchright.String("page.png"),
    FullPage: patchright.Bool(true),
})

// Element
loc.Screenshot(patchright.LocatorScreenshotOptions{
    Path: patchright.String("element.png"),
})
```

## PDF (headless only)

```go
page.PDF(patchright.PagePDFOptions{
    Path: patchright.String("page.pdf"),
})
```

## Network interception

```go
page.Route("**/*.png", func(route patchright.Route) {
    route.Abort()
})

page.Route("**/api/**", func(route patchright.Route) {
    route.Continue()
})
```

## Multiple pages / contexts

```go
// Multiple pages in same browser
page1, _ := browser.NewPage()
page2, _ := browser.NewPage()

// Isolated contexts (separate cookies, storage)
ctx1, _ := browser.NewContext()
ctx2, _ := browser.NewContext()
p1, _ := ctx1.NewPage()
p2, _ := ctx2.NewPage()
```

## Concurrent browsers

```go
pw, _ := patchright.Run()
defer pw.Stop()

var wg sync.WaitGroup
for i := 0; i < 10; i++ {
    wg.Add(1)
    go func() {
        defer wg.Done()
        browser, _ := pw.Chromium.Launch(patchright.BrowserTypeLaunchOptions{
            Headless: patchright.Bool(false),
        })
        defer browser.Close()
        page, _ := browser.NewStealthPage()
        page.Goto("https://example.com")
    }()
}
wg.Wait()
```

## Memory usage

Go client uses ~2-3 MB. Memory is dominated by Chromium. Tabs are much cheaper than browsers.

| RAM | Browsers (~395 MB each) | Tabs (~77 MB each) |
|-----|------------------------|--------------------|
| 1 GB | ~2 | ~10 |
| 2 GB | ~4 | ~24 |
| 4 GB | ~9 | ~50 |
| 8 GB | ~20 | ~104 |
| 16 GB | ~41 | ~211 |

Prefer tabs (pages in one browser/context) when you don't need separate fingerprints:
```go
browser, _ := pw.Chromium.Launch(patchright.BrowserTypeLaunchOptions{
    Headless: patchright.Bool(false),
})
ctx, _ := browser.NewStealthContext()
page1, _ := ctx.NewPage()
page2, _ := ctx.NewPage() // shares browser process, ~77 MB instead of ~395 MB
```

## Pointer helpers

patchright-go uses pointer types for optional fields. Helpers:
```go
patchright.String("value")  // *string
patchright.Bool(true)       // *bool
patchright.Float(1000)      // *float64
patchright.Int(5)           // *int
```

## Key types

| Type | Description |
|------|-------------|
| `*Patchright` | Main instance from `Run()` |
| `BrowserType` | `pw.Chromium` |
| `Browser` | From `Launch()` |
| `BrowserContext` | From `NewContext()` |
| `Page` | From `NewPage()` |
| `Locator` | From `page.Locator()` |
| `Route` | In route handlers |
| `Frame` | From `page.MainFrame()` |

## Env vars (all optional)

| Var | Purpose |
|-----|---------|
| `PATCHRIGHT_DRIVER_PATH` | Override driver directory |
| `PATCHRIGHT_NODEJS_PATH` | Custom Node.js binary |
| `PATCHRIGHT_CLI_PATH` | Custom cli.js path |
| `PATCHRIGHT_NPM_REGISTRY` | npm mirror |
| `NODE_MIRROR` | Node.js download mirror |

All env vars can also be set via `RunOptions` struct fields (which take precedence).
