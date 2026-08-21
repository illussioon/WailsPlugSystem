package httpmiddleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	wailsplugs "github.com/illussioon/WailsPlugSystem"
)

type testLoader []wailsplugs.Package

func (t testLoader) Load(context.Context) ([]wailsplugs.Package, error) { return t, nil }

func TestMiddlewareRendersOnlyHTML(t *testing.T) {
	item := wailsplugs.Package{
		Manifest: wailsplugs.Manifest{FormatVersion: 1, ID: "middleware", Name: "middleware", Version: "1.0.0", APIVersion: "v1", Permissions: []wailsplugs.Permission{wailsplugs.PermissionHTML}},
		Patches:  []wailsplugs.Patch{{Kind: wailsplugs.PatchSetText, Selector: "#title", Value: "patched"}},
	}
	manager := wailsplugs.NewManager(wailsplugs.ManagerOptions{Loader: testLoader{item}})
	if err := manager.Reload(context.Background()); err != nil {
		t.Fatal(err)
	}
	base := http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path == "/api" {
			writer.Header().Set("Content-Type", "application/json")
			_, _ = writer.Write([]byte(`{"ok":true}`))
			return
		}
		writer.Header().Set("Content-Type", "text/html")
		_, _ = writer.Write([]byte(`<html><head></head><body><h1 id="title">original</h1></body></html>`))
	})
	handler := New(manager, base)
	htmlResponse := httptest.NewRecorder()
	handler.ServeHTTP(htmlResponse, httptest.NewRequest(http.MethodGet, "/", nil))
	if !strings.Contains(htmlResponse.Body.String(), ">patched</h1>") {
		t.Fatalf("HTML was not rendered: %s", htmlResponse.Body.String())
	}
	jsonResponse := httptest.NewRecorder()
	handler.ServeHTTP(jsonResponse, httptest.NewRequest(http.MethodGet, "/api", nil))
	if jsonResponse.Body.String() != `{"ok":true}` {
		t.Fatalf("API response was modified: %s", jsonResponse.Body.String())
	}
}
