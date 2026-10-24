// Package web serves the embedded SvelteKit SPA build.
package web

import (
	"embed"
	"io/fs"
	"net/http"
	"strings"
)

// static holds the built SvelteKit client. A source checkout contains only a
// placeholder (static/.gitkeep); scripts/copy-client-build.sh populates it with
// the real `npm run build:client` output before a production Go build.
//
//go:embed all:static
var static embed.FS

// Handler serves the embedded SPA build.
func Handler() http.Handler {
	root, err := fs.Sub(static, "static")
	if err != nil {
		panic(err) // unreachable: the embed pattern always resolves
	}
	return newHandler(root)
}

// newHandler serves real files as-is and falls back to index.html for any
// path that names no file, so client-side routes deep-link correctly.
func newHandler(root fs.FS) http.Handler {
	fileServer := http.FileServer(http.FS(root))
	// The SPA shell, read once. Absent in a client-less checkout.
	index, _ := fs.ReadFile(root, "index.html")

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upath := strings.TrimPrefix(r.URL.Path, "/")

		// "/" and a literal "/index.html" are the shell; so is any path that
		// names no real file (a client-side route). Real files — the _app
		// assets — pass through to the file server untouched.
		if upath == "" || upath == "index.html" {
			serveShell(w, index)
			return
		}
		if _, openErr := root.Open(upath); openErr != nil {
			serveShell(w, index)
			return
		}
		fileServer.ServeHTTP(w, r)
	})
}

// serveShell writes the SPA entry document, or 404 when the client has not
// been built into the binary.
func serveShell(w http.ResponseWriter, index []byte) {
	if len(index) == 0 {
		http.Error(w, "SPA not built", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(index)
}
