package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"
)

type Server struct {
	router chi.Router

	logger zerolog.Logger
}

//go:generate $HOME/go/bin/oapi-codegen -config config.yaml -o api.gen.go openapi.yaml

func NewServer(logger zerolog.Logger) *Server {
	return &Server{
		router: chi.NewRouter(),
		logger: logger,
	}
}

func (s *Server) PostEscortAnnounce(w http.ResponseWriter, r *http.Request) {
	var announce EscortAnnounce
	if err := DecodeJSONBody(r, &announce); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	s.logger.Info().Msgf("Received escort announce: %s", announce.EscortId)

	w.WriteHeader(http.StatusOK)
}

func (s *Server) PostEscortDescribe(w http.ResponseWriter, r *http.Request) {
	var announce EscortDescribe
	if err := DecodeJSONBody(r, &announce); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	s.logger.Info().Msgf("Received escort describe: %f, %f", announce.Latitude, announce.Longitude)

	w.WriteHeader(http.StatusOK)
}
