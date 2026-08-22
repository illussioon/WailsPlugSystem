package httpmiddleware

import (
	"bytes"
	"encoding/json"
	"io"
	"mime"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"

	wailsplugs "github.com/illussioon/WailsPlugSystem"
)

type Middleware struct {
	Manager *wailsplugs.Manager
	Next    http.Handler
}

func New(manager *wailsplugs.Manager, next http.Handler) http.Handler {
	return Middleware{Manager: manager, Next: next}
}

func (m Middleware) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	if strings.HasPrefix(request.URL.Path, wailsplugs.AssetEndpointPrefix) {
		m.handleAsset(writer, request)
		return
	}
	if request.URL.Path == wailsplugs.ConsoleEndpoint {
		m.handleConsole(writer, request)
		return
	}
	if m.Manager == nil || m.Next == nil {
		if m.Next != nil {
			m.Next.ServeHTTP(writer, request)
		}
		return
	}
	capture := &responseCapture{header: make(http.Header), status: http.StatusOK}
	m.Next.ServeHTTP(capture, request)
	contentType := capture.header.Get("Content-Type")
	isHTML := strings.Contains(strings.ToLower(contentType), "text/html") || strings.HasSuffix(strings.ToLower(request.URL.Path), ".html") || request.URL.Path == "/"
	if !isHTML || capture.status == http.StatusNoContent || request.Method == http.MethodHead {
		copyResponse(writer, capture)
		return
	}
	result, err := m.Manager.Render(capture.body.String())
	if err != nil {
		http.Error(writer, "plugin render failed", http.StatusInternalServerError)
		return
	}
	capture.body.Reset()
	capture.body.WriteString(result.HTML)
	capture.header.Del("Content-Length")
	copyResponse(writer, capture)
}

func (m Middleware) handleAsset(writer http.ResponseWriter, request *http.Request) {
	if m.Manager == nil || (request.Method != http.MethodGet && request.Method != http.MethodHead) {
		writer.WriteHeader(http.StatusNotFound)
		return
	}
	rest := strings.TrimPrefix(request.URL.Path, wailsplugs.AssetEndpointPrefix)
	separator := strings.IndexByte(rest, '/')
	if separator <= 0 || separator == len(rest)-1 {
		writer.WriteHeader(http.StatusNotFound)
		return
	}
	pluginID, err := url.PathUnescape(rest[:separator])
	if err != nil || pluginID == "" {
		writer.WriteHeader(http.StatusNotFound)
		return
	}
	assetPath, err := url.PathUnescape(rest[separator+1:])
	if err != nil || strings.Contains(assetPath, "\\") {
		writer.WriteHeader(http.StatusNotFound)
		return
	}
	data, ok := m.Manager.Asset(pluginID, assetPath)
	if !ok {
		writer.WriteHeader(http.StatusNotFound)
		return
	}
	contentType := mime.TypeByExtension(filepath.Ext(assetPath))
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	writer.Header().Set("Content-Type", contentType)
	writer.Header().Set("Cache-Control", "no-cache")
	writer.WriteHeader(http.StatusOK)
	if request.Method != http.MethodHead {
		_, _ = writer.Write(data)
	}
}

func (m Middleware) handleConsole(writer http.ResponseWriter, request *http.Request) {
	if m.Manager == nil || request.Method != http.MethodPost {
		writer.WriteHeader(http.StatusNotFound)
		return
	}
	var message wailsplugs.ConsoleMessage
	decoder := json.NewDecoder(io.LimitReader(request.Body, 1<<20))
	if err := decoder.Decode(&message); err != nil {
		http.Error(writer, "invalid console message", http.StatusBadRequest)
		return
	}
	message.Source = "browser"
	m.Manager.LogConsole(message)
	writer.WriteHeader(http.StatusNoContent)
}

type responseCapture struct {
	header http.Header
	body   bytes.Buffer
	status int
}

func (r *responseCapture) Header() http.Header            { return r.header }
func (r *responseCapture) WriteHeader(status int)         { r.status = status }
func (r *responseCapture) Write(data []byte) (int, error) { return r.body.Write(data) }

func copyResponse(writer http.ResponseWriter, capture *responseCapture) {
	for key, values := range capture.header {
		for _, value := range values {
			writer.Header().Add(key, value)
		}
	}
	writer.WriteHeader(capture.status)
	_, _ = writer.Write(capture.body.Bytes())
}
