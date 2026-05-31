package pipeline

import (
	"bytes"
	"io"
	"net/http"
	"net/url"
)

type RawRequest struct {
	Method string
	Path   string

	Query  url.Values
	Header http.Header

	Body []byte
}

func CaptureRequest(r *http.Request) (*RawRequest, error) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	_ = r.Body.Close()

	hdr := r.Header.Clone()
	hdr.Del("Host")
	hdr.Del("Content-Length")
	hdr.Del("Connection")

	return &RawRequest{
		Method: r.Method,
		Path:   r.URL.Path,
		Query:  cloneValues(r.URL.Query()),
		Header: hdr,
		Body:   body,
	}, nil
}

func (rr *RawRequest) BodyReader() io.Reader {
	return bytes.NewReader(rr.Body)
}

func cloneValues(v url.Values) url.Values {
	out := make(url.Values, len(v))

	for k, vs := range v {
		cp := make([]string, len(vs))
		copy(cp, vs)
		out[k] = cp
	}

	return out
}
