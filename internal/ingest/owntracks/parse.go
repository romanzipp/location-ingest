package owntracks

import (
	"encoding/json"
	"errors"
	"time"

	"codeberg.org/romanzipp/location-ingest/internal/model"
)

type payload struct {
	Type string `json:"_type"`

	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
	Tst int64   `json:"tst"`

	Acc     float64 `json:"acc"`
	Alt     float64 `json:"alt"`
	Vel     float64 `json:"vel"`
	Cog     float64 `json:"cog"`
	Batt    float64 `json:"batt"`
	Trigger string  `json:"t"`
	Tid     string  `json:"tid"`
}

var errSkip = errors.New("owntracks: not a location payload")

func ParseLocation(client string, body []byte) (*model.Location, error) {
	if len(body) == 0 {
		return nil, errSkip
	}

	var p payload
	if err := json.Unmarshal(body, &p); err != nil {
		return nil, err
	}

	if p.Type != "location" {
		return nil, errSkip
	}

	ts := time.Unix(p.Tst, 0).UTC()
	if p.Tst == 0 {
		ts = time.Now().UTC()
	}

	return &model.Location{
		Client:   client,
		Source:   model.SourceOwnTracks,
		Time:     ts,
		Lat:      p.Lat,
		Lon:      p.Lon,
		Accuracy: p.Acc,
		Altitude: p.Alt,
		Speed:    p.Vel,
		Bearing:  p.Cog,
		Battery:  p.Batt,
		Trigger:  p.Trigger,
		DeviceID: p.Tid,
		Raw:      json.RawMessage(body),
	}, nil
}

func IsSkip(err error) bool {
	return errors.Is(err, errSkip)
}
