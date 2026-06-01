package traccar

import (
	"net/http"
	"net/url"
	"testing"

	"codeberg.org/romanzipp/location-ingest/internal/pipeline"
)

func TestParseLocationFromQuery(t *testing.T) {
	raw := &pipeline.RawRequest{
		Query: url.Values{
			"id":        {"phone-1"},
			"lat":       {"52.5"},
			"lon":       {"13.4"},
			"timestamp": {"1700000000"},
			"speed":     {"3.2"},
		},
		Header: http.Header{},
	}

	loc, err := ParseLocation("phone", raw)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	if loc.Lat != 52.5 || loc.Lon != 13.4 {
		t.Fatalf("bad coords")
	}

	if loc.DeviceID != "phone-1" {
		t.Fatalf("device id: %q", loc.DeviceID)
	}
}

func TestParseLocationFromJSON(t *testing.T) {
	body := []byte(`{
		"device_id":"25497507",
		"location":{
			"coords":{"latitude":50.5577,"longitude":9.6902,"accuracy":3.92,"altitude":291.94,"speed":-1,"heading":-1},
			"battery":{"level":0.9,"is_charging":false},
			"activity":{"type":"still"},
			"timestamp":"2026-05-30T17:51:22.906Z"
		}
	}`)

	raw := &pipeline.RawRequest{
		Header: http.Header{"Content-Type": []string{"application/json; charset=utf-8"}},
		Body:   body,
	}

	loc, err := ParseLocation("phone", raw)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	if loc.Lat != 50.5577 || loc.Lon != 9.6902 {
		t.Fatalf("bad coords: %+v", loc)
	}

	if loc.DeviceID != "25497507" {
		t.Fatalf("device id: %q", loc.DeviceID)
	}

	if loc.Battery != 90 {
		t.Fatalf("battery expected 90, got %v", loc.Battery)
	}

	if loc.Speed != 0 || loc.Bearing != 0 {
		t.Fatalf("negative speed/heading should be dropped: %+v", loc)
	}

	if loc.Trigger != "still" {
		t.Fatalf("trigger: %q", loc.Trigger)
	}
}

func TestParseLocationSkipsEmpty(t *testing.T) {
	raw := &pipeline.RawRequest{Header: http.Header{}}

	if _, err := ParseLocation("phone", raw); !IsSkip(err) {
		t.Fatalf("expected skip, got %v", err)
	}
}
