package relay

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"text/template"
	"time"

	"codeberg.org/romanzipp/location-ingest/internal/config"
	"codeberg.org/romanzipp/location-ingest/internal/model"
	"codeberg.org/romanzipp/location-ingest/internal/pipeline"
)

var ErrRateLimited = errors.New("heartbeat rate-limited")

type Heartbeat struct {
	name string

	method      string
	urlTmpl     *template.Template
	bodyTmpl    *template.Template
	minInterval time.Duration

	client *http.Client

	mu       sync.Mutex
	lastSent time.Time
}

type heartbeatContext struct {
	Client   string
	Source   string
	Time     time.Time
	Lat      float64
	Lon      float64
	Accuracy float64
	Altitude float64
	Speed    float64
	Bearing  float64
	Battery  float64
	Trigger  string
	DeviceID string
}

func NewHeartbeat(d *config.Destination) (*Heartbeat, error) {
	method := d.Method
	if method == "" {
		method = http.MethodGet
	}

	urlTmpl, err := template.New(d.Name + ":url").Parse(d.URL)
	if err != nil {
		return nil, fmt.Errorf("parse url template: %w", err)
	}

	var bodyTmpl *template.Template
	if strings.TrimSpace(d.Body) != "" {
		bodyTmpl, err = template.New(d.Name + ":body").Parse(d.Body)
		if err != nil {
			return nil, fmt.Errorf("parse body template: %w", err)
		}
	}

	return &Heartbeat{
		name:        d.Name,
		method:      strings.ToUpper(method),
		urlTmpl:     urlTmpl,
		bodyTmpl:    bodyTmpl,
		minInterval: d.MinInterval,
		client:      newHTTPClient(d.Timeout),
	}, nil
}

func (h *Heartbeat) Name() string { return h.name }

func (h *Heartbeat) RequiresFix() bool { return true }

func (h *Heartbeat) Send(ctx context.Context, _ *pipeline.RawRequest, fix *model.Fix) (*pipeline.Response, error) {
	if !h.allow() {
		return nil, ErrRateLimited
	}

	hc := buildContext(fix)

	var urlBuf bytes.Buffer
	if err := h.urlTmpl.Execute(&urlBuf, hc); err != nil {
		return nil, fmt.Errorf("render url: %w", err)
	}

	var body io.Reader
	if h.bodyTmpl != nil {
		var b bytes.Buffer
		if err := h.bodyTmpl.Execute(&b, hc); err != nil {
			return nil, fmt.Errorf("render body: %w", err)
		}
		body = &b
	}

	req, err := http.NewRequestWithContext(ctx, h.method, urlBuf.String(), body)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}

	resp, err := h.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
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

func (h *Heartbeat) allow() bool {
	if h.minInterval <= 0 {
		return true
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	now := time.Now()
	if now.Sub(h.lastSent) < h.minInterval {
		return false
	}

	h.lastSent = now
	return true
}

func buildContext(fix *model.Fix) heartbeatContext {
	if fix == nil {
		return heartbeatContext{Time: time.Now()}
	}

	return heartbeatContext{
		Client:   fix.Client,
		Source:   string(fix.Source),
		Time:     fix.Time,
		Lat:      fix.Lat,
		Lon:      fix.Lon,
		Accuracy: fix.Accuracy,
		Altitude: fix.Altitude,
		Speed:    fix.Speed,
		Bearing:  fix.Bearing,
		Battery:  fix.Battery,
		Trigger:  fix.Trigger,
		DeviceID: fix.DeviceID,
	}
}
