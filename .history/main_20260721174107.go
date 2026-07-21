package main 

import (
	"fmt"
	"net/http"
	"log"
)


func main() {
	mux := http.NewServeMux()
	
	filepathHandler := http.FileServer(http.Dir("."))
	mux.Handle("/", filepathHandler)

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