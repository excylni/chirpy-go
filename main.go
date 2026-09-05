package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"github.com/excylni/chirpy-go/internal/auth"
	"github.com/excylni/chirpy-go/internal/database"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

func main() {
	godotenv.Load()
	dbURL := os.Getenv("DB_URL")
	db, err := sql.Open("postgres", dbURL)

	dbQueries := database.New(db)

	mux := http.NewServeMux()
	apiCfg := apiConfig{
		dataBaseQueries: dbQueries,
		platform : os.Getenv("PLATFORM"),
		jwtSecret: os.Getenv("JWT_SECRET"),
	}

	filepathHandler := http.FileServer(http.Dir("."))

	handler := http.StripPrefix("/app", filepathHandler)
	mux.Handle("/app/", apiCfg.middlewareMetricsInc(handler))
	
	mux.HandleFunc("POST /admin/reset", apiCfg.resetMetrics)
	mux.HandleFunc("POST /api/users", apiCfg.handleCreateUser)
	mux.HandleFunc("POST /api/login", apiCfg.handleLogin)
	mux.HandleFunc("GET /admin/metrics", apiCfg.handleMetrics)
	mux.HandleFunc("GET /api/chirps", apiCfg.handleReturnChirps)
	mux.HandleFunc("POST /api/chirps", apiCfg.handlePostChirp)
	mux.HandleFunc("GET /api/chirps/{chirpID}", apiCfg.handleReturnChirp)
	mux.HandleFunc("POST /api/refresh", apiCfg.handleRefresh)
	mux.HandleFunc("POST /api/revoke", apiCfg.handleRevokeToken)

	// health check
	mux.HandleFunc("GET /api/healthz", func(w http.ResponseWriter, req *http.Request) {
	// writing content type header
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	server := &http.Server{
		Addr: ":8080",
		Handler: mux,
	}

	fmt.Println("Server starting on http://localhost:8080 ...")

	err = server.ListenAndServe() 
	if err != nil {
		log.Fatal("Server crashed: ", err)
		}
}

type apiConfig struct {
	fileserverHits atomic.Int32
	dataBaseQueries *database.Queries
	platform string
	jwtSecret string
}

type Chirp struct {
	ID uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Body string `json:"body"`
	User_id uuid.UUID `json:"user_id"`
}

type User struct {
	ID uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Email string  `json:"email"`
}

func (cfg *apiConfig) handleReturnChirp (w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	encoder := json.NewEncoder(w)

	idString := r.PathValue("chirpID")
	chirpID, err := uuid.Parse(idString)
	if err != nil {
		respondWithError(w, 400, "invalid ID format")
		return
	}

	ctx := r.Context()
	dbChirp, err := cfg.dataBaseQueries.GetChirpByID(ctx, chirpID)
	if err != nil {
		respondWithError(w, 404, "not found...")
		return
	}
	
	chirp := Chirp{
		ID: dbChirp.ID,
		CreatedAt: dbChirp.CreatedAt,
		UpdatedAt: dbChirp.UpdatedAt,
		Body: dbChirp.Body,
		User_id: dbChirp.UserID,
	}

	w.WriteHeader(200)
	encoder.Encode(chirp)
	
}

func (cfg *apiConfig) handleReturnChirps (w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	encoder := json.NewEncoder(w)
	ctx := r.Context()
	sortedChirps, err := cfg.dataBaseQueries.GetChirps(ctx)

	if err != nil {
		respondWithError(w, 500, "try later")
		return
	}
	var Chirps []Chirp 

	for _, chirp := range(sortedChirps) {
		jsonChirp := Chirp{
		ID: chirp.ID,
		CreatedAt: chirp.CreatedAt,
		UpdatedAt: chirp.UpdatedAt,
		Body: chirp.Body,
		User_id: chirp.UserID,
	}

	Chirps = append(Chirps,jsonChirp)
	}

	w.WriteHeader(200)
	encoder.Encode(Chirps)

}

func (cfg *apiConfig) middlewareMetricsInc (next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits.Add(1)
		next.ServeHTTP(w, r)
	})
}

func (cfg *apiConfig) handleMetrics(w http.ResponseWriter, r *http.Request) {
	hits := cfg.fileserverHits.Load()
	fmt.Fprintf(w, "Hits: %d", hits)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)

	htmlContent := fmt.Sprintf(`
	<!DOCTYPE html>
	<html>
  		<body>
    		<h1>Welcome, Chirpy Admin</h1>
    		<p>Chirpy has been visited %d times!</p>
  		</body>
	</html>
	`, hits)

	w.Write([]byte(htmlContent))
}

func (cfg *apiConfig) resetMetrics(w http.ResponseWriter, r *http.Request) {

	if cfg.platform != "dev" {
		respondWithError(w, 403, "Forbidden")
		return
	}
	ctx := r.Context()
	err := cfg.dataBaseQueries.DeleteUsers(ctx)
	if err != nil {
		respondWithError(w, 500, "Internal Server Error, try again later")
		return
	}

	cfg.fileserverHits.Store(0)
}

func (cfg *apiConfig) handleCreateUser (w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	type parameters struct {
		Email string `json:"email"`
		Password string `json:"password"`
	}

	var req parameters
	encoder := json.NewEncoder(w)
	decoder := json.NewDecoder(r.Body)
	
	err := decoder.Decode(&req)
	if err != nil {
		respondWithError(w, 400, "failed to decode json")
		return
	}

	hashed, err := auth.HashPassword(req.Password)
	if err != nil {
		respondWithError(w, 500, "internal server error")
		return
	}

	// Create User
	ctx := r.Context()
	user, err := cfg.dataBaseQueries.CreateUser(ctx, database.CreateUserParams{
		Email: req.Email,
		HashedPassword: hashed,
	})

	if err != nil {
		respondWithError(w, 500, "internal server error")
		return 
	}

	jsonUser := User{
		ID: user.ID,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
		Email: user.Email,
	}
	
	w.WriteHeader(201)
	encoder.Encode(jsonUser)

}

func profanityfilter(body string) string {
	profanities := []string{ "kerfuffle", "sharbert", "fornax"}

	words := strings.Split(body, " ")

	for i, word := range words {
		loweredWord := strings.ToLower(word)

		for _, badWord := range profanities {
			if loweredWord == badWord {
				words[i] = "****"
			}
		}
	}

	return strings.Join(words, " ")
}

func respondWithError(w http.ResponseWriter, code int, msg string) {
	w.WriteHeader(code)
	error_msg := map[string]string{"error": msg}
	json.NewEncoder(w).Encode(error_msg)

}

func validateChirp(body string) (string, error) {
	if len(body) > 140 {
		return "", errors.New("Chirp too long")
	}
	return body, nil
}

func (cfg *apiConfig) handlePostChirp(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, 401, "missing or invalid token")
		return
	}
	
	userID, err := auth.ValidateJWT(token, cfg.jwtSecret)
	if err != nil {
		respondWithError(w, 401, "missing or invalid token")
	}

	type parameters struct {
		Body string `json:"body"`
	}

	var req parameters
	encoder := json.NewEncoder(w)
	decoder := json.NewDecoder(r.Body)

	err = decoder.Decode(&req)
	if err != nil {
		respondWithError(w, 500, "failed to decode json")
		return
	}

	// checking length
	req.Body, err = validateChirp(req.Body)
	if err != nil {
		respondWithError(w, 400, "Chirp is too long")
		return
	}

	req.Body = profanityfilter(req.Body)

	ctx := r.Context()
	newChirp, err := cfg.dataBaseQueries.CreateChirp(ctx, database.CreateChirpParams{
		Body: req.Body,
		UserID: userID,
	})

	if err != nil{
		respondWithError(w, 500, "internal server error, try later")
		return
	}

	w.WriteHeader(201)

	jsonChirp := Chirp{
		ID: newChirp.ID,
		CreatedAt: newChirp.CreatedAt,
		UpdatedAt: newChirp.UpdatedAt,
		Body: newChirp.Body,
		User_id: newChirp.UserID,
	}

	w.WriteHeader(201)
	encoder.Encode(jsonChirp)
}	

func (cfg *apiConfig) handleLogin(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	type parameters struct {
		Password string `json:"password"`
		Email string `json:"email"`
	}

	type LoginResponse struct {
		ID 	uuid.UUID `json:"id"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`
		Email string `json:"email"`
		Token string `json:"token"`
		RefreshToken string `json:"refresh_token"`
	}

	var req parameters
	encoder := json.NewEncoder(w)
	decoder := json.NewDecoder(r.Body)

	err := decoder.Decode(&req)
	if err != nil {
		respondWithError(w, 400, "failed to decode json")
		return
	}

	// token expires after one hour
	expiresIn := time.Hour
    ctx := r.Context()
	user, err := cfg.dataBaseQueries.GetUserByEmail(ctx, req.Email)
	if err != nil {
		respondWithError(w, 401, "Incorrect email or password")
		return
	}

	match, err := auth.CheckPasswordHash(req.Password, user.HashedPassword)
	if err != nil || !match {
		respondWithError(w, 401, "Incorrect email or password")
		return
	}

	token, err := auth.MakeJWT(user.ID, cfg.jwtSecret, expiresIn)
	refreshTokenStr := auth.MakeRefreshToken()
	expiresAt := time.Now().Add(60 * 24 * time.Hour)

	_, err = cfg.dataBaseQueries.CreateRefreshToken(ctx, database.CreateRefreshTokenParams{
		Token: refreshTokenStr,
		UserID: user.ID,
		ExpiresAt: expiresAt,
	})

	if err != nil {
		respondWithError(w, 500, "Couldn't save refresh token")
		return
	}

	response := LoginResponse{
		ID: user.ID,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
		Email: user.Email,
		Token: token,
		RefreshToken: refreshTokenStr,
	}

	w.WriteHeader(200)
	encoder.Encode(response)
}

func (cfg *apiConfig) handleRefresh(w http.ResponseWriter, r *http.Request) {
	encoder := json.NewEncoder(w)
	refreshTokenStr, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, 401, "missing or invalid token")
		return
	}

	// searching token in the database
	ctx := r.Context()
	refreshToken, err := cfg.dataBaseQueries.GetUserFromRefreshToken(ctx, refreshTokenStr)
	if err != nil {
		respondWithError(w, 401, "invalid refresh token")
		return
	}

	// check if token is expired 
	if refreshToken.ExpiresAt.Before(time.Now()) {
		respondWithError(w, 401, "refresh token expired")
		return
	}

	// check if token is revoked
	if refreshToken.RevokedAt.Valid {
		respondWithError(w, 401, "refresh token invalid")
		return
	}

	// create a new access token
	accessToken, err := auth.MakeJWT(refreshToken.UserID, cfg.jwtSecret, time.Hour)
	if err != nil {
		respondWithError(w, 500, "Could not create access token")
	}

	type response struct {
		Token string `json:"token"`
	}

	tokenresponse := response {
		Token: accessToken,
	}

	w.WriteHeader(200)
	encoder.Encode(tokenresponse)
}

func (cfg *apiConfig) handleRevokeToken(w http.ResponseWriter, r *http.Request) {
	refreshTokenStr, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, 401, "missing or invalid token")
		return
	}

	// searching token in the database
	ctx := r.Context()
	err = cfg.dataBaseQueries.RevokeRefreshToken(ctx, refreshTokenStr)
	if err != nil {
		respondWithError(w, 501, "internal server error")
		return
	}

	w.WriteHeader(204)
}