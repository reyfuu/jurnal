package main

import (
	"net/http"
	"path/filepath"
	"testing"
)

func TestStaticCacheHeaders(t *testing.T) {
	assets, _ := filepath.Glob(filepath.Join("..", "..", "web", "dist", "assets", "*.js"))
	if len(assets) == 0 {
		t.Skip("frontend belum di-build")
	}
	srv := NewServer()
	for path, want := range map[string]string{
		"/":                                    "no-cache",
		"/laporan":                             "no-cache", // fallback SPA
		"/assets/" + filepath.Base(assets[0]): "public, max-age=31536000, immutable",
		"/assets/tidak-ada.js":                 "no-cache", // jatuh ke index.html
	} {
		if got := do(srv, nil, http.MethodGet, path, "").Header().Get("Cache-Control"); got != want {
			t.Errorf("%s: Cache-Control %q, want %q", path, got, want)
		}
	}
}
