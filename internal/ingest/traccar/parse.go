package traccar

import (
	"errors"
	"net/url"
	"strconv"
	"strings"
	"time"

	"codeberg.org/romanzipp/location-ingest/internal/model"
)

var errSkip = errors.New("traccar: missing lat/lon")

func ParseLocation(client string, q url.Values) (*model.Location, error) {
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

func IsSkip(err error) bool {
	return errors.Is(err, errSkip)
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

	for _, layout := range []string{time.RFC3339, "2006-01-02 15:04:05"} {
		if t, err := time.Parse(layout, strings.TrimSpace(s)); err == nil {
			return t.UTC()
		}
	}

	return time.Now().UTC()
}
