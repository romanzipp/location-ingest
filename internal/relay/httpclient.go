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
