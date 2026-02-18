package server

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/httplog/v2"
	"github.com/rs/zerolog/log"

	"github.com/andrewserra/kitchen/internal/config"
	"github.com/andrewserra/kitchen/internal/server/middleware"
	"github.com/andrewserra/kitchen/internal/webhook/custom"
)

type Server struct {
	cfg    *config.Config
	db     *sql.DB
	router chi.Router
}

func New(cfg *config.Config, db *sql.DB) *Server {
	s := &Server{cfg: cfg, db: db}
	s.router = s.buildRouter()
	return s
}

func (s *Server) Run(port string) error {
	addr := fmt.Sprintf(":%s", port)
	log.Info().Str("addr", addr).Str("env", s.cfg.Env).Msg("server starting")
	return http.ListenAndServe(addr, s.router)
}

func (s *Server) buildRouter() chi.Router {
	r := chi.NewRouter()

	logger := httplog.NewLogger("kitchen", httplog.Options{
		JSON:    s.cfg.LogFormat == "json",
		Concise: s.cfg.Env == "production",
	})

	r.Use(httplog.RequestLogger(logger))
	r.Use(chiMiddleware.Recoverer)
	r.Use(chiMiddleware.RequestID)
	r.Use(chiMiddleware.RealIP)

	r.Get("/health", healthHandler)

	r.Route("/webhooks", func(r chi.Router) {
		r.Route("/custom", func(r chi.Router) {
			r.Use(middleware.WebhookAuth(
				s.cfg.WebhookSecrets["custom"],
				"X-Webhook-Signature",
				"sha256=",
			))
			r.Post("/", custom.New(s.db).Handle)
		})
	})

	// Future frontend: serve static files
	r.Handle("/static/*", http.StripPrefix("/static", http.FileServer(http.Dir("./static"))))

	return r
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
