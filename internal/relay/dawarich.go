package relay

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"codeberg.org/romanzipp/location-ingest/internal/config"
	"codeberg.org/romanzipp/location-ingest/internal/model"
	"codeberg.org/romanzipp/location-ingest/internal/pipeline"
)

type Dawarich struct {
	name string

	target *url.URL
	apiKey string
	tid    string

	client *http.Client
}

func NewDawarich(d *config.Destination, cd *config.ClientDestination) (*Dawarich, error) {
	target, err := url.Parse(d.URL)
	if err != nil {
		return nil, fmt.Errorf("parse url: %w", err)
	}

	r := &Dawarich{
		name:   d.Name,
		target: target,
		client: newHTTPClient(d.Timeout),
	}

	if cd != nil {
		r.apiKey = cd.APIKey
		r.tid = cd.Tid
	}

	return r, nil
}

func (r *Dawarich) Name() string { return r.name }

func (r *Dawarich) RequiresLocation() bool { return true }

type dawarichPayload struct {
	Type string  `json:"_type"`
	Lat  float64 `json:"lat"`
	Lon  float64 `json:"lon"`
	Tst  int64   `json:"tst"`

	Acc float64 `json:"acc,omitempty"`
	Alt float64 `json:"alt,omitempty"`
	Vel float64 `json:"vel,omitempty"`
	Cog float64 `json:"cog,omitempty"`

	Batt float64 `json:"batt,omitempty"`

	Tid string `json:"tid,omitempty"`
	T   string `json:"t,omitempty"`
}

func (r *Dawarich) Send(ctx context.Context, _ *pipeline.RawRequest, loc *model.Location) (*pipeline.Response, error) {
	target := *r.target

	q := target.Query()
	q.Set("api_key", r.apiKey)
	target.RawQuery = q.Encode()

	tid := r.tid
	if tid == "" {
		tid = loc.DeviceID
	}

	payload := dawarichPayload{
		Type:    "location",
		Lat:     loc.Lat,
		Lon:     loc.Lon,
		Tst:     loc.Time.UTC().Unix(),
		Acc:     loc.Accuracy,
		Alt:     loc.Altitude,
		Vel:     loc.Speed,
		Cog:     loc.Bearing,
		Batt:    loc.Battery,
		Tid:     tid,
		T:       loc.Trigger,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, target.String(), bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := r.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := readCapped(resp.Body)
	if err != nil {
		return nil, err
	}

	return &pipeline.Response{
		Status:      resp.StatusCode,
		Header:      resp.Header,
		Body:        respBody,
		ContentType: resp.Header.Get("Content-Type"),
	}, nil
}
