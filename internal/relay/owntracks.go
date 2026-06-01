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

type OwnTracksRecorder struct {
	name string
	url  string

	username string
	device   string

	basicUser string
	basicPass string
	hasBasic  bool

	client *http.Client
}

func NewOwnTracksRecorder(d *config.Destination) *OwnTracksRecorder {
	r := &OwnTracksRecorder{
		name:     d.Name,
		url:      d.URL,
		username: d.Username,
		device:   d.Device,
		client:   newHTTPClient(d.Timeout),
	}

	if d.BasicAuth != nil && d.BasicAuth.User != "" {
		r.basicUser = d.BasicAuth.User
		r.basicPass = d.BasicAuth.Password
		r.hasBasic = true
	}

	return r
}

func (r *OwnTracksRecorder) Name() string { return r.name }

func (r *OwnTracksRecorder) Send(ctx context.Context, raw *pipeline.RawRequest, _ *model.Location) (*pipeline.Response, error) {
	target, err := url.Parse(r.url)
	if err != nil {
		return nil, fmt.Errorf("parse url: %w", err)
	}

	q := target.Query()
	if r.username != "" {
		q.Set("u", r.username)
	}
	if r.device != "" {
		q.Set("d", r.device)
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

	req.Header.Set("Content-Type", "application/json")

	if r.username != "" {
		req.Header.Set("X-Limit-U", r.username)
	}
	if r.device != "" {
		req.Header.Set("X-Limit-D", r.device)
	}
	if r.hasBasic {
		req.SetBasicAuth(r.basicUser, r.basicPass)
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
