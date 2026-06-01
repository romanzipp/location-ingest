package config

import (
	"os"
	"path/filepath"
	"testing"
)

func writeTempConfig(t *testing.T, body string) string {
	t.Helper()

	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	return path
}

func TestLoadExample(t *testing.T) {
	cfg, err := Load("../../config.example.yaml")
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	if len(cfg.Clients) != 1 {
		t.Fatalf("want 1 client, got %d", len(cfg.Clients))
	}

	cl := cfg.Clients[0]
	if _, ok := cl.Destinations["kuma"]; !ok {
		t.Fatal("missing kuma in client destinations")
	}

	rec := cl.Destinations["owntracks-recorder"]
	if rec == nil || rec.Username == "" {
		t.Fatalf("owntracks-recorder client overrides missing: %+v", rec)
	}
}

func TestRejectsDuplicatePath(t *testing.T) {
	body := `
destinations:
  - name: r
    kind: owntracks_recorder
    url: http://r
clients:
  - name: a
    sources:
      - {kind: owntracks, path: /x}
    destinations:
      r:
        username: u
        device: d
  - name: b
    sources:
      - {kind: traccar, path: /x}
    destinations:
      r:
        username: u
        device: d
`

	path := writeTempConfig(t, body)

	if _, err := Load(path); err == nil {
		t.Fatal("expected duplicate path error, got nil")
	}
}

func TestRejectsUnknownDestination(t *testing.T) {
	body := `
clients:
  - name: a
    sources:
      - {kind: owntracks, path: /x, primary: ghost}
    destinations:
      ghost:
`

	path := writeTempConfig(t, body)

	if _, err := Load(path); err == nil {
		t.Fatal("expected unknown destination error, got nil")
	}
}

func TestRejectsPrimaryNotInClientDestinations(t *testing.T) {
	body := `
destinations:
  - name: r
    kind: owntracks_recorder
    url: http://r
  - name: s
    kind: owntracks_recorder
    url: http://s
clients:
  - name: a
    sources:
      - {kind: owntracks, path: /x, primary: s}
    destinations:
      r:
        username: u
        device: d
`

	path := writeTempConfig(t, body)

	if _, err := Load(path); err == nil {
		t.Fatal("expected primary-not-in-destinations error, got nil")
	}
}

func TestRejectsTraccarWithoutDeviceID(t *testing.T) {
	body := `
destinations:
  - name: t
    kind: traccar_osmand
    url: http://t
clients:
  - name: a
    sources:
      - {kind: owntracks, path: /x}
    destinations:
      t:
`

	path := writeTempConfig(t, body)

	if _, err := Load(path); err == nil {
		t.Fatal("expected device_id required error, got nil")
	}
}

func TestRejectsOwnTracksRecorderWithoutIdentity(t *testing.T) {
	body := `
destinations:
  - name: r
    kind: owntracks_recorder
    url: http://r
clients:
  - name: a
    sources:
      - {kind: traccar, path: /x}
    destinations:
      r:
`

	path := writeTempConfig(t, body)

	if _, err := Load(path); err == nil {
		t.Fatal("expected username/device required error, got nil")
	}
}
