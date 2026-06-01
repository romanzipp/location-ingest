package traccar

import (
	"net/url"
	"testing"
)

func TestParseLocation(t *testing.T) {
	q := url.Values{
		"id":        {"phone-1"},
		"lat":       {"52.5"},
		"lon":       {"13.4"},
		"timestamp": {"1700000000"},
		"speed":     {"3.2"},
	}

	loc, err := ParseLocation("phone", q)
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

func TestParseLocationSkipsMissing(t *testing.T) {
	if _, err := ParseLocation("phone", url.Values{}); !IsSkip(err) {
		t.Fatalf("expected skip, got %v", err)
	}
}
