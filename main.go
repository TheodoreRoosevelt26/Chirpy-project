package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync/atomic"
	"time"
	"unicode/utf8"

	"github.com/TheodoreRoosevelt26/Chirpy-project.git/internal/database"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

type apiConfig struct {
	fileserverHits atomic.Int32
	database       *database.Queries
}

type User struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Email     string    `json:"email"`
}

type Chirp struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Body      string    `json:"body"`
	UserID    string    `json:"user_id"`
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
	godotenv.Load()
	platform := os.Getenv("PLATFORM")

	if platform == "dev" {
		err := cfg.database.DeleteAllUsers(r.Context())
		if err != nil {
			respondWithError(w, 400, "Unable to delete user table")
			return
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(200)
		cfg.fileserverHits.Store(0)
	} else {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(403)
	}
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

func (cfg *apiConfig) validateChirpHandler(originalChirp string) string {
	fullSentence := strings.ToLower(originalChirp)
	sep := " "
	ogSplitSentence := strings.Split(originalChirp, sep)
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
	/*
		type cleanedResponse struct {
			CleanBody string `json:"cleaned_body"`
		}
		cleanResponse := cleanedResponse{
			CleanBody: cleanedSentence,
		}
	*/
	return cleanedSentence
}

func (cfg *apiConfig) createUser(w http.ResponseWriter, r *http.Request) {
	type userCreation struct {
		Email string `json:"email"`
	}

	decoder := json.NewDecoder(r.Body)
	email := userCreation{}
	err := decoder.Decode(&email)
	if err != nil {
		respondWithError(w, 400, "Unable to process request")
		return
	}
	ctx := r.Context()
	dbUser, err := cfg.database.CreateUser(ctx, email.Email)
	if err != nil {
		respondWithError(w, 400, "Unable to create user")
		return
	}
	user := User{
		ID:        dbUser.ID,
		CreatedAt: dbUser.CreatedAt,
		UpdatedAt: dbUser.UpdatedAt,
		Email:     dbUser.Email,
	}
	respondWithJSON(w, 201, user)
}

func (cfg *apiConfig) chirps(w http.ResponseWriter, r *http.Request) {
	type incomingChirp struct {
		Body   string `json:"body"`
		UserID string `json:"user_id"`
	}
	decoder := json.NewDecoder(r.Body)
	newChirp := incomingChirp{}
	err := decoder.Decode(&newChirp)
	if err != nil {
		respondWithError(w, 400, "Unable to process Chirp")
		return
	}
	count := utf8.RuneCountInString(newChirp.Body)
	if count > 140 {
		code := 400
		msg := "Chirp is too long"
		respondWithError(w, code, msg)
		return
	}
	newChirp.Body = cfg.validateChirpHandler(newChirp.Body)
	ctx := r.Context()
	params := database.CreateChirpParams{
		Body:   newChirp.Body,
		UserID: newChirp.UserID,
	}
	dbChirp, err := cfg.database.CreateChirp(ctx, params)
	if err != nil {
		fmt.Printf("Error %v", err)
		respondWithError(w, 400, "Unable to create Chirp")
		return
	}
	chirp := Chirp{
		ID:        dbChirp.ID,
		CreatedAt: dbChirp.CreatedAt,
		UpdatedAt: dbChirp.UpdatedAt,
		Body:      dbChirp.Body,
		UserID:    dbChirp.UserID,
	}
	respondWithJSON(w, 201, chirp)
}

func main() {
	godotenv.Load()
	dbURL := os.Getenv("DB_URL")
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		//maybe handle it better later, ignore for now.
		fmt.Println("error opening PSQL DB")
	}
	defer db.Close()
	dbQueries := database.New(db)
	apiCfg := &apiConfig{
		database: dbQueries,
	}
	SM := http.NewServeMux()
	Server := &http.Server{Addr: ":8080", Handler: SM}
	fileServer := http.StripPrefix("/app", http.FileServer(http.Dir(".")))
	SM.Handle("/app/", apiCfg.middlewareMetricsInc(fileServer))
	SM.HandleFunc("GET /api/healthz", healthzHandler)
	SM.HandleFunc("GET /admin/metrics", apiCfg.metricsHandler)
	SM.HandleFunc("POST /admin/reset", apiCfg.resetHandler)
	SM.HandleFunc("POST /api/chirps", apiCfg.chirps)
	SM.HandleFunc("POST /api/users", apiCfg.createUser)
	err = Server.ListenAndServe()
	if err != nil {
		log.Fatal("Error: unable to start server")
	}
}
