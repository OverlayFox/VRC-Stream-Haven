package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/rs/zerolog"
)

type Router struct {
	router chi.Router
	logger zerolog.Logger
	server *Server
}

func NewRouter(logger zerolog.Logger) *Router {
	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)

	server := NewServer(logger)

	return &Router{
		router: r,
		logger: logger,
		server: server,
	}
}

func (r *Router) Start() {
	handler := HandlerFromMux(r.server, r.router)

	server := &http.Server{
		Addr:    ":8080",
		Handler: handler,
	}

	errCh := make(chan error, 1)
	go func() {
		r.logger.Info().Msg("Starting API Server on :8080")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		} else {
			close(errCh)
		}
	}()

	if err, ok := <-errCh; ok {
		r.logger.Error().Err(err).Msg("API Server encountered an error")
		return
	}

	r.logger.Info().Msg("API Server stopped gracefully")
}
