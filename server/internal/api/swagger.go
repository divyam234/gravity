package api

import (
	httpSwagger "github.com/swaggo/http-swagger/v2"
	_ "gravity/docs" // Import generated docs
)

func MountSwagger(r *Router) {
	r.chi.Get("/swagger/*", httpSwagger.Handler(
		httpSwagger.URL("/swagger/doc.json"), // The url pointing to API definition
	))
}
