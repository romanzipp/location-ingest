package web

import (
	"embed"
	"html/template"
	"io/fs"
	"net/http"
	"time"

	"codeberg.org/romanzipp/location-ingest/internal/buildinfo"
	"codeberg.org/romanzipp/location-ingest/internal/model"
	"codeberg.org/romanzipp/location-ingest/internal/store"
)

//go:embed templates/*.html
var templatesFS embed.FS

//go:embed static/*
var staticFS embed.FS

type Handler struct {
	ring *store.Ring

	templates *template.Template
	static    http.Handler
}

type pageData struct {
	Title   string
	Version string

	Clients []store.ClientState
	Client  store.ClientState
}

func New(ring *store.Ring) (*Handler, error) {
	funcs := template.FuncMap{
		"formatTime": formatTime,
		"statusOK":   statusOK,
		"reverse":    reverseFixes,
	}

	tpl, err := template.New("").Funcs(funcs).ParseFS(templatesFS, "templates/*.html")
	if err != nil {
		return nil, err
	}

	sub, err := fs.Sub(staticFS, "static")
	if err != nil {
		return nil, err
	}

	return &Handler{
		ring:      ring,
		templates: tpl,
		static:    http.FileServer(http.FS(sub)),
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
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	tpl, err := h.templates.Clone()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if _, err := tpl.ParseFS(templatesFS, "templates/"+page); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

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

func statusOK(code int) bool {
	return code >= 200 && code < 300
}

func reverseFixes(in []model.Fix) []model.Fix {
	out := make([]model.Fix, len(in))
	for i, f := range in {
		out[len(in)-1-i] = f
	}
	return out
}
