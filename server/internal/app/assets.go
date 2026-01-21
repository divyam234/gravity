package app

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed all:dist
var assets embed.FS

func AssetsHandler() http.Handler {
	sub, err := fs.Sub(assets, "dist")
	if err != nil {
		panic(err)
	}
	return http.FileServer(http.FS(sub))
}
