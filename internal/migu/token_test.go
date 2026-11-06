package migu

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMiguTokenFile(t *testing.T) {
	t.Setenv("MIGU_PACTOKEN", "")
	t.Setenv("FREEWAVE_MIGU_TOKEN", "")

	dir := t.TempDir()
	oldDir := miguTokenDir
	miguTokenDir = func() (string, error) { return dir, nil }
	t.Cleanup(func() { miguTokenDir = oldDir })

	if token := loadMiguToken(); token != "" {
		t.Fatalf("expected no token before saving, got %q", token)
	}
	if err := saveMiguToken("abc123"); err != nil {
		t.Fatalf("save token: %v", err)
	}
	path := filepath.Join(dir, "auth.json")
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat token file: %v", err)
	}
	if mode := info.Mode().Perm(); mode != 0o600 {
		t.Fatalf("token file mode = %o, want 600", mode)
	}
	if token := loadMiguToken(); token != "abc123" {
		t.Fatalf("load token = %q, want abc123", token)
	}
	if err := saveMiguToken(""); err != nil {
		t.Fatalf("clear token: %v", err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("token file still exists: %v", err)
	}
	if token := loadMiguToken(); token != "" {
		t.Fatalf("expected no token after clearing, got %q", token)
	}
}
