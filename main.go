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

	"github.com/TheodoreRoosevelt26/Chirpy-project.git/internal/auth"
	"github.com/TheodoreRoosevelt26/Chirpy-project.git/internal/database"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

type apiConfig struct {
	fileserverHits atomic.Int32
	database       *database.Queries
	jwtKey         string
	polkaKey       string
}

type User struct {
	ID           uuid.UUID `json:"id"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	Email        string    `json:"email"`
	Token        string    `json:"token"`
	RefreshToken string    `json:"refresh_token"`
	ChirpyRed    bool      `json:"is_chirpy_red"`
}

type Chirp struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Body      string    `json:"body"`
	UserID    uuid.UUID `json:"user_id"`
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
	return cleanedSentence
}

func (cfg *apiConfig) createUser(w http.ResponseWriter, r *http.Request) {
	type userCreation struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	decoder := json.NewDecoder(r.Body)
	email := userCreation{}
	err := decoder.Decode(&email)
	if err != nil {
		respondWithError(w, 400, "Unable to process request")
		return
	}
	hashedPass, err := auth.HashPassword(email.Password)
	if err != nil {
		respondWithError(w, 400, "Unable to create user, faulty password")
		return
	}
	fmt.Printf("created hash: %v \n", hashedPass)
	params := database.CreateUserParams{
		Email:          email.Email,
		HashedPassword: hashedPass,
	}
	ctx := r.Context()
	dbUser, err := cfg.database.CreateUser(ctx, params)
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
		Body string `json:"body"`
	}
	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, 401, "Unauthorized")
		return
	}
	fromUser, err := auth.ValidateJWT(token, cfg.jwtKey)
	if err != nil {
		respondWithError(w, 401, "Unauthorized")
		return
	}
	decoder := json.NewDecoder(r.Body)
	newChirp := incomingChirp{}
	err = decoder.Decode(&newChirp)
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
		UserID: fromUser,
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

func (cfg *apiConfig) getChirps(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	author := r.URL.Query().Get("author_id")
	order := r.URL.Query().Get("sort")
	var dbChirps []database.Chirp
	var err error
	if author != "" {
		userID, err := uuid.Parse(author)
		if err != nil {
			respondWithError(w, 400, "Unable to retrieve Chirps")
		}
		if order == "desc" {
			dbChirps, err = cfg.database.GetUserChirpsDesc(ctx, userID)
			if err != nil {
				fmt.Printf("Error %v", err)
				respondWithError(w, 400, "Unable to retrieve Chirps")
				return
			}
		} else {
			dbChirps, err = cfg.database.GetUserChirps(ctx, userID)
			if err != nil {
				fmt.Printf("Error %v", err)
				respondWithError(w, 400, "Unable to retrieve Chirps")
				return
			}
		}
		var chirps []Chirp
		for _, dbChirp := range dbChirps {
			chirps = append(chirps, Chirp{
				ID:        dbChirp.ID,
				CreatedAt: dbChirp.CreatedAt,
				UpdatedAt: dbChirp.UpdatedAt,
				Body:      dbChirp.Body,
				UserID:    dbChirp.UserID,
			})
		}
		respondWithJSON(w, 200, chirps)
		return
	}
	if order == "desc" {
		dbChirps, err = cfg.database.GetChirpsDesc(ctx)
		if err != nil {
			fmt.Printf("Error %v", err)
			respondWithError(w, 400, "Unable to retrieve Chirps")
			return
		}
	} else {
		dbChirps, err = cfg.database.GetChirps(ctx)
		if err != nil {
			fmt.Printf("Error %v", err)
			respondWithError(w, 400, "Unable to retrieve Chirps")
			return
		}
	}
	var chirps []Chirp
	for _, dbChirp := range dbChirps {
		chirps = append(chirps, Chirp{
			ID:        dbChirp.ID,
			CreatedAt: dbChirp.CreatedAt,
			UpdatedAt: dbChirp.UpdatedAt,
			Body:      dbChirp.Body,
			UserID:    dbChirp.UserID,
		})
	}
	respondWithJSON(w, 200, chirps)
}

func (cfg *apiConfig) getChirp(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("chirpID"))
	if err != nil {
		fmt.Printf("Error %v", err)
		respondWithError(w, 400, "Unable to parse request")
		return
	}
	fmt.Printf("getChirp parsed ID: %v", id)
	ctx := r.Context()
	dbChirp, err := cfg.database.GetChirp(ctx, id)
	if err != nil {
		fmt.Printf("Error %v", err)
		w.WriteHeader(404)
		return
	}
	chirp := Chirp{
		ID:        dbChirp.ID,
		CreatedAt: dbChirp.CreatedAt,
		UpdatedAt: dbChirp.UpdatedAt,
		Body:      dbChirp.Body,
		UserID:    dbChirp.UserID,
	}
	respondWithJSON(w, 200, chirp)
}

func (cfg *apiConfig) login(w http.ResponseWriter, r *http.Request) {
	type loginRequest struct {
		Password string `json:"password"`
		Email    string `json:"email"`
	}
	ctx := r.Context()
	decoder := json.NewDecoder(r.Body)
	receivedLogin := loginRequest{}
	err := decoder.Decode(&receivedLogin)
	if err != nil {
		fmt.Printf("Error %v", err)
		w.WriteHeader(401)
		return
	}
	fmt.Printf("received password: %v \n received email: %v \n", receivedLogin.Password, receivedLogin.Email)
	hashedPass, err := cfg.database.PullUserPassword(ctx, receivedLogin.Email)
	if err != nil {
		fmt.Printf("Error %v \n", err)
		respondWithError(w, 401, "Incorrect email or password")
		return
	}
	fmt.Printf("pulled hashedPass: %v \n", hashedPass)
	err = auth.CheckPasswordHash(receivedLogin.Password, hashedPass)
	if err != nil {
		fmt.Printf("Error %v \n", err)
		respondWithError(w, 401, "Incorrect email or password")
		return
	}
	dbUser, err := cfg.database.GetUserFromEmail(ctx, receivedLogin.Email)
	if err != nil {
		respondWithError(w, 500, "Unable to retrieve user details")
		return
	}
	user := User{
		ID:           dbUser.ID,
		CreatedAt:    dbUser.CreatedAt,
		UpdatedAt:    dbUser.UpdatedAt,
		Email:        dbUser.Email,
		Token:        "",
		RefreshToken: "",
		ChirpyRed:    dbUser.IsChirpyRed.Bool,
	}
	user.Token, err = auth.MakeJWT(user.ID, cfg.jwtKey, time.Duration(3600)*time.Second)
	if err != nil {
		respondWithError(w, 500, "Unable to create session")
		return
	}
	RefTokStruct := database.RegisterRefreshTokenParams{
		Token:  "",
		UserID: user.ID,
	}
	RefTokStruct.Token, err = auth.MakeRefreshToken()
	if err != nil {
		respondWithError(w, 500, "Unable to create session")
		return
	}
	_, err = cfg.database.RegisterRefreshToken(ctx, RefTokStruct)
	if err != nil {
		respondWithError(w, 500, "Unable to register session")
		return
	}
	user.RefreshToken = RefTokStruct.Token
	respondWithJSON(w, 200, user)
}

func (cfg *apiConfig) refresh(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	receivedRefreshToken, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, 401, "")
		return
	}
	refreshToken, err := cfg.database.LookUpRefreshToken(ctx, receivedRefreshToken)
	if err != nil {
		respondWithError(w, 401, "")
		return
	}
	if time.Now().After(refreshToken.ExpiresAt) || refreshToken.RevokedAt.Valid {
		respondWithError(w, 401, "")
		return
	}
	type respondWithNewToken struct {
		JWT string `json:"token"`
	}
	newJWTString, err := auth.MakeJWT(refreshToken.UserID, cfg.jwtKey, time.Duration(3600)*time.Second)
	if err != nil {
		respondWithError(w, 401, "")
		return
	}
	token := respondWithNewToken{
		JWT: newJWTString,
	}
	respondWithJSON(w, 200, token)
}

func (cfg *apiConfig) revoke(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	recToken, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, 401, "")
		return
	}
	token, err := cfg.database.LookUpRefreshToken(ctx, recToken)
	if err != nil {
		respondWithError(w, 401, "")
		return
	}
	err = cfg.database.RevokeRefreshToken(ctx, token.Token)
	if err != nil {
		respondWithError(w, 401, "")
		return
	}
	w.WriteHeader(204)
}

func (cfg *apiConfig) updateUser(w http.ResponseWriter, r *http.Request) {
	type userDataChange struct {
		Password string `json:"password"`
		Email    string `json:"email"`
	}
	ctx := r.Context()
	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, 401, "")
		return
	}
	userID, err := auth.ValidateJWT(token, cfg.jwtKey)
	if err != nil {
		respondWithError(w, 401, "")
		return
	}
	receivedData := userDataChange{}
	decoder := json.NewDecoder(r.Body)
	err = decoder.Decode(&receivedData)
	if err != nil {
		respondWithError(w, 401, "")
		return
	}
	password, err := auth.HashPassword(receivedData.Password)
	if err != nil {
		respondWithError(w, 400, "Unable to change password, faulty password")
		return
	}
	updatePasswordEmailQuery := database.UpdateUserEmailPasswordParams{
		Email:          receivedData.Email,
		HashedPassword: password,
		ID:             userID,
	}
	err = cfg.database.UpdateUserEmailPassword(ctx, updatePasswordEmailQuery)
	if err != nil {
		respondWithError(w, 401, "")
		return
	}
	type respondStruct struct {
		Email string `json:"email"`
	}
	response := respondStruct{
		Email: receivedData.Email,
	}
	if receivedData.Email == "" {
		respondWithJSON(w, 200, "")
	} else {
		respondWithJSON(w, 200, response)
	}
}

func (cfg *apiConfig) deleteChirp(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, 401, "")
		return
	}
	userID, err := auth.ValidateJWT(token, cfg.jwtKey)
	if err != nil {
		respondWithError(w, 403, "")
		return
	}
	id, err := uuid.Parse(r.PathValue("chirpID"))
	if err != nil {
		respondWithError(w, 404, "")
		return
	}
	chirp, err := cfg.database.GetChirp(ctx, id)
	if err != nil {
		respondWithError(w, 404, "")
		return
	}
	if userID != chirp.UserID {
		respondWithError(w, 403, "")
		return
	}
	err = cfg.database.DeleteChirp(ctx, chirp.ID)
	if err != nil {
		respondWithError(w, 404, "")
		return
	}
	w.WriteHeader(204)
}

func (cfg *apiConfig) polkaHook(w http.ResponseWriter, r *http.Request) {
	type incomingPolkaEvent struct {
		Event string `json:"event"`
		Data  struct {
			UserID string `json:"user_id"`
		} `json:"data"`
	}
	ctx := r.Context()
	polkaVerif, err := auth.GetAPIKey(r.Header)
	if err != nil {
		w.WriteHeader(401)
		return
	}
	if polkaVerif != cfg.polkaKey {
		w.WriteHeader(401)
		return
	}
	incomingEvent := incomingPolkaEvent{}
	decoder := json.NewDecoder(r.Body)
	err = decoder.Decode(&incomingEvent)
	if err != nil {
		w.WriteHeader(404)
		return
	}
	if incomingEvent.Event == "user.upgraded" {
		toUuid, err := uuid.Parse(incomingEvent.Data.UserID)
		if err != nil {
			w.WriteHeader(404)
			return
		}
		changeRequest := database.UpgradeToRedRow{
			ID: toUuid,
		}
		_, err = cfg.database.UpgradeToRed(ctx, changeRequest.ID)
		if err != nil {
			w.WriteHeader(404)
			return
		}
		w.WriteHeader(204)
		return
	}
	w.WriteHeader(204)
}

func main() {
	godotenv.Load()
	dbURL := os.Getenv("DB_URL")
	jwtSecret := os.Getenv("SECRET_JWT_STRING")
	polkaSecret := os.Getenv("POLKA_KEY")
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		//maybe handle it better later, ignore for now.
		fmt.Println("error opening PSQL DB")
	}
	defer db.Close()
	dbQueries := database.New(db)
	apiCfg := &apiConfig{
		database: dbQueries,
		jwtKey:   jwtSecret,
		polkaKey: polkaSecret,
	}
	SM := http.NewServeMux()
	Server := &http.Server{Addr: ":8080", Handler: SM}
	fileServer := http.StripPrefix("/app", http.FileServer(http.Dir(".")))
	SM.Handle("/app/", apiCfg.middlewareMetricsInc(fileServer))
	SM.HandleFunc("GET /api/healthz", healthzHandler)
	SM.HandleFunc("GET /admin/metrics", apiCfg.metricsHandler)
	SM.HandleFunc("POST /admin/reset", apiCfg.resetHandler)
	SM.HandleFunc("POST /api/chirps", apiCfg.chirps)
	SM.HandleFunc("GET /api/chirps", apiCfg.getChirps)
	SM.HandleFunc("GET /api/chirps/{chirpID}", apiCfg.getChirp)
	SM.HandleFunc("DELETE /api/chirps/{chirpID}", apiCfg.deleteChirp)
	SM.HandleFunc("POST /api/users", apiCfg.createUser)
	SM.HandleFunc("PUT /api/users", apiCfg.updateUser)
	SM.HandleFunc("POST /api/login", apiCfg.login)
	SM.HandleFunc("POST /api/refresh", apiCfg.refresh)
	SM.HandleFunc("POST /api/revoke", apiCfg.revoke)
	SM.HandleFunc("POST /api/polka/webhooks", apiCfg.polkaHook)
	err = Server.ListenAndServe()
	if err != nil {
		log.Fatal("Error: unable to start server")
	}
}
