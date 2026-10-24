package web

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
)

func TestHandler(t *testing.T) {
	root := fstest.MapFS{
		"index.html":     {Data: []byte("<!doctype html><html><body>shell</body></html>")},
		"_app/style.css": {Data: []byte("body { color: red; }")},
		"_app/app.js":    {Data: []byte("console.log('hi');")},
	}
	handler := newHandler(root)

	tests := []struct {
		name     string
		path     string
		wantCode int
		wantType string
		wantBody string
	}{
		{"root serves shell", "/", 200, "text/html", "shell"},
		{"client-side route falls back", "/teams/abc/deep", 200, "text/html", "shell"},
		{"literal index.html", "/index.html", 200, "text/html", "shell"},
		{"asset served as file", "/_app/app.js", 200, "text/javascript", "console.log"},
		{"missing file falls back to shell", "/nope.txt", 200, "text/html", "shell"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if rec.Code != tt.wantCode {
				t.Fatalf("status = %d, want %d", rec.Code, tt.wantCode)
			}
			if got := rec.Header().Get("Content-Type"); !strings.HasPrefix(got, tt.wantType) {
				t.Fatalf("content-type = %q, want prefix %q", got, tt.wantType)
			}
			if !strings.Contains(rec.Body.String(), tt.wantBody) {
				t.Fatalf("body = %q, want contains %q", rec.Body.String(), tt.wantBody)
			}
		})
	}
}

// TestHandlerNoShell pins the client-less checkout behavior: with only a
// placeholder embedded, every non-asset path 404s rather than panicking.
func TestHandlerNoShell(t *testing.T) {
	handler := newHandler(fstest.MapFS{
		"_app/app.js": {Data: []byte("console.log('hi');")},
	})

	req := httptest.NewRequest(http.MethodGet, "/teams/x", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}
