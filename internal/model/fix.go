package model

import (
	"encoding/json"
	"time"
)

type Source string

const (
	SourceOwnTracks Source = "owntracks"
	SourceTraccar   Source = "traccar"
)

type Fix struct {
	Client   string `json:"client"`
	Source   Source `json:"source"`
	Time     time.Time `json:"time"`

	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`

	Accuracy float64 `json:"accuracy,omitempty"`
	Altitude float64 `json:"altitude,omitempty"`
	Speed    float64 `json:"speed,omitempty"`
	Bearing  float64 `json:"bearing,omitempty"`
	Battery  float64 `json:"battery,omitempty"`

	Trigger  string `json:"trigger,omitempty"`
	DeviceID string `json:"device_id,omitempty"`

	Raw json.RawMessage `json:"-"`
}
