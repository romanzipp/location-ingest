package relay

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"codeberg.org/romanzipp/location-ingest/internal/config"
	"codeberg.org/romanzipp/location-ingest/internal/model"
	"codeberg.org/romanzipp/location-ingest/internal/pipeline"
)

type OwnTracksRecorder struct {
	name string
	url  string

	client *http.Client
}

func NewOwnTracksRecorder(d *config.Destination) *OwnTracksRecorder {
	return &OwnTracksRecorder{
		name:   d.Name,
		url:    d.URL,
		client: newHTTPClient(d.Timeout),
	}
}

func (r *OwnTracksRecorder) Name() string { return r.name }

func (r *OwnTracksRecorder) Send(ctx context.Context, raw *pipeline.RawRequest, _ *model.Fix) (*pipeline.Response, error) {
	method := raw.Method
	if method == "" {
		method = http.MethodPost
	}

	req, err := http.NewRequestWithContext(ctx, method, r.url, raw.BodyReader())
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}

	mergeQueryAndHeaders(req, raw.Query, raw.Header)

	if req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/json")
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
