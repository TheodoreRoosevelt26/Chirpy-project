package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync/atomic"
	"unicode/utf8"
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

func (cfg *apiConfig) validateChirpHandler(w http.ResponseWriter, r *http.Request) {
	type errorResponse struct {
		Error string `json:"error"`
	}

	type incomingChirp struct {
		Body string `json:"body"`
	}
	decoder := json.NewDecoder(r.Body)
	chirp := incomingChirp{}
	err := decoder.Decode(&chirp)
	if err != nil {
		response := errorResponse{
			Error: "Something went wrong",
		}
		file, err := json.Marshal(response)
		if err != nil {
			w.WriteHeader(500)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(400)
		w.Write(file)
		return
	}
	count := utf8.RuneCountInString(chirp.Body)
	if count > 140 {
		response := errorResponse{
			Error: "Chirp is too long",
		}
		file, err := json.Marshal(response)
		if err != nil {
			w.WriteHeader(500)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(400)
		w.Write(file)
	} else {
		type validResponse struct {
			Valid bool `json:"valid"`
		}
		response := validResponse{
			Valid: true,
		}
		file, err := json.Marshal(response)
		if err != nil {
			w.WriteHeader(500)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(200)
		w.Write(file)
	}
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
	SM.HandleFunc("POST /api/validate_chirp", apiCfg.validateChirpHandler)
	err := Server.ListenAndServe()
	if err != nil {
		log.Fatal("Error: unable to start server")
	}
}
