package patchright

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunOptionsRedirectStderr(t *testing.T) {
	r, w := io.Pipe()
	var output string
	wg := &sync.WaitGroup{}
	readIOAsyncTilEOF(t, r, wg, &output)

	driverPath := t.TempDir()
	options := &RunOptions{
		Stderr:          w,
		DriverDirectory: driverPath,
		Browsers:        []string{},
		Verbose:         true,
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
	}))
	defer ts.Close()

	t.Setenv("PATCHRIGHT_NPM_REGISTRY", ts.URL)
	driver, err := NewDriver(options)
	require.NoError(t, err)
	err = driver.Install()
	require.Error(t, err)
	require.NoError(t, w.Close())
	wg.Wait()

	assert.Contains(t, output, "Downloading driver")
	require.Contains(t, output, fmt.Sprintf("path=%s", driverPath))
}

func TestRunOptions_OnlyInstallShell(t *testing.T) {
	if getBrowserName() != "chromium" {
		t.Skip("chromium only")
		return
	}

	r, w := io.Pipe()
	var output string
	wg := &sync.WaitGroup{}
	readIOAsyncTilEOF(t, r, wg, &output)

	driverPath := t.TempDir()
	driver, err := NewDriver(&RunOptions{
		Stdout:           w,
		DriverDirectory:  driverPath,
		Browsers:         []string{getBrowserName()},
		Verbose:          true,
		OnlyInstallShell: true,
		DryRun:           true,
	})
	require.NoError(t, err)
	browserPath := t.TempDir()

	t.Setenv("PATCHRIGHT_BROWSERS_PATH", browserPath)

	err = driver.Install()
	require.NoError(t, err)
	require.NoError(t, w.Close())
	wg.Wait()

	assert.Contains(t, output, "chromium-headless-shell")
	assert.NotContains(t, output, "Chrome for Testing")
}

func TestDriverInstall(t *testing.T) {
	driverPath := t.TempDir()
	driver, err := NewDriver(&RunOptions{
		DriverDirectory: driverPath,
		Browsers:        []string{getBrowserName()},
		Verbose:         true,
	})
	if err != nil {
		t.Fatalf("could not start driver: %v", err)
	}
	browserPath := t.TempDir()
	err = os.Setenv("PATCHRIGHT_BROWSERS_PATH", browserPath)
	if err != nil {
		t.Fatalf("could not set PATCHRIGHT_BROWSERS_PATH: %v", err)
	}
	defer os.Unsetenv("PATCHRIGHT_BROWSERS_PATH") //nolint:errcheck
	err = driver.Install()
	if err != nil {
		t.Fatalf("could not install driver: %v", err)
	}
	err = driver.Uninstall()
	if err != nil {
		t.Fatalf("could not uninstall driver: %v", err)
	}
}

func TestNpmRegistryEnv(t *testing.T) {
	driverPath := t.TempDir()
	driver, err := NewDriver(&RunOptions{
		DriverDirectory:     driverPath,
		SkipInstallBrowsers: true,
	})
	if err != nil {
		t.Fatalf("could not start driver: %v", err)
	}
	uri := ""
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		uri = r.URL.String()
		w.WriteHeader(404)
	}))
	defer ts.Close()

	t.Setenv("PATCHRIGHT_NPM_REGISTRY", ts.URL)
	err = driver.Install()
	if err == nil || !strings.Contains(err.Error(), "404 Not Found") || !strings.Contains(uri, "patchright") {
		t.Fatalf("PATCHRIGHT_NPM_REGISTRY does not work: %v", err)
	}
}

func TestNodePlatformSuffix(t *testing.T) {
	suffix, err := nodePlatformSuffix()
	switch runtime.GOARCH {
	case "amd64", "arm64":
		require.NoError(t, err)
		assert.NotEmpty(t, suffix)
	default:
		// e.g. linux/arm has no prebuilt Node.js binary on nodejs.org.
		require.Error(t, err)
		assert.Contains(t, err.Error(), "PATCHRIGHT_NODEJS_PATH")
	}
}

func TestPatchDriverBundleAllowsMissingPageErrorLocation(t *testing.T) {
	driverPath := t.TempDir()
	bundlePath := filepath.Join(driverPath, "package", "lib", "coreBundle.js")
	require.NoError(t, os.MkdirAll(filepath.Dir(bundlePath), 0o755))
	require.NoError(t, os.WriteFile(bundlePath, []byte(`location:{
url:pageError.location.url,
line: pageError.location.lineNumber,
column:    pageError.location.columnNumber
}`), 0o644))

	driver, err := NewDriver(&RunOptions{DriverDirectory: driverPath})
	require.NoError(t, err)
	require.NoError(t, driver.patchDriverBundle())
	require.NoError(t, driver.patchDriverBundle())

	data, err := os.ReadFile(bundlePath)
	require.NoError(t, err)
	require.Contains(t, string(data), `pageError.location?.url || ""`)
	require.Contains(t, string(data), `pageError.location?.lineNumber || 0`)
	require.Contains(t, string(data), `pageError.location?.columnNumber || 0`)
}

func TestShouldNotHangWhenPlaywrightUnexpectedExit(t *testing.T) {
	if getBrowserName() != "chromium" {
		t.Skip("chromium only")
		return
	}

	pw, err := Run()
	require.NoError(t, err)
	defer func() {
		_ = pw.Stop()
	}()
	browser, err := pw.Chromium.Launch()
	require.NoError(t, err)
	context, err := browser.NewContext()
	require.NoError(t, err)

	// Get the process ID directly from Playwright
	pid := pw.Pid()
	require.NotZero(t, pid, "Playwright process PID should not be zero")

	// Kill the process
	err = killProcessByPid(pid)
	require.NoError(t, err)

	_, err = context.NewPage()
	require.Error(t, err)
}

func TestGetNodeExecutable(t *testing.T) {
	opts := &RunOptions{DriverDirectory: "testDirectory"}

	// When PATCHRIGHT_NODEJS_PATH is set, use that path.
	t.Setenv("PATCHRIGHT_NODEJS_PATH", "envDir/node.exe")
	executable := getNodeExecutable(opts)
	assert.Equal(t, "envDir/node.exe", executable)

	require.NoError(t, os.Unsetenv("PATCHRIGHT_NODEJS_PATH"))
	executable = getNodeExecutable(opts)
	assert.Contains(t, executable, "testDirectory")

	// When NodeJSPath option is set, prefer it over env.
	opts.NodeJSPath = "/opt/node"
	executable = getNodeExecutable(opts)
	assert.Equal(t, "/opt/node", executable)
}

func TestGetDriverCliJs(t *testing.T) {
	opts := &RunOptions{DriverDirectory: "testDirectory"}

	// When PATCHRIGHT_CLI_PATH is set, use that path directly.
	t.Setenv("PATCHRIGHT_CLI_PATH", "/custom/cli.js")
	assert.Equal(t, "/custom/cli.js", getDriverCliJs(opts))

	// Otherwise fall back to the assumed <DriverDirectory>/package/cli.js layout.
	require.NoError(t, os.Unsetenv("PATCHRIGHT_CLI_PATH"))
	cliJs := getDriverCliJs(opts)
	assert.Contains(t, cliJs, "testDirectory")
	assert.Contains(t, cliJs, "cli.js")

	// When CLIPath option is set, prefer it over env.
	opts.CLIPath = "/opt/cli.js"
	assert.Equal(t, "/opt/cli.js", getDriverCliJs(opts))
}

func killProcessByPid(pid int) error {
	process, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	if err := process.Kill(); err != nil {
		return err
	}
	return nil
}

func getBrowserName() string {
	browserName, hasEnv := os.LookupEnv("BROWSER")
	if hasEnv {
		return browserName
	}
	return "chromium"
}

func readIOAsyncTilEOF(t *testing.T, r *io.PipeReader, wg *sync.WaitGroup, output *string) {
	t.Helper()
	wg.Add(1)
	go func() {
		defer wg.Done()
		buf := bufio.NewReader(r)
		for {
			line, _, err := buf.ReadLine()
			if err == io.EOF {
				break
			}
			*output += string(line)
		}
		_ = r.Close()
	}()
}
