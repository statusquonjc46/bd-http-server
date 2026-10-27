package main

import (
	"net/http"
	"sync/atomic"
)

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(200)
	w.Write([]byte("OK"))
}

type apiConfig struct {
	fileserverHits atomic.Int32
}

func (cfg *apiConfig) middlewareMetricsIncrement(next http.Handler) http.Handler {
	cfg.fileserverHits.Add(cfg.fileserverHits.Load())
	return next
}
