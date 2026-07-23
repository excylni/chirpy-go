package main 

import (
	"fmt"
	"net/http"
	"log"
	"sync/atomic"
)

type apiConfig struct {
	fileserverHits atomic.Int32
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
}

func (cfg *apiConfig) resetMetrics(w http.ResponseWriter, r *http.Request) {
	cfg.fileserverHits.Store(0)
}

func main() {
	mux := http.NewServeMux()
	apiCfg := apiConfig{}
	
	
	filepathHandler := http.FileServer(http.Dir("."))

	handler := http.StripPrefix("/app", filepathHandler)
	mux.Handle("/app/", apiCfg.middlewareMetricsInc(handler))
	
	mux.HandleFunc("POST /reset", apiCfg.resetMetrics)
	mux.HandleFunc("GET /metrics", apiCfg.handleMetrics)

	// health check
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, req *http.Request) {
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
	err := server.ListenAndServe()
	if err != nil {
		log.Fatal("Server crashed: ", err)
		}
}