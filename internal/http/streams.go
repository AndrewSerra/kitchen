package http

import (
	"github.com/AndrewSerra/kitchen/internal/db"
	"github.com/go-chi/chi/v5"
)

type StreamsRoute struct {
	dbService  *db.DBService
	middleware *KitchenMiddleware
}

func NewStreamsRoute(dbService *db.DBService, kitchenMiddleware *KitchenMiddleware) *StreamsRoute {
	if dbService == nil {
		panic("dbService is nil")
	}
	return &StreamsRoute{dbService: dbService, middleware: kitchenMiddleware}
}

func (sr *StreamsRoute) Routes() *chi.Mux {
	router := chi.NewRouter()
	router.Use(sr.middleware.WithValidAccessToken)
	return router
}
