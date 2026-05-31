package relay

import (
	"fmt"

	"codeberg.org/romanzipp/location-ingest/internal/config"
	"codeberg.org/romanzipp/location-ingest/internal/pipeline"
)

func Build(cfg *config.Config) (map[string]pipeline.Relay, error) {
	out := make(map[string]pipeline.Relay, len(cfg.Destinations))

	for i := range cfg.Destinations {
		d := &cfg.Destinations[i]

		r, err := buildOne(d)
		if err != nil {
			return nil, fmt.Errorf("destination %q: %w", d.Name, err)
		}

		out[d.Name] = r
	}

	return out, nil
}

func buildOne(d *config.Destination) (pipeline.Relay, error) {
	switch d.Kind {
	case config.KindOwnTracksRecorder:
		return NewOwnTracksRecorder(d), nil

	case config.KindTraccarOsmand:
		return NewTraccarOsmand(d), nil

	case config.KindHeartbeat:
		return NewHeartbeat(d)
	}

	return nil, fmt.Errorf("unknown destination kind %q", d.Kind)
}
