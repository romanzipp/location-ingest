package server

import (
	"log/slog"
	"net/http"
	"os"
	"sort"
	"strings"

	"codeberg.org/romanzipp/location-ingest/internal/pipeline"
)

const httpLogEnv = "HTTP_LOG"

func httpLogEnabled() bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv(httpLogEnv)))

	switch v {
	case "", "0", "false", "off", "no":
		return false
	}

	return true
}

func logIngress(logger *slog.Logger, r *http.Request, raw *pipeline.RawRequest, rc routeContext) {
	if !httpLogEnabled() {
		return
	}

	attrs := []any{
		"client", rc.client,
		"kind", rc.kind,
		"method", raw.Method,
		"path", raw.Path,
		"remote", clientAddr(r),
	}

	if len(raw.Query) > 0 {
		attrs = append(attrs, "query", flattenValues(raw.Query))
	}

	if len(raw.Header) > 0 {
		attrs = append(attrs, "headers", flattenHeader(raw.Header))
	}

	if len(raw.Body) > 0 {
		attrs = append(attrs, "body", string(raw.Body), "body_bytes", len(raw.Body))
	}

	logger.Info("ingress request", attrs...)
}

func clientAddr(r *http.Request) string {
	if v := r.Header.Get("X-Forwarded-For"); v != "" {
		if i := strings.IndexByte(v, ','); i > 0 {
			return strings.TrimSpace(v[:i])
		}
		return strings.TrimSpace(v)
	}

	if v := r.Header.Get("X-Real-Ip"); v != "" {
		return v
	}

	return r.RemoteAddr
}

func flattenValues(v map[string][]string) map[string]string {
	out := make(map[string]string, len(v))

	for k, vs := range v {
		out[k] = strings.Join(vs, ",")
	}

	return out
}

func flattenHeader(h http.Header) map[string]string {
	keys := make([]string, 0, len(h))
	for k := range h {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	out := make(map[string]string, len(keys))
	for _, k := range keys {
		out[k] = strings.Join(h[k], ",")
	}

	return out
}
