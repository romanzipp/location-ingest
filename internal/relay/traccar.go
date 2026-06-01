package relay

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"

	"codeberg.org/romanzipp/location-ingest/internal/config"
	"codeberg.org/romanzipp/location-ingest/internal/model"
	"codeberg.org/romanzipp/location-ingest/internal/pipeline"
)

type TraccarOsmand struct {
	name string
	url  string

	deviceID string

	client *http.Client
}

func NewTraccarOsmand(d *config.Destination, cd *config.ClientDestination) *TraccarOsmand {
	r := &TraccarOsmand{
		name:   d.Name,
		url:    d.URL,
		client: newHTTPClient(d.Timeout),
	}

	if cd != nil {
		r.deviceID = cd.DeviceID
	}

	return r
}

func (r *TraccarOsmand) Name() string { return r.name }

func (r *TraccarOsmand) RequiresLocation() bool { return true }

func (r *TraccarOsmand) Send(ctx context.Context, _ *pipeline.RawRequest, loc *model.Location) (*pipeline.Response, error) {
	target, err := url.Parse(r.url)
	if err != nil {
		return nil, fmt.Errorf("parse url: %w", err)
	}

	q := target.Query()

	id := loc.DeviceID
	if r.deviceID != "" {
		id = r.deviceID
	}
	if id != "" {
		q.Set("id", id)
	}

	q.Set("lat", formatFloat(loc.Lat))
	q.Set("lon", formatFloat(loc.Lon))
	q.Set("timestamp", strconv.FormatInt(loc.Time.UTC().Unix(), 10))

	if loc.Accuracy != 0 {
		q.Set("accuracy", formatFloat(loc.Accuracy))
	}
	if loc.Altitude != 0 {
		q.Set("altitude", formatFloat(loc.Altitude))
	}
	if loc.Speed != 0 {
		q.Set("speed", formatFloat(loc.Speed))
	}
	if loc.Bearing != 0 {
		q.Set("bearing", formatFloat(loc.Bearing))
	}
	if loc.Battery != 0 {
		q.Set("batt", formatFloat(loc.Battery))
	}

	target.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}

	resp, err := r.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return &pipeline.Response{
		Status:      resp.StatusCode,
		Header:      resp.Header,
		Body:        body,
		ContentType: resp.Header.Get("Content-Type"),
	}, nil
}

func formatFloat(v float64) string {
	return strconv.FormatFloat(v, 'f', -1, 64)
}
