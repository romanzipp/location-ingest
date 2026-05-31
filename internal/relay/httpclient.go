package relay

import (
	"net/http"
	"time"
)

func newHTTPClient(timeout time.Duration) *http.Client {
	if timeout <= 0 {
		timeout = 10 * time.Second
	}

	return &http.Client{Timeout: timeout}
}

func mergeQueryAndHeaders(dst *http.Request, srcQuery map[string][]string, srcHeader http.Header) {
	q := dst.URL.Query()
	for k, vs := range srcQuery {
		for _, v := range vs {
			q.Set(k, v)
		}
	}
	dst.URL.RawQuery = q.Encode()

	for k, vs := range srcHeader {
		for _, v := range vs {
			dst.Header.Add(k, v)
		}
	}
}
