package traccar

import (
	"encoding/json"
	"errors"
	"net/url"
	"strconv"
	"strings"
	"time"

	"codeberg.org/romanzipp/location-ingest/internal/model"
	"codeberg.org/romanzipp/location-ingest/internal/pipeline"
)

var errSkip = errors.New("traccar: no location data")

func ParseLocation(client string, raw *pipeline.RawRequest) (*model.Location, error) {
	if loc, err := parseQuery(client, raw.Query); err == nil {
		return loc, nil
	}

	if isJSONBody(raw) {
		return parseJSON(client, raw.Body)
	}

	return nil, errSkip
}

func IsSkip(err error) bool {
	return errors.Is(err, errSkip)
}

func isJSONBody(raw *pipeline.RawRequest) bool {
	if len(raw.Body) == 0 {
		return false
	}

	ct := strings.ToLower(raw.Header.Get("Content-Type"))
	if strings.Contains(ct, "json") {
		return true
	}

	t := strings.TrimLeft(string(raw.Body), " \t\r\n")
	return strings.HasPrefix(t, "{")
}

func parseQuery(client string, q url.Values) (*model.Location, error) {
	if len(q) == 0 {
		return nil, errSkip
	}

	lat, okLat := parseFloat(q.Get("lat"))
	lon, okLon := parseFloat(q.Get("lon"))
	if !okLat || !okLon {
		return nil, errSkip
	}

	ts := parseTimestamp(q.Get("timestamp"))

	id := q.Get("id")
	if id == "" {
		id = q.Get("deviceid")
	}

	speed, _ := parseFloat(q.Get("speed"))
	bearing, _ := parseFloat(q.Get("bearing"))
	if bearing == 0 {
		bearing, _ = parseFloat(q.Get("heading"))
	}
	alt, _ := parseFloat(q.Get("altitude"))
	acc, _ := parseFloat(q.Get("accuracy"))
	batt, _ := parseFloat(q.Get("batt"))

	return &model.Location{
		Client:   client,
		Source:   model.SourceTraccar,
		Time:     ts,
		Lat:      lat,
		Lon:      lon,
		Accuracy: acc,
		Altitude: alt,
		Speed:    speed,
		Bearing:  bearing,
		Battery:  batt,
		DeviceID: id,
	}, nil
}

type jsonPayload struct {
	DeviceID string `json:"device_id"`

	Location struct {
		Coords struct {
			Latitude  float64 `json:"latitude"`
			Longitude float64 `json:"longitude"`
			Accuracy  float64 `json:"accuracy"`
			Altitude  float64 `json:"altitude"`
			Speed     float64 `json:"speed"`
			Heading   float64 `json:"heading"`
		} `json:"coords"`

		Battery struct {
			Level      float64 `json:"level"`
			IsCharging bool    `json:"is_charging"`
		} `json:"battery"`

		Activity struct {
			Type string `json:"type"`
		} `json:"activity"`

		Timestamp string `json:"timestamp"`
		Heartbeat bool   `json:"heartbeat"`
	} `json:"location"`
}

func parseJSON(client string, body []byte) (*model.Location, error) {
	var p jsonPayload
	if err := json.Unmarshal(body, &p); err != nil {
		return nil, err
	}

	c := p.Location.Coords
	if c.Latitude == 0 && c.Longitude == 0 {
		return nil, errSkip
	}

	loc := &model.Location{
		Client:   client,
		Source:   model.SourceTraccar,
		Time:     parseTimestamp(p.Location.Timestamp),
		Lat:      c.Latitude,
		Lon:      c.Longitude,
		Accuracy: c.Accuracy,
		Altitude: c.Altitude,
		DeviceID: p.DeviceID,
		Trigger:  p.Location.Activity.Type,
		Raw:      body,
	}

	if c.Speed >= 0 {
		loc.Speed = c.Speed
	}
	if c.Heading >= 0 {
		loc.Bearing = c.Heading
	}
	if p.Location.Battery.Level > 0 {
		loc.Battery = p.Location.Battery.Level * 100
	}

	return loc, nil
}

func parseFloat(s string) (float64, bool) {
	if s == "" {
		return 0, false
	}

	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, false
	}

	return v, true
}

func parseTimestamp(s string) time.Time {
	if s == "" {
		return time.Now().UTC()
	}

	if n, err := strconv.ParseInt(s, 10, 64); err == nil {
		if n > 1e12 {
			return time.UnixMilli(n).UTC()
		}
		return time.Unix(n, 0).UTC()
	}

	for _, layout := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02 15:04:05"} {
		if t, err := time.Parse(layout, strings.TrimSpace(s)); err == nil {
			return t.UTC()
		}
	}

	return time.Now().UTC()
}
