package httpmiddleware

import (
	"bytes"
	"net/http"
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
