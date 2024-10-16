package main

import (
	"fmt"
	"log"
	"net/http"
	"sync/atomic"
)

type apiConfig struct {
	fileserverHits atomic.Int32
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits.Add(1)
		next.ServeHTTP(w, r)
	})
}

func healthzHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(200)
	w.Write([]byte("OK"))
}

func (cfg *apiConfig) metricsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(200)
	val := fmt.Sprintf("<html><body><h1>Welcome, Chirpy Admin</h1><p>Chirpy has been visited %d times!</p></body></html>", cfg.fileserverHits.Load())
	w.Write([]byte(val))
}

func (cfg *apiConfig) resetHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(200)
	cfg.fileserverHits.Store(0)
}

func main() {
	apiCfg := &apiConfig{}
	SM := http.NewServeMux()
	Server := &http.Server{Addr: ":8080", Handler: SM}
	fileServer := http.StripPrefix("/app", http.FileServer(http.Dir(".")))
	SM.Handle("/app/", apiCfg.middlewareMetricsInc(fileServer))
	SM.HandleFunc("GET /api/healthz", healthzHandler)
	SM.HandleFunc("GET /admin/metrics", apiCfg.metricsHandler)
	SM.HandleFunc("POST /admin/reset", apiCfg.resetHandler)
	err := Server.ListenAndServe()
	if err != nil {
		log.Fatal("Error: unable to start server")
	}
}
