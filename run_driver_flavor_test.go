package patchright

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

// makeNpmTgz builds a gzipped tar in npm layout (every entry under "package/").
func makeNpmTgz(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	for name, content := range files {
		hdr := &tar.Header{Name: "package/" + name, Mode: 0o644, Size: int64(len(content)), Typeflag: tar.TypeReg}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestTransformRunOptionsDriver(t *testing.T) {
	// Default flavor + dir.
	d, err := NewDriver(&RunOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if d.options.Driver != DriverPatchright {
		t.Errorf("default Driver = %q, want %q", d.options.Driver, DriverPatchright)
	}
	if filepath.Base(d.options.DriverDirectory) != "patchright-driver" {
		t.Errorf("default dir = %q, want .../patchright-driver", d.options.DriverDirectory)
	}

	// Playwright flavor gets its own default dir.
	d2, err := NewDriver(&RunOptions{Driver: DriverPlaywright})
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(d2.options.DriverDirectory) != "playwright-driver" {
		t.Errorf("playwright dir = %q, want .../playwright-driver", d2.options.DriverDirectory)
	}

	// Invalid flavor is rejected.
	if _, err := NewDriver(&RunOptions{Driver: "webkit"}); err == nil {
		t.Error("expected error for invalid Driver")
	}
}

func TestDownloadDriverFlavorSelectsPackages(t *testing.T) {
	cases := []struct {
		driver   string
		wantMain string
		wantCore string
	}{
		{DriverPatchright, "patchright", "patchright-core"},
		{DriverPlaywright, "playwright", "playwright-core"},
	}
	for _, tc := range cases {
		t.Run(tc.driver, func(t *testing.T) {
			const version = "1.60.0"
			mainTgz := makeNpmTgz(t, map[string]string{"cli.js": "console.log('cli')"})
			coreTgz := makeNpmTgz(t, map[string]string{"lib/index.js": "module.exports = {}"})

			var mu sync.Mutex
			var paths []string
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				paths = append(paths, r.URL.Path)
				mu.Unlock()
				switch r.URL.Path {
				case "/" + tc.wantMain + "/-/" + tc.wantMain + "-" + version + ".tgz":
					w.Write(mainTgz)
				case "/" + tc.wantCore + "/-/" + tc.wantCore + "-" + version + ".tgz":
					w.Write(coreTgz)
				default:
					http.NotFound(w, r)
				}
			}))
			defer srv.Close()

			dir := t.TempDir()
			d, err := NewDriver(&RunOptions{
				Driver:          tc.driver,
				Version:         version,
				NpmRegistry:     srv.URL,
				DriverDirectory: dir,
			})
			if err != nil {
				t.Fatal(err)
			}
			if err := d.downloadPatchrightPackage(); err != nil {
				t.Fatalf("download package: %v", err)
			}
			if err := d.downloadPatchrightCore(); err != nil {
				t.Fatalf("download core: %v", err)
			}

			// cli.js in package/, core in package/node_modules/<core>/.
			if _, err := os.Stat(filepath.Join(dir, "package", "cli.js")); err != nil {
				t.Errorf("cli.js missing: %v", err)
			}
			corePath := filepath.Join(dir, "package", "node_modules", tc.wantCore, "lib", "index.js")
			if _, err := os.Stat(corePath); err != nil {
				t.Errorf("core impl missing at %s: %v", corePath, err)
			}

			// The correct flavor's packages were requested, not the other's.
			mu.Lock()
			defer mu.Unlock()
			joined := "|" + join(paths, "|") + "|"
			if !contains(joined, "/"+tc.wantMain+"/-/") || !contains(joined, "/"+tc.wantCore+"/-/") {
				t.Errorf("did not request %s/%s packages; got %v", tc.wantMain, tc.wantCore, paths)
			}
			other := "patchright"
			if tc.driver == DriverPatchright {
				other = "playwright"
			}
			if contains(joined, "/"+other+"/-/") {
				t.Errorf("unexpectedly requested %s packages; got %v", other, paths)
			}
		})
	}
}

func join(s []string, sep string) string {
	out := ""
	for i, v := range s {
		if i > 0 {
			out += sep
		}
		out += v
	}
	return out
}

func contains(hay, needle string) bool {
	return bytes.Contains([]byte(hay), []byte(needle))
}
