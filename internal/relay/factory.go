package relay

import (
	"fmt"

	"codeberg.org/romanzipp/location-ingest/internal/config"
	"codeberg.org/romanzipp/location-ingest/internal/pipeline"
)

// RelayKey builds the dispatcher key for a (client, destination) pair.
func RelayKey(client, destination string) string {
	return client + "/" + destination
}

func Build(cfg *config.Config) (map[string]pipeline.Relay, error) {
	destIdx := make(map[string]*config.Destination, len(cfg.Destinations))
	for i := range cfg.Destinations {
		destIdx[cfg.Destinations[i].Name] = &cfg.Destinations[i]
	}

	out := map[string]pipeline.Relay{}

	for _, cl := range cfg.Clients {
		for dn, cd := range cl.Destinations {
			d, ok := destIdx[dn]
			if !ok {
				return nil, fmt.Errorf("client %q: destination %q not defined", cl.Name, dn)
			}

			r, err := buildOne(d, cd)
			if err != nil {
				return nil, fmt.Errorf("client %q destination %q: %w", cl.Name, dn, err)
			}

			out[RelayKey(cl.Name, dn)] = r
		}
	}

	return out, nil
}

func buildOne(d *config.Destination, cd *config.ClientDestination) (pipeline.Relay, error) {
	switch d.Kind {
	case config.KindOwnTracksRecorder:
		return NewOwnTracksRecorder(d, cd)

	case config.KindTraccarOsmand:
		return NewTraccarOsmand(d, cd)

	case config.KindHeartbeat:
		return NewHeartbeat(d)
	}

	return nil, fmt.Errorf("unknown destination kind %q", d.Kind)
}
