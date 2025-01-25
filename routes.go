package main

import (
	"context"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/rs/cors"
)

func idExtractMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ankamaId := chi.URLParam(r, "id")
		ctx := context.WithValue(r.Context(), "id", ankamaId)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func Router() chi.Router {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(cors.Default().Handler)
	r.Use(middleware.Timeout(10 * time.Second))

	r.Route("/meta/webhooks", func(r chi.Router) {
		r.Get("/twitter", handleGetMetaTwitterSubscriptions)
		r.Get("/rss", handleGetMetaRssSubscriptions)
		r.Get("/almanax", handleGetMetaAlmanaxSubscriptions)
	})

	r.Route("/webhooks", func(r chi.Router) {

		r.Route("/rss", func(r chi.Router) {
			r.Post("/", handleCreateRssHook)
			r.With(idExtractMiddleware).Route("/{id}", func(r chi.Router) {
				r.Get("/", handleGetRss)
				r.Delete("/", handleDeleteRss)
				r.Put("/", handlePutRss)
			})
		})

		r.Route("/twitter", func(r chi.Router) {
			r.Post("/", handleCreateTwitterHook)
			r.With(idExtractMiddleware).Route("/{id}", func(r chi.Router) {
				r.Get("/", handleGetTwitter)
				r.Delete("/", handleDeleteTwitter)
				r.Put("/", handlePutTwitter)
			})
		})

		r.Route("/almanax", func(r chi.Router) {
			r.Post("/", handleCreateAlmanax)
			r.With(idExtractMiddleware).Route("/{id}", func(r chi.Router) {
				r.Get("/", handleGetAlmanax)
				r.Delete("/", handleDeleteAlmanaxHook)
				r.Put("/", handlePutAlmanax)
			})
		})

	})

	return r
}
