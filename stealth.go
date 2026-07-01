package patchright

import (
	"fmt"
	"regexp"
	"strings"
)

var chromeVersionRe = regexp.MustCompile(`(Chrome/)(\d+)\.\d+\.\d+\.\d+`)

// PatchHeadlessUA transforms a headless Chromium user agent into a realistic
// Chrome user agent. It replaces "HeadlessChrome" with "Chrome" and normalizes
// the version to major-only format (e.g. 149.0.0.0) matching real Chrome
// behavior.
func PatchHeadlessUA(ua string) string {
	ua = strings.Replace(ua, "HeadlessChrome", "Chrome", 1)
	ua = chromeVersionRe.ReplaceAllString(ua, "${1}${2}.0.0.0")
	return ua
}

// NewStealthPage creates a new page with a patched user agent that masks
// headless browser fingerprints. It fetches the browser's default UA, patches
// it, and creates a context with the corrected UA applied.
//
// This is the recommended way to create pages for anti-detection use cases.
// Equivalent to calling NewStealthContext + NewPage.
func (b *browserImpl) NewStealthPage(options ...BrowserNewPageOptions) (Page, error) {
	opts := BrowserNewContextOptions{}
	if len(options) == 1 {
		opts = BrowserNewContextOptions(options[0])
	}
	if opts.UserAgent == nil {
		patchedUA, err := b.getPatchedUA()
		if err != nil {
			return nil, err
		}
		opts.UserAgent = String(patchedUA)
	}
	context, err := b.NewContext(opts)
	if err != nil {
		return nil, err
	}
	page, err := context.NewPage()
	if err != nil {
		context.Close()
		return nil, err
	}
	page.(*pageImpl).ownedContext = context
	context.(*browserContextImpl).ownedPage = page
	return page, nil
}

// NewStealthContext creates a new browser context with a patched user agent
// that masks headless browser fingerprints. If UserAgent is not set in
// options, it automatically fetches the browser's default UA and patches it.
func (b *browserImpl) NewStealthContext(options ...BrowserNewContextOptions) (BrowserContext, error) {
	opts := BrowserNewContextOptions{}
	if len(options) == 1 {
		opts = options[0]
	}
	if opts.UserAgent == nil {
		patchedUA, err := b.getPatchedUA()
		if err != nil {
			return nil, err
		}
		opts.UserAgent = String(patchedUA)
	}
	return b.NewContext(opts)
}

func (b *browserImpl) getPatchedUA() (string, error) {
	b.stealthUAOnce.Do(func() {
		ctx, err := b.NewContext()
		if err != nil {
			b.stealthUAErr = fmt.Errorf("stealth UA: could not create context: %w", err)
			return
		}
		page, err := ctx.NewPage()
		if err != nil {
			if closeErr := ctx.Close(); closeErr != nil {
				logger.Error("stealth UA: could not close context", "error", closeErr)
			}
			b.stealthUAErr = fmt.Errorf("stealth UA: could not create page: %w", err)
			return
		}
		rawUA, err := page.Evaluate("() => navigator.userAgent")
		if closeErr := ctx.Close(); closeErr != nil {
			logger.Error("stealth UA: could not close context", "error", closeErr)
		}
		if err != nil {
			b.stealthUAErr = fmt.Errorf("stealth UA: could not evaluate: %w", err)
			return
		}
		ua, ok := rawUA.(string)
		if !ok {
			b.stealthUAErr = fmt.Errorf("stealth UA: unexpected type %T", rawUA)
			return
		}
		b.stealthUA = PatchHeadlessUA(ua)
	})
	return b.stealthUA, b.stealthUAErr
}
