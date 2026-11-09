package http

import (
	"encoding/json"
	"net/http"

	"github.com/AndrewSerra/kitchen/internal/logger"
)

func writeResponse(w http.ResponseWriter, status int, body interface{}) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	return json.NewEncoder(w).Encode(body)
}

func responseResourceOK(w http.ResponseWriter, body interface{}) error {
	return writeResponse(w, http.StatusOK, body)
}

func responseResourceCreated(w http.ResponseWriter, body interface{}) error {
	return writeResponse(w, http.StatusCreated, body)
}

func responseResourceNotFound(w http.ResponseWriter, body interface{}) error {
	return writeResponse(w, http.StatusNotFound, body)
}

func responseResourceBadRequest(w http.ResponseWriter, body interface{}) error {
	return writeResponse(w, http.StatusBadRequest, body)
}

func responseResourceInternalServerError(w http.ResponseWriter, err error, body interface{}) error {
	logger.Error("internal server error", "err", err)
	return writeResponse(w, http.StatusInternalServerError, body)
}

func responseResourceConflict(w http.ResponseWriter, body interface{}) error {
	return writeResponse(w, http.StatusConflict, body)
}

func responseResourceUnauthorized(w http.ResponseWriter, body interface{}) error {
	return writeResponse(w, http.StatusUnauthorized, body)
}

func responseResourceForbidden(w http.ResponseWriter, body interface{}) error {
	return writeResponse(w, http.StatusForbidden, body)
}
