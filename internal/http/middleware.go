package http

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/AndrewSerra/kitchen/internal/db"
	"github.com/AndrewSerra/kitchen/internal/models"
	"github.com/AndrewSerra/kitchen/internal/utils"
)

var AuthorizationHeaderMissingError = errors.New("missing authorization header")
var AuthorizationHeaderInvalidError = errors.New("invalid authorization header")

type KitchenMiddleware struct {
	dbService      *db.DBService
	tokenGenerator *utils.TokenGenerator
}

func NewKitchenMiddleware(dbService *db.DBService) *KitchenMiddleware {
	if dbService == nil {
		panic("dbService is nil")
	}

	return &KitchenMiddleware{
		dbService:      dbService,
		tokenGenerator: utils.NewTokenGenerator(),
	}
}

func (km *KitchenMiddleware) extractAuthorizationHeaderToken(r *http.Request) (string, error) {
	authorizationHeader := r.Header.Get("Authorization")
	if authorizationHeader == "" {
		return "", AuthorizationHeaderMissingError
	}

	headerParts := strings.SplitN(authorizationHeader, " ", 2)
	if len(headerParts) != 2 || !strings.EqualFold(headerParts[0], "Bearer") || headerParts[1] == "" {
		return "", AuthorizationHeaderInvalidError
	}

	return headerParts[1], nil
}

func (km *KitchenMiddleware) WithValidAccessToken(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tokenString, err := km.extractAuthorizationHeaderToken(r)
		if err != nil {
			responseResourceUnauthorized(w, models.AuthMiddlewareUnauthorizedResponse{
				Success: false,
				Message: err.Error(),
			})
			return
		}

		userID, err := km.tokenGenerator.GetUserIDFromToken(tokenString)
		if err != nil {
			responseResourceUnauthorized(w, models.AuthMiddlewareUnauthorizedResponse{
				Success: false,
				Message: err.Error(),
			})
			return
		}

		ctx := context.WithValue(r.Context(), "userID", userID)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (km *KitchenMiddleware) WithValidRefreshToken(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tokenString, err := km.extractAuthorizationHeaderToken(r)
		if err != nil {
			responseResourceUnauthorized(w, models.AuthMiddlewareUnauthorizedResponse{
				Success: false,
				Message: err.Error(),
			})
			return
		}

		tokenId, err := km.tokenGenerator.GetTokenIDFromToken(tokenString)
		if err != nil {
			responseResourceUnauthorized(w, models.AuthMiddlewareUnauthorizedResponse{
				Success: false,
				Message: err.Error(),
			})
			return
		}

		tokenRecord, err := km.dbService.GetRefreshToken(tokenId)
		if err != nil {
			responseResourceUnauthorized(w, models.AuthMiddlewareUnauthorizedResponse{
				Success: false,
				Message: err.Error(),
			})
			return
		}

		// Token hash mismatch check
		if !utils.CompareTokenToHash(tokenString, tokenRecord.Token) {
			responseResourceUnauthorized(w, models.AuthMiddlewareUnauthorizedResponse{
				Success: false,
				Message: "Invalid token",
			})
			return
		}

		next.ServeHTTP(w, r)
	})
}
