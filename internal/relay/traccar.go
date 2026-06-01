package relay

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"

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

func (r *TraccarOsmand) Send(ctx context.Context, raw *pipeline.RawRequest, _ *model.Location) (*pipeline.Response, error) {
	target, err := url.Parse(r.url)
	if err != nil {
		return nil, fmt.Errorf("parse url: %w", err)
	}

	q := target.Query()
	for k, vs := range raw.Query {
		for _, v := range vs {
			q.Set(k, v)
		}
	}

	if r.deviceID != "" {
		q.Set("id", r.deviceID)
		q.Del("deviceid")
	}

	target.RawQuery = q.Encode()

	method := raw.Method
	if method == "" {
		method = http.MethodPost
	}

	req, err := http.NewRequestWithContext(ctx, method, target.String(), raw.BodyReader())
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}

	if ct := raw.Header.Get("Content-Type"); ct != "" {
		req.Header.Set("Content-Type", ct)
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
