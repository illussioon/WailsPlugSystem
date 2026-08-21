package client_test

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/illussioon/WailsPlugSystem/client"
	"github.com/illussioon/WailsPlugSystem/plugin"
)

func TestClientFacadeLoadsPluginBuiltBySDK(t *testing.T) {
	pluginsDir := t.TempDir()
	definition := plugin.New("sdk.example", "SDK Example", "1.0.0").
		Priority(10).
		HTML().
		SetText("title", "#title", "from SDK", plugin.WithConflictKey("title"))
	if _, err := definition.Build(filepath.Join(pluginsDir, "sdk.example.plugs")); err != nil {
		t.Fatal(err)
	}
	clientInstance, err := client.New(client.Options{Directory: pluginsDir, StrictDependencies: true})
	if err != nil {
		t.Fatal(err)
	}
	if err := clientInstance.Reload(context.Background()); err != nil {
		t.Fatal(err)
	}
	result, err := clientInstance.Render(`<html><head></head><body><h1 id="title">original</h1></body></html>`)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(result.HTML, ">from SDK</h1>") {
		t.Fatalf("SDK client did not render plugin: %s", result.HTML)
	}
	if len(clientInstance.Packages()) != 1 {
		t.Fatalf("expected one active package, got %d", len(clientInstance.Packages()))
	}
}
