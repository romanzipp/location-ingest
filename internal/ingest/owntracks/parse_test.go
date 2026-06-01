package owntracks

import "testing"

func TestParseLocation(t *testing.T) {
	body := []byte(`{"_type":"location","lat":52.5,"lon":13.4,"tst":1700000000,"tid":"rz","acc":7,"batt":89}`)

	loc, err := ParseLocation("phone", body)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	if loc.Lat != 52.5 || loc.Lon != 13.4 {
		t.Fatalf("bad coords: %v %v", loc.Lat, loc.Lon)
	}

	if loc.DeviceID != "rz" || loc.Battery != 89 {
		t.Fatalf("bad extra fields: %+v", loc)
	}
}

func TestParseLocationSkipsEmpty(t *testing.T) {
	if _, err := ParseLocation("phone", nil); !IsSkip(err) {
		t.Fatalf("expected skip, got %v", err)
	}
}

func TestParseLocationSkipsNonLocation(t *testing.T) {
	body := []byte(`{"_type":"lwt","tst":1700000000}`)

	if _, err := ParseLocation("phone", body); !IsSkip(err) {
		t.Fatalf("expected skip, got %v", err)
	}
}
