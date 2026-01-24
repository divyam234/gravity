package app

import (
	"embed"
	"io/fs"
	"net/http"
	"os"
	"path"
	"strings"
)

//go:embed all:dist
var assets embed.FS

func AssetsHandler() http.HandlerFunc {
	spaFS, err := fs.Sub(assets, "dist")
	if err != nil {
		panic(err)
	}
	return func(w http.ResponseWriter, r *http.Request) {
		filePath := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
		f, err := spaFS.Open(filePath)
		if err == nil {
			defer f.Close()
		}
		if os.IsNotExist(err) {
			r.URL.Path = "/"
			filePath = "index.html"
		}
		if filePath == "index.html" {
			w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		} else {
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		}

		http.FileServer(http.FS(spaFS)).ServeHTTP(w, r)
	}
}
