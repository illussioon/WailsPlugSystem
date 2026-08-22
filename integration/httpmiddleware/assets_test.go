package httpmiddleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	wailsplugs "github.com/illussioon/WailsPlugSystem"
)

func TestExternalAssetRoute(t *testing.T) {
	item := wailsplugs.Package{
		Manifest: wailsplugs.Manifest{
			FormatVersion: wailsplugs.FormatVersion,
			ID:            "asset.example",
			Name:          "Asset Example",
			Version:       "1.0.0",
			APIVersion:    wailsplugs.APIVersion,
		},
		Assets: map[string][]byte{"assets/chunks/lazy.js": []byte("export const ready = true;")},
	}
	loader := consoleLoader{item}
	manager := wailsplugs.NewManager(wailsplugs.ManagerOptions{Loader: loader})
	if err := manager.Reload(context.Background()); err != nil {
		t.Fatal(err)
	}
	handler := New(manager, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))

	request := httptest.NewRequest(http.MethodGet, wailsplugs.AssetEndpointPrefix+"asset.example/chunks/lazy.js", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || response.Body.String() != "export const ready = true;" {
		t.Fatalf("unexpected asset response: status=%d body=%q", response.Code, response.Body.String())
	}
	if !strings.Contains(response.Header().Get("Content-Type"), "javascript") {
		t.Fatalf("unexpected asset content type: %q", response.Header().Get("Content-Type"))
	}

	headRequest := httptest.NewRequest(http.MethodHead, wailsplugs.AssetEndpointPrefix+"asset.example/chunks/lazy.js", nil)
	headResponse := httptest.NewRecorder()
	handler.ServeHTTP(headResponse, headRequest)
	if headResponse.Code != http.StatusOK || headResponse.Body.Len() != 0 {
		t.Fatalf("unexpected HEAD response: status=%d body=%q", headResponse.Code, headResponse.Body.String())
	}

	traversalRequest := httptest.NewRequest(http.MethodGet, wailsplugs.AssetEndpointPrefix+"asset.example/../manifest.json", nil)
	traversalResponse := httptest.NewRecorder()
	handler.ServeHTTP(traversalResponse, traversalRequest)
	if traversalResponse.Code != http.StatusNotFound {
		t.Fatalf("path traversal was not rejected: status=%d", traversalResponse.Code)
	}
}
