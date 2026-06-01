package server

import (
	"net/http"
	"sort"

	"codeberg.org/romanzipp/location-ingest/internal/config"
	"codeberg.org/romanzipp/location-ingest/internal/ingest/owntracks"
	"codeberg.org/romanzipp/location-ingest/internal/ingest/traccar"
	"codeberg.org/romanzipp/location-ingest/internal/model"
	"codeberg.org/romanzipp/location-ingest/internal/pipeline"
	"codeberg.org/romanzipp/location-ingest/internal/relay"
	"codeberg.org/romanzipp/location-ingest/internal/store"
)

type routeContext struct {
	client string
	kind   string

	primary      string
	destinations []string
}

func (s *Server) buildRoute(cl config.Client, src config.Source) http.Handler {
	keys := make([]string, 0, len(cl.Destinations))
	for dn := range cl.Destinations {
		keys = append(keys, relay.RelayKey(cl.Name, dn))
	}
	sort.Strings(keys)

	primary := ""
	if src.Primary != "" {
		primary = relay.RelayKey(cl.Name, src.Primary)
	}

	rc := routeContext{
		client:       cl.Name,
		kind:         src.Kind,
		primary:      primary,
		destinations: keys,
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

	logIngress(s.logger, r, raw, rc)

	loc := s.parseLocation(rc, raw)

	req := &store.IngressRequest{
		Client: rc.client,
		Source: rc.kind,
		Method: raw.Method,
		Path:   raw.Path,
		Query:  raw.Query,
		Header: raw.Header,
		Body:   raw.Body,
		HasLocation: loc != nil,
	}

	s.ring.Add(req)

	if rc.primary == "" {
		s.dispatcher.DispatchFanout(req, rc.destinations, "", raw, loc)
		writeDefaultResponse(w, rc.kind)
		return
	}

	resp, err := s.dispatcher.DispatchPrimary(r.Context(), req, rc.primary, raw, loc)
	if err != nil {
		s.logger.Warn("primary dispatch failed", "client", rc.client, "destination", rc.primary, "err", err)
		http.Error(w, "upstream error", http.StatusBadGateway)
		return
	}

	s.dispatcher.DispatchFanout(req, rc.destinations, rc.primary, raw, loc)

	if resp == nil {
		writeDefaultResponse(w, rc.kind)
		return
	}

	proxyResponse(w, resp, rc.kind)
}

func (s *Server) parseLocation(rc routeContext, raw *pipeline.RawRequest) *model.Location {
	switch rc.kind {
	case config.KindOwnTracks:
		loc, err := owntracks.ParseLocation(rc.client, raw.Body)
		if err != nil {
			if !owntracks.IsSkip(err) {
				s.logger.Warn("owntracks parse failed", "client", rc.client, "err", err)
			}
			return nil
		}
		return loc

	case config.KindTraccar:
		loc, err := traccar.ParseLocation(rc.client, raw)
		if err != nil {
			if !traccar.IsSkip(err) {
				s.logger.Warn("traccar parse failed", "client", rc.client, "err", err)
			}
			return nil
		}
		return loc
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
