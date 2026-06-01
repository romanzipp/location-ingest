package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server       Server        `yaml:"server"`
	Web          Web           `yaml:"web"`
	Clients      []Client      `yaml:"clients"`
	Destinations []Destination `yaml:"destinations"`
}

type Server struct {
	Listen string `yaml:"listen"`
}

type Web struct {
	Enabled             bool `yaml:"enabled"`
	MaxRequestsPerClient int `yaml:"max_requests_per_client"`
}

type Client struct {
	Name         string   `yaml:"name"`
	Sources      []Source `yaml:"sources"`
	Destinations []string `yaml:"destinations"`
}

type Source struct {
	Kind    string `yaml:"kind"`
	Path    string `yaml:"path"`
	Primary string `yaml:"primary"`
}

type Destination struct {
	Name string `yaml:"name"`
	Kind string `yaml:"kind"`
	URL  string `yaml:"url"`

	Method string `yaml:"method,omitempty"`
	Body   string `yaml:"body,omitempty"`

	MinInterval time.Duration `yaml:"min_interval,omitempty"`
	Timeout     time.Duration `yaml:"timeout,omitempty"`
}

const (
	KindOwnTracks         = "owntracks"
	KindTraccar           = "traccar"
	KindOwnTracksRecorder = "owntracks_recorder"
	KindTraccarOsmand     = "traccar_osmand"
	KindHeartbeat         = "heartbeat"
)

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse yaml: %w", err)
	}

	cfg.applyDefaults()

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func (c *Config) applyDefaults() {
	if c.Server.Listen == "" {
		c.Server.Listen = ":8080"
	}

	if c.Web.MaxRequestsPerClient <= 0 {
		c.Web.MaxRequestsPerClient = 500
	}

	for i := range c.Destinations {
		d := &c.Destinations[i]

		if d.Timeout == 0 {
			d.Timeout = 10 * time.Second
		}

		if d.Kind == KindHeartbeat && d.Method == "" {
			d.Method = "GET"
		}
	}
}

func (c *Config) validate() error {
	if len(c.Clients) == 0 {
		return fmt.Errorf("no clients configured")
	}

	destIdx, err := c.validateDestinations()
	if err != nil {
		return err
	}

	return c.validateClients(destIdx)
}

func (c *Config) validateDestinations() (map[string]*Destination, error) {
	idx := map[string]*Destination{}

	for i := range c.Destinations {
		d := &c.Destinations[i]

		if d.Name == "" {
			return nil, fmt.Errorf("destination[%d] has empty name", i)
		}

		if _, dup := idx[d.Name]; dup {
			return nil, fmt.Errorf("destination %q is duplicated", d.Name)
		}

		switch d.Kind {
		case KindOwnTracksRecorder, KindTraccarOsmand, KindHeartbeat:
		default:
			return nil, fmt.Errorf("destination %q: unknown kind %q", d.Name, d.Kind)
		}

		if d.URL == "" {
			return nil, fmt.Errorf("destination %q: url is required", d.Name)
		}

		idx[d.Name] = d
	}

	return idx, nil
}

func (c *Config) validateClients(destIdx map[string]*Destination) error {
	clientNames := map[string]struct{}{}
	paths := map[string]string{}

	for _, cl := range c.Clients {
		if cl.Name == "" {
			return fmt.Errorf("client has empty name")
		}

		if _, dup := clientNames[cl.Name]; dup {
			return fmt.Errorf("client %q is duplicated", cl.Name)
		}
		clientNames[cl.Name] = struct{}{}

		if len(cl.Sources) == 0 {
			return fmt.Errorf("client %q: no sources", cl.Name)
		}

		clientDests := map[string]struct{}{}
		for _, dn := range cl.Destinations {
			if _, ok := destIdx[dn]; !ok {
				return fmt.Errorf("client %q: destination %q not defined", cl.Name, dn)
			}
			clientDests[dn] = struct{}{}
		}

		if err := validateSources(cl, clientDests, paths); err != nil {
			return err
		}
	}

	return nil
}

func validateSources(cl Client, clientDests map[string]struct{}, paths map[string]string) error {
	for _, s := range cl.Sources {
		switch s.Kind {
		case KindOwnTracks, KindTraccar:
		default:
			return fmt.Errorf("client %q: unknown source kind %q", cl.Name, s.Kind)
		}

		if s.Path == "" || s.Path[0] != '/' {
			return fmt.Errorf("client %q: source path must start with /", cl.Name)
		}

		if owner, dup := paths[s.Path]; dup {
			return fmt.Errorf("path %q is used by both %q and %q", s.Path, owner, cl.Name)
		}
		paths[s.Path] = cl.Name

		if s.Primary == "" {
			continue
		}

		if _, ok := clientDests[s.Primary]; !ok {
			return fmt.Errorf("client %q: primary %q not in destinations list", cl.Name, s.Primary)
		}
	}

	return nil
}

func (c *Config) DestinationByName(name string) *Destination {
	for i := range c.Destinations {
		if c.Destinations[i].Name == name {
			return &c.Destinations[i]
		}
	}
	return nil
}
