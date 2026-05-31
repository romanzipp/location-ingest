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

	if cfg.DestinationByName("kuma") == nil {
		t.Fatal("missing kuma destination")
	}
}

func TestRejectsDuplicatePath(t *testing.T) {
	body := `
clients:
  - name: a
    sources:
      - {kind: owntracks, path: /x}
    destinations: []
  - name: b
    sources:
      - {kind: traccar, path: /x}
    destinations: []
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
    destinations: [ghost]
`

	path := writeTempConfig(t, body)

	if _, err := Load(path); err == nil {
		t.Fatal("expected unknown destination error, got nil")
	}
}
