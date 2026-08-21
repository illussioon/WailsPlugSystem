package main

import (
	"context"
	"strings"
	"testing"

	"github.com/illussioon/WailsPlugSystem/client"
)

func TestIPPluginIsLoadedFromPluginsDirectory(t *testing.T) {
	plugins, err := client.New(client.Options{
		Directory:       "./plugins",
		AllowJavaScript: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := plugins.Reload(context.Background()); err != nil {
		t.Fatal(err)
	}
	result, err := plugins.Render(`<html><head></head><body><main><h1 id="app-title">Host</h1></main></body></html>`)
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{
		"data-wailsplugs-ip",
		"data-ip-refresh",
		"Ваш IP: определение...",
		"api64.ipify.org?format=json",
	} {
		if !strings.Contains(result.HTML, expected) {
			t.Fatalf("plugin output does not contain %q: %s", expected, result.HTML)
		}
	}
	if len(result.Plugins) != 1 || result.Plugins[0] != "example.ip" {
		t.Fatalf("unexpected active plugins: %#v", result.Plugins)
	}
}
