package owntracks

import "testing"

func TestParseFix(t *testing.T) {
	body := []byte(`{"_type":"location","lat":52.5,"lon":13.4,"tst":1700000000,"tid":"rz","acc":7,"batt":89}`)

	fix, err := ParseFix("phone", body)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	if fix.Lat != 52.5 || fix.Lon != 13.4 {
		t.Fatalf("bad coords: %v %v", fix.Lat, fix.Lon)
	}

	if fix.DeviceID != "rz" || fix.Battery != 89 {
		t.Fatalf("bad extra fields: %+v", fix)
	}
}

func TestParseFixSkipsEmpty(t *testing.T) {
	if _, err := ParseFix("phone", nil); !IsSkip(err) {
		t.Fatalf("expected skip, got %v", err)
	}
}

func TestParseFixSkipsNonLocation(t *testing.T) {
	body := []byte(`{"_type":"lwt","tst":1700000000}`)

	if _, err := ParseFix("phone", body); !IsSkip(err) {
		t.Fatalf("expected skip, got %v", err)
	}
}
