package geoip

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDBFileUsableRejectsStub(t *testing.T) {
	path := filepath.Join(t.TempDir(), "geoip.db")
	if err := os.WriteFile(path, []byte("stub\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if dbFileUsable(path) {
		t.Fatal("stub database must not be treated as usable")
	}
}

func TestDownloadDBFileRejectsInvalidDatabaseWithoutReplacingExistingFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "geoip.mmdb")
	if err := os.WriteFile(path, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := downloadDBFile(path, "://bad-url"); err == nil {
		t.Fatal("invalid URL must fail")
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "old" {
		t.Fatalf("failed download must not replace existing file, got %q", string(got))
	}
}
