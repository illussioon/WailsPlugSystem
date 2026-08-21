package plugin_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	wailsplugs "github.com/illussioon/WailsPlugSystem"
	"github.com/illussioon/WailsPlugSystem/plugin"
)

func TestConsoleHelpersBuildCalls(t *testing.T) {
	path := filepath.Join(t.TempDir(), "console.plugs")
	definition := plugin.New("console.example", "Console Example", "1.0.0").
		Console("host-message", `Hello "World"`).
		ConsoleBrowser("browser-message", "Hello Browser")
	if _, err := definition.Build(path); err != nil {
		t.Fatal(err)
	}
	item, err := wailsplugs.OpenPackage(path, wailsplugs.PackageOptions{})
	if err != nil {
		t.Fatal(err)
	}
	joined := ""
	for _, data := range item.Assets {
		joined += string(data) + "\n"
	}
	for _, expected := range []string{"Wails.print.console(\"Hello \\\"World\\\"\")", "Wails.print.console.browser(\"Hello Browser\")"} {
		if !strings.Contains(joined, expected) {
			t.Fatalf("generated console asset missing %q: %s", expected, joined)
		}
	}
}

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

func TestDefinitionLifecycleMessages(t *testing.T) {
	path := filepath.Join(t.TempDir(), "lifecycle.plugs")
	definition := plugin.New("lifecycle.example", "Lifecycle Example", "1.0.0").
		OnLoad("plugin loaded").
		OnUnload("plugin unloaded")
	if _, err := definition.Build(path); err != nil {
		t.Fatal(err)
	}
	item, err := wailsplugs.OpenPackage(path, wailsplugs.PackageOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if item.Manifest.Lifecycle.Load != "plugin loaded" || item.Manifest.Lifecycle.Unload != "plugin unloaded" {
		t.Fatalf("unexpected lifecycle manifest: %#v", item.Manifest.Lifecycle)
	}
}

func TestFileBasedAuthoring(t *testing.T) {
	sourceDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(sourceDir, "panel.html"), []byte(`<section class="panel">Loaded from file</section>`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sourceDir, "main.css"), []byte(`.panel { color: red; }`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sourceDir, "main.js"), []byte(`console.log("file asset");`), 0644); err != nil {
		t.Fatal(err)
	}
	assetDir := filepath.Join(sourceDir, "static")
	if err := os.MkdirAll(filepath.Join(assetDir, "nested"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(assetDir, "nested", "icon.txt"), []byte("icon"), 0644); err != nil {
		t.Fatal(err)
	}

	output := filepath.Join(t.TempDir(), "file-based.plugs")
	definition := plugin.New("file.example", "File Example", "1.0.0").
		AppendHTMLFile("panel", "body", filepath.Join(sourceDir, "panel.html")).
		CSSFile("styles", "styles/main.css", filepath.Join(sourceDir, "main.css")).
		JSFile("script", "scripts/main.js", filepath.Join(sourceDir, "main.js")).
		AssetFile("assets/readme.txt", filepath.Join(sourceDir, "panel.html")).
		AssetsDir(assetDir)
	if _, err := definition.Build(output); err != nil {
		t.Fatal(err)
	}
	item, err := wailsplugs.OpenPackage(output, wailsplugs.PackageOptions{})
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"assets/styles/main.css", "assets/scripts/main.js", "assets/readme.txt", "assets/nested/icon.txt"} {
		if _, ok := item.Assets[path]; !ok {
			t.Fatalf("missing file-based asset %q; assets=%v", path, item.Assets)
		}
	}
	if len(item.Patches) != 3 || !strings.Contains(item.Patches[0].Value, "Loaded from file") {
		t.Fatalf("unexpected file-based patches: %#v", item.Patches)
	}
}

func TestFileBasedAuthoringReportsMissingFile(t *testing.T) {
	definition := plugin.New("missing.example", "Missing Example", "1.0.0").
		CSSFile("missing", "missing.css", filepath.Join(t.TempDir(), "missing.css"))
	if _, err := definition.Build(filepath.Join(t.TempDir(), "missing.plugs")); err == nil {
		t.Fatal("expected missing source file error")
	}
}
