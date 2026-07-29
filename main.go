package main 

import (
	"fmt"
	"net/http"
	"log"
	"sync/atomic"
	"encoding/json"
	"strings"
	"github.com/joho/godotenv"
	"github.com/google/uuid"
	"os"
	_ "github.com/lib/pq"
	"database/sql"
	"github.com/excylni/chirpy-go/internal/database"
	"time"
	"errors"
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
	}

	filepathHandler := http.FileServer(http.Dir("."))

	handler := http.StripPrefix("/app", filepathHandler)
	mux.Handle("/app/", apiCfg.middlewareMetricsInc(handler))
	
	mux.HandleFunc("POST /api/chirps", apiCfg.handlePostChirp)
	mux.HandleFunc("POST /admin/reset", apiCfg.resetMetrics)
	mux.HandleFunc("POST /api/users", apiCfg.handleCreateUser)
	mux.HandleFunc("GET /admin/metrics", apiCfg.handleMetrics)

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
	}

	var req parameters
	encoder := json.NewEncoder(w)
	decoder := json.NewDecoder(r.Body)
	
	err := decoder.Decode(&req)
	if err != nil {
		respondWithError(w, 400, "failed to decode json")
		return
	}

	// Create User
	ctx := r.Context()
	user, err := cfg.dataBaseQueries.CreateUser(ctx, req.Email)
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
	
	type parameters struct {
		Body string `json:"body"`
		User_id uuid.UUID `json:"user_id"`	
	}

	var req parameters
	encoder := json.NewEncoder(w)
	decoder := json.NewDecoder(r.Body)

	err := decoder.Decode(&req)
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
		UserID: req.User_id,

	})

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

