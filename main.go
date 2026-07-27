package main 

import (
	"fmt"
	"net/http"
	"log"
	"sync/atomic"
	"encoding/json"
	"strings"
	"github.com/joho/godotenv"
	"os"
	_ "github.com/lib/pq"
	"database/sql"
	"github.com/excylni/chirpy-go/internal/database"
)

type apiConfig struct {
	fileserverHits atomic.Int32
	dataBaseQueries *database.Queries
}

type Chirp struct {
	Body string `json:"body"`
	Error string `json:"error,omitempty"`
	Cleaned_body string `json:"cleaned_body"`
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
	cfg.fileserverHits.Store(0)
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


func handlerValidateChirp(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	var req Chirp
	encoder := json.NewEncoder(w)
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&req)

	if err != nil {
		w.WriteHeader(500)
		encoder.Encode(Chirp{Error: "Something went wrong"})
		return
	}

	// checking length
	if len(req.Body) > 140 {
		w.WriteHeader(http.StatusBadRequest)
		encoder.Encode(Chirp{Error: "Chirp is too long"})
		return
	}

	cleaned_body := profanityfilter(req.Body)

	w.WriteHeader(200)
	encoder.Encode(Chirp{Cleaned_body: cleaned_body})

	}	

func main() {
	fmt.Print("Hello World!")
	godotenv.Load()
	dbURL := os.Getenv("DB_URL")
	db, err := sql.Open("postgres", dbURL)

	dbQueries := database.New(db)

	mux := http.NewServeMux()
	apiCfg := apiConfig{
		dataBaseQueries: dbQueries,
	}

	filepathHandler := http.FileServer(http.Dir("."))

	handler := http.StripPrefix("/app", filepathHandler)
	mux.Handle("/app/", apiCfg.middlewareMetricsInc(handler))
	
	mux.HandleFunc("POST /api/validate_chirp", handlerValidateChirp)
	mux.HandleFunc("POST /admin/reset", apiCfg.resetMetrics)
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

	//fmt.Println("Server starting on http://localhost:8080 ...")

	err = server.ListenAndServe() 
	if err != nil {
		log.Fatal("Server crashed: ", err)
		}
}