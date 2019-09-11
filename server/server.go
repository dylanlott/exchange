package server

import (
	"net/http"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"

	"github.com/dylanlott/exchange/db"
	v1 "github.com/dylanlott/exchange/server/api/v1"
)

// NewRouter returns a new HTTP handler that implements the main server routes
func NewRouter(database *db.DB) http.Handler {
	router := chi.NewRouter()

	// Set up our middleware with sane defaults
	router.Use(middleware.RealIP)
	router.Use(middleware.Logger)
	router.Use(middleware.Recoverer)
	router.Use(middleware.DefaultCompress)
	router.Use(middleware.Timeout(60 * time.Second))

	// Set up our API
	router.Mount("/api/v1/", v1.NewRouter(database))

	// serve web app
	router.Handle("/*", http.FileServer(http.Dir("./web/exchange/dist")))

	return router
}
