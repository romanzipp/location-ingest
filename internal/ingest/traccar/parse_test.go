package traccar

import (
	"net/url"
	"testing"
)

func TestParseFix(t *testing.T) {
	q := url.Values{
		"id":        {"phone-1"},
		"lat":       {"52.5"},
		"lon":       {"13.4"},
		"timestamp": {"1700000000"},
		"speed":     {"3.2"},
	}

	fix, err := ParseFix("phone", q)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	if fix.Lat != 52.5 || fix.Lon != 13.4 {
		t.Fatalf("bad coords")
	}

	if fix.DeviceID != "phone-1" {
		t.Fatalf("device id: %q", fix.DeviceID)
	}
}

func TestParseFixSkipsMissing(t *testing.T) {
	if _, err := ParseFix("phone", url.Values{}); !IsSkip(err) {
		t.Fatalf("expected skip, got %v", err)
	}
}
