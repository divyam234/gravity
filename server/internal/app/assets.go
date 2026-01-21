package app

import (
	"embed"
	"io/fs"
	"net/http"

	"github.com/wailsapp/wails/v3/pkg/application"
)

//go:embed all:dist
var assets embed.FS

func AssetsHandler() http.Handler {
	sub, err := fs.Sub(assets, "dist")
	if err != nil {
		panic(err)
	}
	return application.AssetFileServerFS(sub)
}
