package wailsplugs

import (
	"context"
	"strings"
	"testing"
)

type staticLoader []Package

func (s staticLoader) Load(context.Context) ([]Package, error) { return s, nil }

func testManifest(id string, priority int, permissions ...Permission) Manifest {
	return Manifest{FormatVersion: FormatVersion, ID: id, Name: id, Version: "1.0.0", APIVersion: APIVersion, Priority: priority, Permissions: permissions}
}

func TestPriorityConflictAndRollback(t *testing.T) {
	low := Package{Manifest: testManifest("low", 1, PermissionHTML), Patches: []Patch{{ID: "title", Kind: PatchSetText, Selector: "#title", Value: "low", ConflictKey: "title"}}}
	high := Package{Manifest: testManifest("high", 10, PermissionHTML), Patches: []Patch{{ID: "title", Kind: PatchSetText, Selector: "#title", Value: "high", ConflictKey: "title"}}}
	manager := NewManager(ManagerOptions{Loader: staticLoader{low, high}, StrictDependencies: true})
	if err := manager.Reload(context.Background()); err != nil {
		t.Fatal(err)
	}
	result, err := manager.Render(`<html><head></head><body><h1 id="title">original</h1></body></html>`)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(result.HTML, ">high</h1>") {
		t.Fatalf("high priority patch did not win: %s", result.HTML)
	}
	if !result.Decisions[0].Applied || result.Decisions[1].Applied {
		t.Fatalf("unexpected decisions: %#v", result.Decisions)
	}
	if !strings.Contains(result.Decisions[1].Reason, "lower priority") {
		t.Fatalf("missing conflict reason: %#v", result.Decisions[1])
	}
	manager.packages = map[string]Package{}
	rolledBack, err := manager.Render(`<html><head></head><body><h1 id="title">original</h1></body></html>`)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(rolledBack.HTML, ">original</h1>") {
		t.Fatalf("render did not roll back: %s", rolledBack.HTML)
	}
}

func TestSanitizerRemovesDangerousMarkup(t *testing.T) {
	item := Package{Manifest: testManifest("safe", 1, PermissionHTML), Patches: []Patch{{Kind: PatchAppendHTML, Selector: "body", Value: `<div onclick="evil()"><a href="javascript:alert(1)">safe</a><script>alert(1)</script></div>`}}}
	manager := NewManager(ManagerOptions{Loader: staticLoader{item}, StrictDependencies: true})
	if err := manager.Reload(context.Background()); err != nil {
		t.Fatal(err)
	}
	result, err := manager.Render(`<html><head></head><body></body></html>`)
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"onclick", "javascript:"} {
		if strings.Contains(strings.ToLower(result.HTML), forbidden) {
			t.Fatalf("sanitizer left %q in %s", forbidden, result.HTML)
		}
	}
	if strings.Contains(strings.ToLower(result.HTML), "<script") {
		t.Fatalf("sanitizer left a script while JavaScript is disabled: %s", result.HTML)
	}
	if !strings.Contains(result.HTML, ">safe</a>") {
		t.Fatalf("safe text missing: %s", result.HTML)
	}
}

func TestJavaScriptRequiresExplicitPolicy(t *testing.T) {
	item := Package{
		Manifest: testManifest("script", 1, PermissionJS),
		Patches:  []Patch{{Kind: PatchInjectJS, Asset: "assets/app.js"}},
		Assets:   map[string][]byte{"assets/app.js": []byte("window.__plugin_loaded = true;")},
	}
	manager := NewManager(ManagerOptions{Loader: staticLoader{item}})
	if err := manager.Reload(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, err := manager.Render(`<html><head></head><body></body></html>`); err == nil {
		t.Fatal("JavaScript was injected while policy was disabled")
	}
	manager = NewManager(ManagerOptions{Loader: staticLoader{item}, AllowJavaScript: true})
	if err := manager.Reload(context.Background()); err != nil {
		t.Fatal(err)
	}
	result, err := manager.Render(`<html><head></head><body></body></html>`)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(result.HTML, "__plugin_loaded") {
		t.Fatalf("JavaScript was not injected with explicit policy: %s", result.HTML)
	}
}

type sequenceLoader struct {
	packages []Package
}

func (s *sequenceLoader) Load(context.Context) ([]Package, error) {
	return append([]Package(nil), s.packages...), nil
}

func TestPluginLifecycleMessages(t *testing.T) {
	loader := &sequenceLoader{}
	var messages []ConsoleMessage
	manager := NewManager(ManagerOptions{
		Loader:          loader,
		AllowJavaScript: true,
		HostLogger: func(message ConsoleMessage) {
			messages = append(messages, message)
		},
	})
	old := Package{Manifest: testManifest("example", 1), SHA256: "old"}
	old.Manifest.Lifecycle = Lifecycle{Load: "old loaded", Unload: "old unloaded"}
	newItem := Package{Manifest: testManifest("example", 1), SHA256: "new"}
	newItem.Manifest.Lifecycle = Lifecycle{Load: "new loaded", Unload: "new unloaded"}

	loader.packages = []Package{old}
	if err := manager.Reload(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(messages) != 1 || messages[0].Message != "old loaded" || messages[0].Source != "plugin.load" {
		t.Fatalf("unexpected initial lifecycle messages: %#v", messages)
	}
	first, err := manager.Render(`<html><head></head><body></body></html>`)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(first.HTML, `Wails.plugin.print.load("old loaded")`) {
		t.Fatalf("load lifecycle script missing: %s", first.HTML)
	}
	second, err := manager.Render(`<html><head></head><body></body></html>`)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(second.HTML, "old loaded") {
		t.Fatalf("lifecycle event was injected more than once: %s", second.HTML)
	}

	loader.packages = []Package{newItem}
	if err := manager.Reload(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(messages) != 3 || messages[1].Message != "old unloaded" || messages[2].Message != "new loaded" {
		t.Fatalf("unexpected replacement lifecycle messages: %#v", messages)
	}
	replacement, err := manager.Render(`<html><head></head><body></body></html>`)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(replacement.HTML, `Wails.plugin.print.unload("old unloaded")`) || !strings.Contains(replacement.HTML, `Wails.plugin.print.load("new loaded")`) {
		t.Fatalf("replacement lifecycle scripts missing: %s", replacement.HTML)
	}

	loader.packages = nil
	if err := manager.Reload(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(messages) != 4 || messages[3].Message != "new unloaded" {
		t.Fatalf("unexpected final unload messages: %#v", messages)
	}
}
