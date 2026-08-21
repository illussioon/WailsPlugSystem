package pack_test

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	wailsplugs "github.com/illussioon/WailsPlugSystem"
	"github.com/illussioon/WailsPlugSystem/pack"
)

func TestBuildAndOpenPackage(t *testing.T) {
	root := t.TempDir()
	assets := filepath.Join(root, "assets")
	if err := os.MkdirAll(assets, 0755); err != nil {
		t.Fatal(err)
	}
	css := []byte("body { color: red; }")
	if err := os.WriteFile(filepath.Join(assets, "style.css"), css, 0644); err != nil {
		t.Fatal(err)
	}
	manifest := wailsplugs.Manifest{FormatVersion: 1, ID: "demo", Name: "demo", Version: "1.0.0", APIVersion: "v1", Priority: 5, Permissions: []wailsplugs.Permission{wailsplugs.PermissionHTML, wailsplugs.PermissionCSS}}
	manifestBytes, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "manifest.json"), manifestBytes, 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "patches.json"), []byte(`[{"id":"css","kind":"inject_css","asset":"assets/style.css"}]`), 0644); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "demo.plugs")
	if _, err := pack.Build(pack.Options{InputDir: root, Output: path}); err != nil {
		t.Fatal(err)
	}
	item, err := wailsplugs.OpenPackage(path, wailsplugs.PackageOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if item.Manifest.ID != "demo" || item.SHA256 == "" {
		t.Fatalf("bad package: %#v", item)
	}
	sum := sha256.Sum256(css)
	if got := item.Manifest.Files[0].SHA256; got != hex.EncodeToString(sum[:]) {
		t.Fatalf("bad asset hash: %s", got)
	}
}
