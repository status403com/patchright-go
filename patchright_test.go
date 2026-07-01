package patchright

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testHTML = `<!DOCTYPE html>
<html>
<head><title>Patchright Test Page</title></head>
<body>
  <h1 id="heading">Hello Patchright</h1>
  <p class="info">Test paragraph</p>
  <div data-testid="container">
    <span class="nested">Nested text</span>
  </div>

  <input id="text-input" type="text" placeholder="Type here" />
  <textarea id="textarea">Initial content</textarea>

  <button id="click-btn" onclick="document.getElementById('click-result').textContent='clicked'">Click Me</button>
  <div id="click-result"></div>

  <button id="dblclick-btn" ondblclick="document.getElementById('dbl-result').textContent='double-clicked'">Double Click</button>
  <div id="dbl-result"></div>

  <div id="hover-target" onmouseenter="this.textContent='hovered'" style="padding:10px;border:1px solid black">Hover me</div>

  <select id="select-box">
    <option value="a">Option A</option>
    <option value="b">Option B</option>
    <option value="c">Option C</option>
  </select>

  <input id="checkbox" type="checkbox" />

  <a id="link" href="https://example.com">Example Link</a>

  <div id="dynamic"></div>
  <script>
    setTimeout(function() {
      document.getElementById('dynamic').innerHTML = '<span id="delayed">Appeared after 500ms</span>';
    }, 500);
  </script>

  <form id="form" onsubmit="event.preventDefault(); document.getElementById('form-result').textContent='submitted'">
    <input name="email" type="email" />
    <input name="password" type="password" />
    <button type="submit">Submit</button>
  </form>
  <div id="form-result"></div>

  <ul id="list">
    <li>Item 1</li>
    <li>Item 2</li>
    <li>Item 3</li>
    <li>Item 4</li>
    <li>Item 5</li>
  </ul>
</body>
</html>`

func setupTestPage(t *testing.T) (Page, func()) {
	t.Helper()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, testHTML)
	}))

	pw, err := Run()
	require.NoError(t, err)

	browser, err := pw.Chromium.Launch(BrowserTypeLaunchOptions{
		Headless: Bool(true),
	})
	require.NoError(t, err)

	page, err := browser.NewStealthPage()
	require.NoError(t, err)

	_, err = page.Goto(ts.URL, PageGotoOptions{
		WaitUntil: WaitUntilStateLoad,
	})
	require.NoError(t, err)

	cleanup := func() {
		browser.Close()
		pw.Stop()
		ts.Close()
	}
	return page, cleanup
}

func TestCSSSelector(t *testing.T) {
	page, cleanup := setupTestPage(t)
	defer cleanup()

	text, err := page.Locator("#heading").TextContent()
	require.NoError(t, err)
	assert.Equal(t, "Hello Patchright", text)

	text, err = page.Locator(".info").TextContent()
	require.NoError(t, err)
	assert.Equal(t, "Test paragraph", text)

	text, err = page.Locator("[data-testid='container'] .nested").TextContent()
	require.NoError(t, err)
	assert.Equal(t, "Nested text", text)
}

func TestXPathSelector(t *testing.T) {
	page, cleanup := setupTestPage(t)
	defer cleanup()

	text, err := page.Locator("xpath=//h1[@id='heading']").TextContent()
	require.NoError(t, err)
	assert.Equal(t, "Hello Patchright", text)

	text, err = page.Locator("xpath=//p[contains(@class,'info')]").TextContent()
	require.NoError(t, err)
	assert.Equal(t, "Test paragraph", text)
}

func TestTextSelector(t *testing.T) {
	page, cleanup := setupTestPage(t)
	defer cleanup()

	visible, err := page.Locator("text=Click Me").IsVisible()
	require.NoError(t, err)
	assert.True(t, visible)
}

func TestEvaluateTypes(t *testing.T) {
	page, cleanup := setupTestPage(t)
	defer cleanup()

	result, err := page.Evaluate("() => 42")
	require.NoError(t, err)
	assert.Equal(t, 42, result)

	result, err = page.Evaluate("() => 'hello'")
	require.NoError(t, err)
	assert.Equal(t, "hello", result)

	result, err = page.Evaluate("() => true")
	require.NoError(t, err)
	assert.Equal(t, true, result)

	result, err = page.Evaluate("() => null")
	require.NoError(t, err)
	assert.Nil(t, result)

	result, err = page.Evaluate("() => [1, 2, 3]")
	require.NoError(t, err)
	arr, ok := result.([]any)
	require.True(t, ok)
	assert.Len(t, arr, 3)

	result, err = page.Evaluate("() => ({name: 'test', value: 123})")
	require.NoError(t, err)
	obj, ok := result.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "test", obj["name"])
}

func TestEvaluateWithArgument(t *testing.T) {
	page, cleanup := setupTestPage(t)
	defer cleanup()

	result, err := page.Evaluate("(x) => x * 3", 7)
	require.NoError(t, err)
	assert.Equal(t, 21, result)
}

func TestEvaluateHandle(t *testing.T) {
	page, cleanup := setupTestPage(t)
	defer cleanup()

	handle, err := page.EvaluateHandle("() => document.querySelector('#heading')")
	require.NoError(t, err)
	require.NotNil(t, handle)
}

func TestEvaluateDOM(t *testing.T) {
	page, cleanup := setupTestPage(t)
	defer cleanup()

	result, err := page.Evaluate("() => document.title")
	require.NoError(t, err)
	assert.Equal(t, "Patchright Test Page", result)

	result, err = page.Evaluate("() => document.querySelector('#heading').textContent")
	require.NoError(t, err)
	assert.Equal(t, "Hello Patchright", result)
}

func TestClickButton(t *testing.T) {
	page, cleanup := setupTestPage(t)
	defer cleanup()

	require.NoError(t, page.Locator("#click-btn").Click())
	text, err := page.Locator("#click-result").TextContent()
	require.NoError(t, err)
	assert.Equal(t, "clicked", text)
}

func TestDoubleClick(t *testing.T) {
	page, cleanup := setupTestPage(t)
	defer cleanup()

	require.NoError(t, page.Locator("#dblclick-btn").Dblclick())
	text, err := page.Locator("#dbl-result").TextContent()
	require.NoError(t, err)
	assert.Equal(t, "double-clicked", text)
}

func TestHover(t *testing.T) {
	page, cleanup := setupTestPage(t)
	defer cleanup()

	require.NoError(t, page.Locator("#hover-target").Hover())
	text, err := page.Locator("#hover-target").TextContent()
	require.NoError(t, err)
	assert.Equal(t, "hovered", text)
}

func TestFillInput(t *testing.T) {
	page, cleanup := setupTestPage(t)
	defer cleanup()

	loc := page.Locator("#text-input")
	require.NoError(t, loc.Fill("Hello World"))
	val, err := loc.InputValue()
	require.NoError(t, err)
	assert.Equal(t, "Hello World", val)
}

func TestPressSequentially(t *testing.T) {
	page, cleanup := setupTestPage(t)
	defer cleanup()

	loc := page.Locator("#text-input")
	require.NoError(t, loc.PressSequentially("typed"))
	val, err := loc.InputValue()
	require.NoError(t, err)
	assert.Equal(t, "typed", val)
}

func TestFillTextarea(t *testing.T) {
	page, cleanup := setupTestPage(t)
	defer cleanup()

	loc := page.Locator("#textarea")
	require.NoError(t, loc.Fill("New content\nWith newline"))
	val, err := loc.InputValue()
	require.NoError(t, err)
	assert.Equal(t, "New content\nWith newline", val)
}

func TestSelectDropdown(t *testing.T) {
	page, cleanup := setupTestPage(t)
	defer cleanup()

	vals, err := page.Locator("#select-box").SelectOption(SelectOptionValues{
		Values: StringSlice("b"),
	})
	require.NoError(t, err)
	assert.Equal(t, []string{"b"}, vals)
}

func TestCheckbox(t *testing.T) {
	page, cleanup := setupTestPage(t)
	defer cleanup()

	loc := page.Locator("#checkbox")
	require.NoError(t, loc.Check())
	checked, err := loc.IsChecked()
	require.NoError(t, err)
	assert.True(t, checked)

	require.NoError(t, loc.Uncheck())
	checked, err = loc.IsChecked()
	require.NoError(t, err)
	assert.False(t, checked)
}

func TestKeyboardSubmitForm(t *testing.T) {
	page, cleanup := setupTestPage(t)
	defer cleanup()

	page.Locator("input[name='email']").Fill("test@test.com")
	page.Locator("input[name='password']").Fill("secret123")
	page.Locator("input[name='password']").Press("Enter")
	time.Sleep(200 * time.Millisecond)

	text, err := page.Locator("#form-result").TextContent()
	require.NoError(t, err)
	assert.Equal(t, "submitted", text)
}

func TestWaitForSelector(t *testing.T) {
	page, cleanup := setupTestPage(t)
	defer cleanup()

	el, err := page.WaitForSelector("#delayed", PageWaitForSelectorOptions{
		Timeout: Float(5000),
	})
	require.NoError(t, err)
	text, err := el.TextContent()
	require.NoError(t, err)
	assert.Equal(t, "Appeared after 500ms", text)
}

func TestLocatorCount(t *testing.T) {
	page, cleanup := setupTestPage(t)
	defer cleanup()

	count, err := page.Locator("#list li").Count()
	require.NoError(t, err)
	assert.Equal(t, 5, count)
}

func TestLocatorNth(t *testing.T) {
	page, cleanup := setupTestPage(t)
	defer cleanup()

	text, err := page.Locator("#list li").Nth(2).TextContent()
	require.NoError(t, err)
	assert.Equal(t, "Item 3", text)
}

func TestLocatorAll(t *testing.T) {
	page, cleanup := setupTestPage(t)
	defer cleanup()

	items, err := page.Locator("#list li").All()
	require.NoError(t, err)
	assert.Len(t, items, 5)
}

func TestLocatorGetAttribute(t *testing.T) {
	page, cleanup := setupTestPage(t)
	defer cleanup()

	val, err := page.Locator("#link").GetAttribute("href")
	require.NoError(t, err)
	assert.Equal(t, "https://example.com", val)
}

func TestLocatorVisibleEnabled(t *testing.T) {
	page, cleanup := setupTestPage(t)
	defer cleanup()

	vis, err := page.Locator("#heading").IsVisible()
	require.NoError(t, err)
	assert.True(t, vis)

	en, err := page.Locator("#click-btn").IsEnabled()
	require.NoError(t, err)
	assert.True(t, en)
}

func TestPageTitle(t *testing.T) {
	page, cleanup := setupTestPage(t)
	defer cleanup()

	title, err := page.Title()
	require.NoError(t, err)
	assert.Equal(t, "Patchright Test Page", title)
}

func TestPageContent(t *testing.T) {
	page, cleanup := setupTestPage(t)
	defer cleanup()

	content, err := page.Content()
	require.NoError(t, err)
	assert.Contains(t, content, "Hello Patchright")
}

func TestPageScreenshot(t *testing.T) {
	page, cleanup := setupTestPage(t)
	defer cleanup()

	data, err := page.Screenshot()
	require.NoError(t, err)
	assert.Greater(t, len(data), 1000)
}

func TestMainFrameEval(t *testing.T) {
	page, cleanup := setupTestPage(t)
	defer cleanup()

	result, err := page.MainFrame().Evaluate("() => document.title")
	require.NoError(t, err)
	assert.Equal(t, "Patchright Test Page", result)
}

func TestNavigatorWebdriverFalse(t *testing.T) {
	page, cleanup := setupTestPage(t)
	defer cleanup()

	result, err := page.Evaluate("() => navigator.webdriver")
	require.NoError(t, err)
	assert.Equal(t, false, result)
}

func TestStealthUAPatched(t *testing.T) {
	page, cleanup := setupTestPage(t)
	defer cleanup()

	result, err := page.Evaluate("() => navigator.userAgent")
	require.NoError(t, err)
	ua := result.(string)
	assert.NotContains(t, ua, "HeadlessChrome")
	assert.Contains(t, ua, ".0.0.0")
}

func TestPatchHeadlessUA(t *testing.T) {
	cases := []struct {
		input    string
		expected string
	}{
		{
			"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) HeadlessChrome/149.0.7827.55 Safari/537.36",
			"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/149.0.0.0 Safari/537.36",
		},
		{
			"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) HeadlessChrome/120.0.6099.109 Safari/537.36",
			"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
		},
		{
			"Mozilla/5.0 Chrome/100.0.0.0 Safari/537.36",
			"Mozilla/5.0 Chrome/100.0.0.0 Safari/537.36",
		},
	}
	for _, tc := range cases {
		result := PatchHeadlessUA(tc.input)
		assert.Equal(t, tc.expected, result, "input: %s", tc.input)
	}
}
