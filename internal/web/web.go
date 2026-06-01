package web

import (
	"embed"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"sort"
	"strings"
	"time"

	"codeberg.org/romanzipp/location-ingest/internal/buildinfo"
	"codeberg.org/romanzipp/location-ingest/internal/store"
)

//go:embed templates/*.html
var templatesFS embed.FS

//go:embed static/*
var staticFS embed.FS

type Handler struct {
	ring *store.Ring

	pages  map[string]*template.Template
	static http.Handler
}

type pageData struct {
	Title   string
	Version string

	Clients []store.ClientState
	Client  store.ClientState
}

type requestView struct {
	ID uint64

	Time   time.Time
	Source string
	Method string
	Path   string

	HasLocation bool

	Query   []kv
	Headers []kv
	Body    string

	Destinations []store.DestinationResult
}

type kv struct {
	Key   string
	Value string
}

func New(ring *store.Ring) (*Handler, error) {
	funcs := template.FuncMap{
		"formatTime":   formatTime,
		"requestViews": buildRequestViews,
		"isJSON":       isJSON,
	}

	pages := map[string]*template.Template{}
	for _, page := range []string{"index.html", "client.html"} {
		tpl, err := template.New(page).Funcs(funcs).ParseFS(templatesFS, "templates/layout.html", "templates/"+page)
		if err != nil {
			return nil, fmt.Errorf("parse %s: %w", page, err)
		}
		pages[page] = tpl
	}

	sub, err := fs.Sub(staticFS, "static")
	if err != nil {
		return nil, err
	}

	return &Handler{
		ring:   ring,
		pages:  pages,
		static: http.FileServer(http.FS(sub)),
	}, nil
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("/", h.index)
	mux.HandleFunc("/clients/", h.client)
	mux.Handle("/static/", http.StripPrefix("/static/", h.static))
}

func (h *Handler) index(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	data := pageData{
		Title:   "overview",
		Version: buildinfo.Version,
		Clients: h.ring.Clients(),
	}

	h.render(w, "index.html", data)
}

func (h *Handler) client(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Path[len("/clients/"):]
	if name == "" {
		http.NotFound(w, r)
		return
	}

	state, ok := h.ring.Client(name)
	if !ok {
		http.NotFound(w, r)
		return
	}

	data := pageData{
		Title:   name,
		Version: buildinfo.Version,
		Client:  state,
	}

	h.render(w, "client.html", data)
}

func (h *Handler) render(w http.ResponseWriter, page string, data pageData) {
	tpl, ok := h.pages[page]
	if !ok {
		http.Error(w, "unknown page", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	if err := tpl.ExecuteTemplate(w, "layout", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return "—"
	}
	return t.UTC().Format("2006-01-02 15:04:05 UTC")
}

func buildRequestViews(reqs []*store.IngressRequest) []requestView {
	out := make([]requestView, 0, len(reqs))

	for i := len(reqs) - 1; i >= 0; i-- {
		req := reqs[i]

		out = append(out, requestView{
			ID:           req.ID,
			Time:         req.Time,
			Source:       req.Source,
			Method:       req.Method,
			Path:         req.Path,
			HasLocation:       req.HasLocation,
			Query:        sortedKV(req.Query),
			Headers:      sortedKV(req.Header),
			Body:         string(req.Body),
			Destinations: req.Destinations(),
		})
	}

	return out
}

func sortedKV(m map[string][]string) []kv {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	out := make([]kv, 0, len(keys))
	for _, k := range keys {
		out = append(out, kv{Key: k, Value: strings.Join(m[k], ", ")})
	}

	return out
}

func isJSON(body string) bool {
	t := strings.TrimSpace(body)
	if t == "" {
		return false
	}
	return t[0] == '{' || t[0] == '['
}
