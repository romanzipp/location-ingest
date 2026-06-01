package pipeline

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	"codeberg.org/romanzipp/location-ingest/internal/model"
	"codeberg.org/romanzipp/location-ingest/internal/store"
)

const (
	queueSize     = 32
	fanoutTimeout = 15 * time.Second
	shutdownGrace = 5 * time.Second
)

type Dispatcher struct {
	logger *slog.Logger

	relays  map[string]Relay
	workers map[string]*worker

	wg     sync.WaitGroup
	once   sync.Once
	closed chan struct{}
}

type worker struct {
	relay Relay
	ch    chan job
}

type job struct {
	req *store.IngressRequest

	raw *RawRequest
	fix *model.Fix
}

func NewDispatcher(logger *slog.Logger, relays map[string]Relay) *Dispatcher {
	d := &Dispatcher{
		logger:  logger,
		relays:  relays,
		workers: make(map[string]*worker, len(relays)),
		closed:  make(chan struct{}),
	}

	for name, r := range relays {
		w := &worker{relay: r, ch: make(chan job, queueSize)}
		d.workers[name] = w

		d.wg.Add(1)
		go d.run(w)
	}

	return d
}

func (d *Dispatcher) run(w *worker) {
	defer d.wg.Done()

	for j := range w.ch {
		ctx, cancel := context.WithTimeout(context.Background(), fanoutTimeout)
		d.send(ctx, w.relay, j.req, j.raw, j.fix)
		cancel()
	}
}

func (d *Dispatcher) send(ctx context.Context, r Relay, req *store.IngressRequest, raw *RawRequest, fix *model.Fix) *Response {
	start := time.Now()

	resp, err := r.Send(ctx, raw, fix)

	result := store.DestinationResult{
		Name:     r.Name(),
		At:       time.Now().UTC(),
		Duration: time.Since(start),
	}

	if resp != nil {
		result.Status = resp.Status
	}

	if err != nil {
		result.Error = err.Error()
		d.logger.Warn("relay failed", "client", req.Client, "destination", r.Name(), "err", err)
	}

	req.AddResult(result)

	return resp
}

func (d *Dispatcher) DispatchPrimary(ctx context.Context, req *store.IngressRequest, primary string, raw *RawRequest, fix *model.Fix) (*Response, error) {
	r, ok := d.relays[primary]
	if !ok {
		return nil, errors.New("primary relay not found: " + primary)
	}

	if fix == nil && requiresFix(r) {
		req.AddResult(store.DestinationResult{Name: primary, Skipped: "requires location fix"})
		return nil, nil
	}

	return d.send(ctx, r, req, raw, fix), nil
}

func (d *Dispatcher) DispatchFanout(req *store.IngressRequest, destinations []string, primary string, raw *RawRequest, fix *model.Fix) {
	for _, name := range destinations {
		if name == primary {
			continue
		}

		w, ok := d.workers[name]
		if !ok {
			continue
		}

		if fix == nil && requiresFix(w.relay) {
			req.AddResult(store.DestinationResult{Name: name, Skipped: "requires location fix"})
			continue
		}

		select {
		case w.ch <- job{req: req, raw: raw, fix: fix}:
		default:
			d.logger.Warn("destination queue full, dropping", "client", req.Client, "destination", name)
			req.AddResult(store.DestinationResult{Name: name, Skipped: "queue full"})
		}
	}
}

func (d *Dispatcher) Close() {
	d.once.Do(func() {
		for _, w := range d.workers {
			close(w.ch)
		}

		done := make(chan struct{})
		go func() {
			d.wg.Wait()
			close(done)
		}()

		select {
		case <-done:
		case <-time.After(shutdownGrace):
			d.logger.Warn("dispatcher shutdown timed out")
		}

		close(d.closed)
	})
}
