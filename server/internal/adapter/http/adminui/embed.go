// Package adminui serves the admin single-page app.
//
// Embedded in the binary with go:embed rather than read off disk: the service
// already ships as one file and a deployment that has to keep a web root in
// step with a binary is a deployment that will eventually be out of step.
//
// Every byte here is static. All authority lives in the API — this package
// serves the same HTML to an anonymous visitor as to an owner, and the page
// shows a login form until /auth/me says otherwise. Nothing is hidden by
// hiding it; the UI only decides what to DRAW, and the server decides what is
// allowed.
package adminui

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed assets/*
var assets embed.FS

// Handler serves the app, falling back to index.html so a refresh on a
// client-side route does not 404.
func Handler() http.Handler {
	sub, err := fs.Sub(assets, "assets")
	if err != nil {
		// The embed is compile-time; this cannot fail in a built binary.
		panic(err)
	}
	files := http.FileServer(http.FS(sub))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		name := r.URL.Path
		if name == "" || name == "/" {
			name = "index.html"
		}
		if _, err := fs.Stat(sub, trimSlash(name)); err != nil {
			r = r.Clone(r.Context())
			r.URL.Path = "/"
			http.ServeFileFS(w, r, sub, "index.html")
			return
		}
		// The admin UI must never be framed, and must never be cached across a
		// deploy — a stale bundle talking to a newer API is a confusing class
		// of bug to chase.
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Cache-Control", "no-store")
		files.ServeHTTP(w, r)
	})
}

func trimSlash(s string) string {
	for len(s) > 0 && s[0] == '/' {
		s = s[1:]
	}
	if s == "" {
		return "index.html"
	}
	return s
}
