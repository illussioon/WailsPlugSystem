package plugin_test

import (
	"path/filepath"
	"testing"

	wailsplugs "github.com/illussioon/WailsPlugSystem"
	"github.com/illussioon/WailsPlugSystem/plugin"
)

func TestDefinitionBuildsAssetsAndPatches(t *testing.T) {
	path := filepath.Join(t.TempDir(), "authoring.plugs")
	definition := plugin.New("authoring.example", "Authoring Example", "1.2.3").
		Priority(25).
		HTML().
		DependsOn("base.example", "1.0.0").
		SetAttr("mark", "body", "data-plugin", "authoring").
		AddCSS("style", "theme.css", []byte("body { color: blue; }"), plugin.WithConflictKey("theme"), plugin.Optional()).
		AddJS("script", "app.js", []byte("window.authoring = true;"))
	if _, err := definition.Build(path); err != nil {
		t.Fatal(err)
	}
	item, err := wailsplugs.OpenPackage(path, wailsplugs.PackageOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if item.Manifest.ID != "authoring.example" || item.Manifest.Priority != 25 {
		t.Fatalf("unexpected manifest: %#v", item.Manifest)
	}
	if !item.Manifest.HasPermission(wailsplugs.PermissionCSS) || !item.Manifest.HasPermission(wailsplugs.PermissionJS) {
		t.Fatal("asset helpers did not add permissions")
	}
	if len(item.Patches) != 3 || len(item.Assets) != 2 {
		t.Fatalf("unexpected generated package: patches=%d assets=%d", len(item.Patches), len(item.Assets))
	}
}
