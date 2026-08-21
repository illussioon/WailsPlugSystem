package httpmiddleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	wailsplugs "github.com/illussioon/WailsPlugSystem"
)

type consoleLoader []wailsplugs.Package

func (l consoleLoader) Load(context.Context) ([]wailsplugs.Package, error) { return l, nil }

func TestBrowserConsoleMessageReachesHostLogger(t *testing.T) {
	var received []wailsplugs.ConsoleMessage
	manager := wailsplugs.NewManager(wailsplugs.ManagerOptions{
		Loader: consoleLoader{},
		HostLogger: func(message wailsplugs.ConsoleMessage) {
			received = append(received, message)
		},
	})
	if err := manager.Reload(context.Background()); err != nil {
		t.Fatal(err)
	}
	base := http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "text/html")
		_, _ = writer.Write([]byte(`<html><head></head><body><main>host</main></body></html>`))
	})
	handler := New(manager, base)
	request := httptest.NewRequest(http.MethodPost, wailsplugs.ConsoleEndpoint, strings.NewReader(`{"message":"Hello World","args":[42]}`))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusNoContent {
		t.Fatalf("unexpected status: %d %s", response.Code, response.Body.String())
	}
	if len(received) != 1 || received[0].Message != "Hello World" || received[0].Source != "browser" {
		t.Fatalf("unexpected console messages: %#v", received)
	}
	if len(received[0].Args) != 1 {
		t.Fatalf("expected one argument, got %#v", received[0].Args)
	}
}

func TestConsoleBridgeIsInjectedIntoHTML(t *testing.T) {
	manager := wailsplugs.NewManager(wailsplugs.ManagerOptions{Loader: consoleLoader{}, AllowJavaScript: true})
	if err := manager.Reload(context.Background()); err != nil {
		t.Fatal(err)
	}
	result, err := manager.Render(`<html><head></head><body></body></html>`)
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{"window.Wails", "print.console", "print.console.browser", wailsplugs.ConsoleEndpoint} {
		if !strings.Contains(result.HTML, expected) {
			t.Fatalf("console bridge missing %q: %s", expected, result.HTML)
		}
	}
}
