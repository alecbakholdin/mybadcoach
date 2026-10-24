package web

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
)

func TestHandler(t *testing.T) {
	const immutable = "public, max-age=31536000, immutable"

	root := fstest.MapFS{
		"index.html":                          {Data: []byte("<!doctype html><html><body>shell</body></html>")},
		"_app/style.css":                      {Data: []byte("body { color: red; }")},
		"_app/app.js":                         {Data: []byte("console.log('hi');")},
		"_app/immutable/chunks/app.abc123.js": {Data: []byte("console.log('immutable');")},
		"fonts/sora-latin.woff2":              {Data: []byte("font-bytes")},
	}
	handler := newHandler(root)

	tests := []struct {
		name      string
		path      string
		wantType  string
		wantBody  string
		wantCache string
		wantCode  int
	}{
		{"root serves shell", "/", "text/html", "shell", "", 200},
		{"client-side route falls back", "/teams/abc/deep", "text/html", "shell", "", 200},
		{"literal index.html", "/index.html", "text/html", "shell", "", 200},
		{"asset served as file", "/_app/app.js", "text/javascript", "console.log", "", 200},
		{"immutable asset cached forever", "/_app/immutable/chunks/app.abc123.js", "text/javascript", "immutable", immutable, 200},
		{"font cached forever", "/fonts/sora-latin.woff2", "", "font-bytes", immutable, 200},
		{"missing file falls back to shell", "/nope.txt", "text/html", "shell", "", 200},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, http.NoBody)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if rec.Code != tt.wantCode {
				t.Fatalf("status = %d, want %d", rec.Code, tt.wantCode)
			}
			if tt.wantType != "" {
				if got := rec.Header().Get("Content-Type"); !strings.HasPrefix(got, tt.wantType) {
					t.Fatalf("content-type = %q, want prefix %q", got, tt.wantType)
				}
			}
			if got := rec.Header().Get("Cache-Control"); got != tt.wantCache {
				t.Fatalf("cache-control = %q, want %q", got, tt.wantCache)
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

	req := httptest.NewRequest(http.MethodGet, "/teams/x", http.NoBody)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}
