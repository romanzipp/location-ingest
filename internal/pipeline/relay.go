package pipeline

import (
	"context"

	"codeberg.org/romanzipp/location-ingest/internal/model"
)

type Response struct {
	Status int

	Header map[string][]string
	Body   []byte

	ContentType string
}

type Relay interface {
	Name() string
	Send(ctx context.Context, raw *RawRequest, loc *model.Location) (*Response, error)
}

type LocationRequirer interface {
	RequiresLocation() bool
}

func requiresLocation(r Relay) bool {
	fr, ok := r.(LocationRequirer)
	return ok && fr.RequiresLocation()
}
