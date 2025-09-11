package main

import (
	"log"
	"net/http"
	"sync/atomic"
)

// atomic.Int32 is a really cool standard-library type
// that allows us to safely increment and read an integer value
// across multiple goroutines (HTTP requests).
type apiConfig struct {
	fileServerHits atomic.Int32
}

const (
	port         = "8080"
	filePathRoot = "."
)

func main() {
	mux := &http.ServeMux{}
	server := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}
	cfg := &apiConfig{
		fileServerHits: atomic.Int32{},
	}

	// /app route handler to increment hits
	appHandler := (http.StripPrefix("/app", http.FileServer(http.Dir(filePathRoot))))
	wrapped := cfg.middlewareMetricsInc(appHandler)
	mux.Handle("/app/", wrapped)
	mux.Handle("/app", wrapped)

	// API health checker
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Header().Add("Content-Type", "text/plain; charset=utf-8")
		w.Write([]byte("OK"))
	})

	// metrics tracking API
	mux.HandleFunc("GET /metrics", cfg.handleMetrics)

	// reset fileServerHits in cfg
	mux.HandleFunc("POST /reset", cfg.handleReset)

	log.Printf("Serving files from %s on port: %s\n", filePathRoot, port)
	log.Fatal(server.ListenAndServe())
}
