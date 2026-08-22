package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/illussioon/WailsPlugSystem/client"
)

func TestMultiPluginDemoIntegration(t *testing.T) {
	plugins, err := client.New(client.Options{
		Directory:          "./plugins",
		StrictDependencies: true,
		AllowJavaScript:    true,
		AllowRootReplace:   false,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := plugins.Reload(context.Background()); err != nil {
		t.Fatal(err)
	}
	if got := len(plugins.Packages()); got != 3 {
		t.Fatalf("loaded %d plugins, want 3", got)
	}

	result, err := plugins.Render(`<!doctype html><html><head></head><body>
<header id="host-header"><div id="plugin-header-slot"></div></header>
<aside id="host-sidebar"><nav id="plugin-leftmenu-slot"></nav></aside>
<main><div id="plugin-tester-slot"></div><span id="host-status">waiting</span></main>
</body></html>`)
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{
		"header-plugin",
		"leftmenu-plugin",
		"tester-panel",
		"data-wailsplugs-host-css=\"true\"",
		"/__wailsplugs/assets/tester/tester.js",
		"Wails.plugin.print.load",
	} {
		if !strings.Contains(result.HTML, expected) {
			t.Fatalf("rendered HTML does not contain %q: %s", expected, result.HTML)
		}
	}
	if len(result.Decisions) == 0 {
		t.Fatal("expected patch decisions")
	}
	chunk, ok := plugins.Manager().Asset("tester", "chunks/tester-chunk.js")
	if !ok || !strings.Contains(string(chunk), "chunkCheck") {
		t.Fatalf("dynamic chunk is not available through manager asset lookup: %q", chunk)
	}
	for _, decision := range result.Decisions {
		if decision.ConflictKey == "demo:host-status" && decision.Applied {
			if decision.PluginID != "header" {
				t.Fatalf("priority conflict winner is %q, want header", decision.PluginID)
			}
		}
	}

	next := http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "text/html")
		_, _ = writer.Write([]byte(`<!doctype html><html><head></head><body><main id="plugin-tester-slot"></main></body></html>`))
	})
	server := httptest.NewServer(plugins.Handler(next))
	defer server.Close()
	response, err := http.Get(server.URL + "/__wailsplugs/assets/tester/tester-data.json")
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("asset route returned %s", response.Status)
	}

	app := &App{plugins: plugins}
	if got := len(app.GetPluginReport()); got != 3 {
		t.Fatalf("plugin report has %d entries, want 3", got)
	}
}
