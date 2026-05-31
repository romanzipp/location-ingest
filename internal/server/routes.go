package server

import (
	"net/http"

	"codeberg.org/romanzipp/location-ingest/internal/config"
	"codeberg.org/romanzipp/location-ingest/internal/ingest/owntracks"
	"codeberg.org/romanzipp/location-ingest/internal/ingest/traccar"
	"codeberg.org/romanzipp/location-ingest/internal/model"
	"codeberg.org/romanzipp/location-ingest/internal/pipeline"
)

type routeContext struct {
	client       string
	kind         string
	primary      string
	destinations []string
}

func (s *Server) buildRoute(cl config.Client, src config.Source) http.Handler {
	rc := routeContext{
		client:       cl.Name,
		kind:         src.Kind,
		primary:      src.Primary,
		destinations: cl.Destinations,
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.handleIngress(w, r, rc)
	})
}

func (s *Server) handleIngress(w http.ResponseWriter, r *http.Request, rc routeContext) {
	raw, err := pipeline.CaptureRequest(r)
	if err != nil {
		s.logger.Warn("capture failed", "client", rc.client, "err", err)
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	fix := s.parseFix(rc, raw)
	if fix != nil {
		s.ring.AddFix(*fix)
	}

	if rc.primary == "" {
		s.dispatcher.DispatchFanout(rc.client, rc.destinations, "", raw, fix)
		writeDefaultResponse(w, rc.kind)
		return
	}

	resp, err := s.dispatcher.DispatchPrimary(r.Context(), rc.client, rc.primary, raw, fix)
	if err != nil {
		s.logger.Warn("primary dispatch failed", "client", rc.client, "destination", rc.primary, "err", err)
		http.Error(w, "upstream error", http.StatusBadGateway)
		return
	}

	s.dispatcher.DispatchFanout(rc.client, rc.destinations, rc.primary, raw, fix)

	if resp == nil {
		writeDefaultResponse(w, rc.kind)
		return
	}

	proxyResponse(w, resp, rc.kind)
}

func (s *Server) parseFix(rc routeContext, raw *pipeline.RawRequest) *model.Fix {
	switch rc.kind {
	case config.KindOwnTracks:
		fix, err := owntracks.ParseFix(rc.client, raw.Body)
		if err != nil {
			if !owntracks.IsSkip(err) {
				s.logger.Warn("owntracks parse failed", "client", rc.client, "err", err)
			}
			return nil
		}
		return fix

	case config.KindTraccar:
		fix, err := traccar.ParseFix(rc.client, raw.Query)
		if err != nil {
			if !traccar.IsSkip(err) {
				s.logger.Warn("traccar parse failed", "client", rc.client, "err", err)
			}
			return nil
		}
		return fix
	}

	return nil
}

func proxyResponse(w http.ResponseWriter, resp *pipeline.Response, kind string) {
	for k, vs := range resp.Header {
		if skipHopByHop(k) {
			continue
		}
		for _, v := range vs {
			w.Header().Add(k, v)
		}
	}

	status := resp.Status
	if status == 0 {
		status = http.StatusOK
	}

	if len(resp.Body) == 0 {
		writeDefaultResponse(w, kind)
		return
	}

	w.WriteHeader(status)
	_, _ = w.Write(resp.Body)
}

func writeDefaultResponse(w http.ResponseWriter, kind string) {
	switch kind {
	case config.KindOwnTracks:
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("[]"))

	default:
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}
}

func skipHopByHop(h string) bool {
	switch http.CanonicalHeaderKey(h) {
	case "Connection", "Keep-Alive", "Transfer-Encoding", "Te", "Trailer", "Upgrade", "Proxy-Authorization", "Proxy-Authenticate":
		return true
	}
	return false
}
