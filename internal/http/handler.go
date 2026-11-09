package http

import "github.com/go-chi/chi/v5"

type HttpRoutes interface {
	Routes() *chi.Mux
}
