package store

import (
	"sync"
	"sync/atomic"
	"time"
)

type DestinationResult struct {
	Name string

	At       time.Time
	Status   int
	Duration time.Duration

	Error   string
	Skipped string
}

func (d DestinationResult) OK() bool {
	return d.Status >= 200 && d.Status < 300
}

type IngressRequest struct {
	ID uint64

	Time   time.Time
	Client string
	Source string

	Method string
	Path   string

	Query  map[string][]string
	Header map[string][]string
	Body   []byte

	HasLocation bool

	mu           sync.Mutex
	destinations []DestinationResult
}

func (r *IngressRequest) AddResult(d DestinationResult) {
	if d.At.IsZero() {
		d.At = time.Now().UTC()
	}

	r.mu.Lock()
	r.destinations = append(r.destinations, d)
	r.mu.Unlock()
}

func (r *IngressRequest) Destinations() []DestinationResult {
	r.mu.Lock()
	defer r.mu.Unlock()

	out := make([]DestinationResult, len(r.destinations))
	copy(out, r.destinations)

	return out
}

type ClientState struct {
	Name string

	LastSeen time.Time
	Count    int

	Requests []*IngressRequest
}

type Ring struct {
	mu       sync.RWMutex
	capacity int

	clients map[string]*entry
	order   []string

	nextID atomic.Uint64
}

type entry struct {
	name     string
	requests []*IngressRequest
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

func (r *Ring) Add(req *IngressRequest) {
	if req.ID == 0 {
		req.ID = r.nextID.Add(1)
	}

	if req.Time.IsZero() {
		req.Time = time.Now().UTC()
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	e := r.client(req.Client)
	e.requests = append(e.requests, req)

	if len(e.requests) > r.capacity {
		e.requests = e.requests[len(e.requests)-r.capacity:]
	}
}

func (r *Ring) client(name string) *entry {
	e, ok := r.clients[name]
	if ok {
		return e
	}

	e = &entry{name: name}
	r.clients[name] = e
	r.order = append(r.order, name)

	return e
}

func (r *Ring) Clients() []ClientState {
	r.mu.RLock()
	defer r.mu.RUnlock()

	out := make([]ClientState, 0, len(r.order))
	for _, n := range r.order {
		out = append(out, r.snapshot(r.clients[n], false))
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

	return r.snapshot(e, true), true
}

func (r *Ring) Request(client string, id uint64) (*IngressRequest, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	e, ok := r.clients[client]
	if !ok {
		return nil, false
	}

	for _, req := range e.requests {
		if req.ID == id {
			return req, true
		}
	}

	return nil, false
}

func (r *Ring) snapshot(e *entry, withRequests bool) ClientState {
	cs := ClientState{Name: e.name, Count: len(e.requests)}

	if cs.Count > 0 {
		cs.LastSeen = e.requests[cs.Count-1].Time
	}

	if !withRequests {
		return cs
	}

	cs.Requests = make([]*IngressRequest, cs.Count)
	copy(cs.Requests, e.requests)

	return cs
}
