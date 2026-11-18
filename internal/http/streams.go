package http

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strconv"

	"github.com/AndrewSerra/kitchen/internal/db"
	"github.com/AndrewSerra/kitchen/internal/logger"
	"github.com/AndrewSerra/kitchen/internal/models"
	"github.com/AndrewSerra/kitchen/internal/utils"
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
	router.Post("/", sr.CreateStream)
	router.Get("/", sr.GetAllStreams)
	router.Get("/{id}", sr.GetStream)
	router.Delete("/{id}", sr.DeleteStream)
	return router
}

func (sr *StreamsRoute) CreateStream(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(UserIDContextKey).(int)
	if !ok {
		responseResourceUnauthorized(w, models.CreateStreamResponse{
			Success: false,
			Message: "user_id not found in context",
			Stream:  nil,
		})
		return
	}

	var request models.CreateStreamRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		responseResourceBadRequest(w, models.CreateStreamResponse{
			Success: false,
			Message: err.Error(),
			Stream:  nil,
		})
		return
	}

	if err := utils.ValidateCreateStreamRequest(&request); err != nil {
		responseResourceBadRequest(w, models.CreateStreamResponse{
			Success: false,
			Message: err.Error(),
			Stream:  nil,
		})
		return
	}

	stream, err := sr.dbService.CreateStream(request.Name, userID)
	if err != nil {
		if errors.Is(err, db.StreamExistsError) {
			responseResourceConflict(w, models.CreateStreamResponse{
				Success: false,
				Message: err.Error(),
				Stream:  nil,
			})
			return
		}
		if errors.Is(err, db.StreamUserNotFoundError) {
			responseResourceUnauthorized(w, models.CreateStreamResponse{
				Success: false,
				Message: err.Error(),
				Stream:  nil,
			})
			return
		}
		logger.Error("Failed to create stream", "error", err)
		responseResourceInternalServerError(w, err, models.CreateStreamResponse{
			Success: false,
			Message: "Failed to create stream",
			Stream:  nil,
		})
		return
	}
	responseResourceCreated(w, models.CreateStreamResponse{
		Success: true,
		Message: "Stream created successfully",
		Stream:  stream,
	})
}

func (sr *StreamsRoute) GetStream(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(UserIDContextKey).(int)
	if !ok {
		responseResourceUnauthorized(w, models.CreateStreamResponse{
			Success: false,
			Message: "user_id not found in context",
			Stream:  nil,
		})
		return
	}

	streamId, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		responseResourceBadRequest(w, models.GetStreamResponse{
			Success: false,
			Message: "invalid stream id",
		})
		return
	}
	stream, err := sr.dbService.GetStream(userID, streamId)
	if err != nil {
		if errors.Is(err, db.StreamNotFoundError) {
			responseResourceNotFound(w, models.GetStreamResponse{
				Success: false,
				Message: "Stream not found",
			})
			return
		}

		logger.Error("Failed to get stream", "error", err)
		responseResourceInternalServerError(w, err, models.GetStreamResponse{
			Success: false,
			Message: "Failed to get stream",
		})
		return
	}

	responseResourceOK(w, models.GetStreamResponse{
		Success: true,
		Message: "Stream fetched successfully",
		Stream:  stream,
	})
}

func (sr *StreamsRoute) GetAllStreams(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(UserIDContextKey).(int)
	if !ok {
		responseResourceUnauthorized(w, models.GetAllStreamsResponse{
			Success: false,
			Message: "user_id not found in context",
		})
		return
	}

	filter, err := parseStreamListFilter(r.URL.Query())
	if err != nil {
		responseResourceBadRequest(w, models.GetAllStreamsResponse{
			Success: false,
			Message: err.Error(),
		})
		return
	}

	streams, err := sr.dbService.ListStreams(userID, filter)
	if err != nil {
		if errors.Is(err, db.InvalidStreamFilterError) {
			responseResourceBadRequest(w, models.GetAllStreamsResponse{
				Success: false,
				Message: err.Error(),
			})
			return
		}
		logger.Error("Failed to get all streams", "error", err)
		responseResourceInternalServerError(w, err, models.GetAllStreamsResponse{
			Success: false,
			Message: "Failed to get all streams",
		})
		return
	}

	responseResourceOK(w, models.GetAllStreamsResponse{
		Success: true,
		Message: "Stream fetched successfully",
		Streams: streams,
	})
}

func parseStreamListFilter(query url.Values) (models.StreamListFilter, error) {
	filter := models.StreamListFilter{
		Name:    query.Get("name"),
		SortBy:  query.Get("sort_by"),
		SortDir: query.Get("sort_direction"),
	}

	if idRaw := query.Get("id"); idRaw != "" {
		id, err := strconv.Atoi(idRaw)
		if err != nil {
			return models.StreamListFilter{}, errors.New("invalid id")
		}
		filter.ID = id
	}

	if limitRaw := query.Get("limit"); limitRaw != "" {
		limit, err := strconv.Atoi(limitRaw)
		if err != nil || limit < 0 {
			return models.StreamListFilter{}, errors.New("invalid limit")
		}
		filter.Limit = limit
	}

	if offsetRaw := query.Get("offset"); offsetRaw != "" {
		offset, err := strconv.Atoi(offsetRaw)
		if err != nil || offset < 0 {
			return models.StreamListFilter{}, errors.New("invalid offset")
		}
		filter.Offset = offset
	}

	return filter, nil
}

func (sr *StreamsRoute) DeleteStream(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(UserIDContextKey).(int)
	if !ok {
		responseResourceUnauthorized(w, models.DeleteStreamResponse{
			Success: false,
			Message: "user_id not found in context",
		})
		return
	}

	streamId, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		responseResourceBadRequest(w, models.DeleteStreamResponse{
			Success: false,
			Message: "invalid stream id",
		})
		return
	}

	if err := sr.dbService.DeleteStream(userID, streamId); err != nil {
		if errors.Is(err, db.StreamNotFoundError) {
			responseResourceNotFound(w, models.DeleteStreamResponse{
				Success: false,
				Message: "Stream not found",
			})
			return
		}

		logger.Error("Failed to delete stream", "error", err)
		responseResourceInternalServerError(w, err, models.DeleteStreamResponse{
			Success: false,
			Message: "Failed to delete stream",
		})
		return
	}

	responseResourceOK(w, models.DeleteStreamResponse{
		Success: true,
		Message: "Stream deleted successfully",
	})
}
