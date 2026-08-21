package loader

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	wailsplugs "github.com/illussioon/WailsPlugSystem"
	"github.com/illussioon/WailsPlugSystem/pack"
)

func TestSHA256Allowlist(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "assets"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "assets", "main.js"), []byte("console.log('ok')"), 0644); err != nil {
		t.Fatal(err)
	}
	manifest := `{"format_version":1,"id":"loader-test","name":"loader-test","version":"1.0.0","api_version":"v1","permissions":["js"]}`
	if err := os.WriteFile(filepath.Join(root, "manifest.json"), []byte(manifest), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "patches.json"), []byte(`[]`), 0644); err != nil {
		t.Fatal(err)
	}
	packagePath := filepath.Join(root, "loader-test.plugs")
	if _, err := pack.Build(pack.Options{InputDir: root, Output: packagePath}); err != nil {
		t.Fatal(err)
	}
	item, err := wailsplugs.OpenPackage(packagePath, wailsplugs.PackageOptions{})
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := (SHA256Allowlist{Dir: root, SHA256: []string{item.SHA256}}).Load(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded) != 1 {
		t.Fatalf("got %d loaded packages", len(loaded))
	}
	loaded, err = (SHA256Allowlist{Dir: root, SHA256: []string{"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}}).Load(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded) != 0 {
		t.Fatalf("unknown hash loaded %d packages", len(loaded))
	}
}
