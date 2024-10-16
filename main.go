package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
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

func respondWithError(w http.ResponseWriter, code int, msg string) {
	type errorResponse struct {
		Error string `json:"error"`
	}
	response := errorResponse{
		Error: msg,
	}
	file, err := json.Marshal(response)
	if err != nil {
		w.WriteHeader(500)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	w.Write(file)
}

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	file, err := json.Marshal(payload)
	if err != nil {
		w.WriteHeader(500)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	w.Write(file)
}

func (cfg *apiConfig) validateChirpHandler(w http.ResponseWriter, r *http.Request) {
	type incomingChirp struct {
		Body string `json:"body"`
	}
	decoder := json.NewDecoder(r.Body)
	chirp := incomingChirp{}
	err := decoder.Decode(&chirp)
	if err != nil {
		code := 400
		msg := "Something went wrong"
		respondWithError(w, code, msg)
		return
	}
	count := utf8.RuneCountInString(chirp.Body)
	if count > 140 {
		code := 400
		msg := "Chirp is too long"
		respondWithError(w, code, msg)
		return
	}

	fullSentence := strings.ToLower(chirp.Body)
	sep := " "
	ogSplitSentence := strings.Split(chirp.Body, sep)
	splitSentence := strings.Split(fullSentence, sep)
	for index, word := range splitSentence {
		switch word {
		case "kerfuffle":
			ogSplitSentence[index] = "****"
		case "sharbert":
			ogSplitSentence[index] = "****"
		case "fornax":
			ogSplitSentence[index] = "****"
		default:
			continue
		}
	}
	cleanedSentence := strings.Join(ogSplitSentence, sep)
	type cleanedResponse struct {
		CleanBody string `json:"cleaned_body"`
	}
	cleanResponse := cleanedResponse{
		CleanBody: cleanedSentence,
	}
	code := 200
	respondWithJSON(w, code, cleanResponse)
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
