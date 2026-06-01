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
	Send(ctx context.Context, raw *RawRequest, fix *model.Fix) (*Response, error)
}

type FixRequirer interface {
	RequiresFix() bool
}

func requiresFix(r Relay) bool {
	fr, ok := r.(FixRequirer)
	return ok && fr.RequiresFix()
}
