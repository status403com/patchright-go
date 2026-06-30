// Package patchright is a library to automate Chromium with
// a single API. Patchright is a patched version of Playwright built to evade
// bot detection while remaining capable, reliable and fast.
package patchright

// DeviceDescriptor represents a single device
type DeviceDescriptor struct {
	UserAgent          string  `json:"userAgent"`
	Viewport           *Size   `json:"viewport"`
	Screen             *Size   `json:"screen"`
	DeviceScaleFactor  float64 `json:"deviceScaleFactor"`
	IsMobile           bool    `json:"isMobile"`
	HasTouch           bool    `json:"hasTouch"`
	DefaultBrowserType string  `json:"defaultBrowserType"`
}

// Patchright represents a Patchright instance
type Patchright struct {
	channelOwner
	Selectors Selectors
	Chromium  BrowserType
	Firefox   BrowserType
	WebKit    BrowserType
	Request   APIRequest
	Devices   map[string]*DeviceDescriptor
}

// Stop stops the Patchright instance
func (p *Patchright) Stop() error {
	return p.connection.Stop()
}

// Pid returns the process ID of the Patchright driver process, or 0 if not available
func (p *Patchright) Pid() int {
	if pt, ok := p.connection.transport.(*pipeTransport); ok {
		if pt.process != nil {
			return pt.process.Pid
		}
	}
	return 0
}

func (p *Patchright) setSelectors(selectors Selectors) {
	if p.initializer["selectors"] != nil {
		selectorsOwner := fromChannel(p.initializer["selectors"]).(*selectorsOwnerImpl)
		p.Selectors.(*selectorsImpl).removeChannel(selectorsOwner)
		p.Selectors = selectors
		p.Selectors.(*selectorsImpl).addChannel(selectorsOwner)
	} else {
		p.Selectors = selectors
	}
}

func newPatchright(parent *channelOwner, objectType string, guid string, initializer map[string]any) *Patchright {
	pw := &Patchright{
		Selectors: newSelectorsImpl(),
		Chromium:  fromChannel(initializer["chromium"]).(*browserTypeImpl),
		Firefox:   fromChannel(initializer["firefox"]).(*browserTypeImpl),
		WebKit:    fromChannel(initializer["webkit"]).(*browserTypeImpl),
		Devices:   make(map[string]*DeviceDescriptor),
	}
	pw.createChannelOwner(pw, parent, objectType, guid, initializer)
	pw.Request = newApiRequestImpl(pw)
	pw.Chromium.(*browserTypeImpl).patchright = pw
	pw.Firefox.(*browserTypeImpl).patchright = pw
	pw.WebKit.(*browserTypeImpl).patchright = pw
	if initializer["selectors"] != nil {
		selectorsOwner := fromChannel(initializer["selectors"]).(*selectorsOwnerImpl)
		pw.Selectors.(*selectorsImpl).addChannel(selectorsOwner)
		pw.connection.afterClose = func() {
			pw.Selectors.(*selectorsImpl).removeChannel(selectorsOwner)
		}
	}
	if pw.connection.localUtils != nil {
		pw.Devices = pw.connection.localUtils.Devices
	}
	return pw
}

//go:generate bash scripts/generate-api.sh
