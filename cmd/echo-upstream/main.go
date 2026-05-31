package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
)

func main() {
	addr := flag.String("addr", ":9001", "listen address")
	flag.Parse()

	mux := http.NewServeMux()
	mux.HandleFunc("/", handle)

	log.Printf("echo-upstream listening on %s", *addr)

	if err := http.ListenAndServe(*addr, mux); err != nil {
		log.Fatal(err)
	}
}

func handle(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	_ = r.Body.Close()

	log.Printf("→ %s %s", r.Method, r.URL.String())

	for k, vs := range r.Header {
		for _, v := range vs {
			log.Printf("   %s: %s", k, v)
		}
	}

	if len(body) > 0 {
		log.Printf("   body: %s", body)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if _, err := fmt.Fprint(w, "[]"); err != nil {
		log.Printf("write: %v", err)
	}
}
