package patchright

import (
	"os"
	"path/filepath"
	"testing"
)

// A partial/interrupted install can leave package/cli.js present but the bundled
// Node.js runtime missing. isUpToDateDriver must then report not-up-to-date (so
// DownloadDriver re-fetches everything) instead of hard-erroring on every run with
// "could not run driver: ... node[.exe]: no such file".
func TestIsUpToDateDriver_MissingManagedNodeReinstalls(t *testing.T) {
	// Ensure the managed-node path is exercised (no caller override).
	t.Setenv("PATCHRIGHT_NODEJS_PATH", "")
	t.Setenv("PATCHRIGHT_CLI_PATH", "")

	tmp := t.TempDir()
	pkgDir := filepath.Join(tmp, "package")
	if err := os.MkdirAll(pkgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// cli.js present...
	if err := os.WriteFile(filepath.Join(pkgDir, "cli.js"), []byte("// stub"), 0o644); err != nil {
		t.Fatal(err)
	}
	// ...but the bundled node executable was never written.

	d := &PatchrightDriver{Version: "1.60.0", options: &RunOptions{DriverDirectory: tmp}}

	up2date, err := d.isUpToDateDriver()
	if err != nil {
		t.Fatalf("partial install should not error, got: %v", err)
	}
	if up2date {
		t.Fatal("partial install must report up2date=false so the driver is re-downloaded")
	}
}

// A clean directory (no cli.js at all) is likewise not-up-to-date, with no error.
func TestIsUpToDateDriver_EmptyDirIsNotUpToDate(t *testing.T) {
	t.Setenv("PATCHRIGHT_NODEJS_PATH", "")
	t.Setenv("PATCHRIGHT_CLI_PATH", "")

	tmp := t.TempDir()
	d := &PatchrightDriver{Version: "1.60.0", options: &RunOptions{DriverDirectory: tmp}}

	up2date, err := d.isUpToDateDriver()
	if err != nil {
		t.Fatalf("empty dir should not error, got: %v", err)
	}
	if up2date {
		t.Fatal("empty dir must report up2date=false")
	}
}
