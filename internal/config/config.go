package config

import (
	"fmt"
	"os"
	"path"
	"strings"
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
	Enabled              bool `yaml:"enabled"`
	MaxRequestsPerClient int  `yaml:"max_requests_per_client"`
}

type Client struct {
	Name    string   `yaml:"name"`
	Sources []Source `yaml:"sources"`

	Destinations map[string]*ClientDestination `yaml:"destinations"`
}

type ClientDestination struct {
	Username  string     `yaml:"username,omitempty"`
	Device    string     `yaml:"device,omitempty"`
	BasicAuth *BasicAuth `yaml:"basic_auth,omitempty"`

	DeviceID string `yaml:"device_id,omitempty"`
}

type BasicAuth struct {
	User     string `yaml:"user"`
	Password string `yaml:"password"`
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

	expanded, missing := expandEnv(string(data))
	if len(missing) > 0 {
		return nil, fmt.Errorf("config references unset env vars: %v", missing)
	}

	var cfg Config
	if err := yaml.Unmarshal([]byte(expanded), &cfg); err != nil {
		return nil, fmt.Errorf("parse yaml: %w", err)
	}

	cfg.applyDefaults()

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func expandEnv(s string) (string, []string) {
	var missing []string
	seen := map[string]struct{}{}

	out := os.Expand(s, func(name string) string {
		v, ok := os.LookupEnv(name)
		if !ok {
			if _, dup := seen[name]; !dup {
				missing = append(missing, name)
				seen[name] = struct{}{}
			}
			return ""
		}
		return v
	})

	return out, missing
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

		for dn, cd := range cl.Destinations {
			d, ok := destIdx[dn]
			if !ok {
				return fmt.Errorf("client %q: destination %q not defined at top level", cl.Name, dn)
			}

			if err := validateClientDestination(cl.Name, d, cd); err != nil {
				return err
			}
		}

		if err := validateSources(cl, paths); err != nil {
			return err
		}
	}

	return nil
}

func validatePath(client, p string) error {
	if p == "" {
		return fmt.Errorf("client %q: source path is empty", client)
	}

	if !strings.HasPrefix(p, "/") {
		return fmt.Errorf("client %q: source path %q must start with /", client, p)
	}

	if strings.ContainsAny(p, "?#") {
		return fmt.Errorf("client %q: source path %q must not contain ? or #", client, p)
	}

	if path.Clean(p) != p {
		return fmt.Errorf("client %q: source path %q is not canonical (got %q after cleaning)", client, p, path.Clean(p))
	}

	return nil
}

func validateClientDestination(client string, d *Destination, cd *ClientDestination) error {
	switch d.Kind {
	case KindTraccarOsmand:
		if cd == nil || cd.DeviceID == "" {
			return fmt.Errorf("client %q destination %q (traccar_osmand): device_id is required so non-Traccar sources can be relayed", client, d.Name)
		}

	case KindOwnTracksRecorder:
		if cd == nil || cd.Username == "" || cd.Device == "" {
			return fmt.Errorf("client %q destination %q (owntracks_recorder): username and device are required so non-OwnTracks sources can be relayed", client, d.Name)
		}
	}

	return nil
}

func validateSources(cl Client, paths map[string]string) error {
	for _, s := range cl.Sources {
		switch s.Kind {
		case KindOwnTracks, KindTraccar:
		default:
			return fmt.Errorf("client %q: unknown source kind %q", cl.Name, s.Kind)
		}

		if err := validatePath(cl.Name, s.Path); err != nil {
			return err
		}

		if owner, dup := paths[s.Path]; dup {
			return fmt.Errorf("path %q is used by both %q and %q", s.Path, owner, cl.Name)
		}
		paths[s.Path] = cl.Name

		if s.Primary == "" {
			continue
		}

		if _, ok := cl.Destinations[s.Primary]; !ok {
			return fmt.Errorf("client %q: primary %q not in destinations map", cl.Name, s.Primary)
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
