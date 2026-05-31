package server

import (
	"log/slog"
	"net/http"

	"codeberg.org/romanzipp/location-ingest/internal/config"
	"codeberg.org/romanzipp/location-ingest/internal/pipeline"
	"codeberg.org/romanzipp/location-ingest/internal/relay"
	"codeberg.org/romanzipp/location-ingest/internal/store"
)

type Server struct {
	logger *slog.Logger
	cfg    *config.Config

	ring       *store.Ring
	dispatcher *pipeline.Dispatcher

	mux *http.ServeMux
}

func New(cfg *config.Config, logger *slog.Logger) (*Server, error) {
	ring := store.NewRing(cfg.Web.MaxFixesPerClient)

	relays, err := relay.Build(cfg)
	if err != nil {
		return nil, err
	}

	dispatcher := pipeline.NewDispatcher(logger, ring, relays)

	s := &Server{
		logger:     logger,
		cfg:        cfg,
		ring:       ring,
		dispatcher: dispatcher,
		mux:        http.NewServeMux(),
	}

	s.registerRoutes()

	return s, nil
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

func (s *Server) Close() {
	if s.dispatcher != nil {
		s.dispatcher.Close()
	}
}

func (s *Server) registerRoutes() {
	s.mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	for _, cl := range s.cfg.Clients {
		for _, src := range cl.Sources {
			route := s.buildRoute(cl, src)
			s.mux.Handle(src.Path, route)
			s.logger.Info("route registered", "client", cl.Name, "kind", src.Kind, "path", src.Path)
		}
	}
}
