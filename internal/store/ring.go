package store

import (
	"sync"
	"time"

	"codeberg.org/romanzipp/location-ingest/internal/model"
)

type DestinationStatus struct {
	Name string

	LastTime   time.Time
	LastStatus int
	LastError  string
}

type ClientState struct {
	Name string

	LastFix *model.Fix
	Fixes   []model.Fix

	Destinations []DestinationStatus
}

type Ring struct {
	mu       sync.RWMutex
	capacity int

	clients map[string]*entry
	order   []string
}

type entry struct {
	name string

	fixes []model.Fix

	dest    map[string]*DestinationStatus
	destOrd []string
}

func NewRing(capacity int) *Ring {
	if capacity <= 0 {
		capacity = 500
	}

	return &Ring{
		capacity: capacity,
		clients:  map[string]*entry{},
	}
}

func (r *Ring) client(name string) *entry {
	e, ok := r.clients[name]
	if ok {
		return e
	}

	e = &entry{name: name, dest: map[string]*DestinationStatus{}}
	r.clients[name] = e
	r.order = append(r.order, name)

	return e
}

func (r *Ring) AddFix(fix model.Fix) {
	r.mu.Lock()
	defer r.mu.Unlock()

	e := r.client(fix.Client)

	e.fixes = append(e.fixes, fix)
	if len(e.fixes) > r.capacity {
		e.fixes = e.fixes[len(e.fixes)-r.capacity:]
	}
}

func (r *Ring) RecordDestination(client, dest string, status int, errMsg string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	e := r.client(client)

	s, ok := e.dest[dest]
	if !ok {
		s = &DestinationStatus{Name: dest}
		e.dest[dest] = s
		e.destOrd = append(e.destOrd, dest)
	}

	s.LastTime = time.Now()
	s.LastStatus = status
	s.LastError = errMsg
}

func (r *Ring) Clients() []ClientState {
	r.mu.RLock()
	defer r.mu.RUnlock()

	out := make([]ClientState, 0, len(r.order))
	for _, n := range r.order {
		out = append(out, r.snapshot(r.clients[n]))
	}

	return out
}

func (r *Ring) Client(name string) (ClientState, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	e, ok := r.clients[name]
	if !ok {
		return ClientState{Name: name}, false
	}

	return r.snapshot(e), true
}

func (r *Ring) snapshot(e *entry) ClientState {
	cs := ClientState{Name: e.name}

	if n := len(e.fixes); n > 0 {
		cp := make([]model.Fix, n)
		copy(cp, e.fixes)

		cs.Fixes = cp

		last := cs.Fixes[n-1]
		cs.LastFix = &last
	}

	cs.Destinations = make([]DestinationStatus, 0, len(e.destOrd))
	for _, n := range e.destOrd {
		cs.Destinations = append(cs.Destinations, *e.dest[n])
	}

	return cs
}
