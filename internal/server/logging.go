package server

import (
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"
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

type statusRecorder struct {
	http.ResponseWriter

	status int
	bytes  int
}

func (s *statusRecorder) WriteHeader(code int) {
	s.status = code
	s.ResponseWriter.WriteHeader(code)
}

func (s *statusRecorder) Write(b []byte) (int, error) {
	if s.status == 0 {
		s.status = http.StatusOK
	}

	n, err := s.ResponseWriter.Write(b)
	s.bytes += n

	return n, err
}

func accessLog(logger *slog.Logger, next http.Handler) http.Handler {
	if !httpLogEnabled() {
		return next
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		rec := &statusRecorder{ResponseWriter: w}
		next.ServeHTTP(rec, r)

		status := rec.status
		if status == 0 {
			status = http.StatusOK
		}

		logger.Info("http",
			"method", r.Method,
			"path", r.URL.Path,
			"query", r.URL.RawQuery,
			"status", status,
			"bytes", rec.bytes,
			"duration_ms", time.Since(start).Milliseconds(),
			"remote", clientAddr(r),
			"user_agent", r.UserAgent(),
		)
	})
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
