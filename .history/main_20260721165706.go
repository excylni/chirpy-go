package main 

import (
	"fmt"
	"net/http"
)


func main() {
	mux := http.NewServeMux()

	server := &http.Server{
		Addr: ":8080",
		Handler: mux,
	}


	fmt.Println("Server starting")
	err := server.ListenAndServe()
	if err != nil {
		log.Fatal("Server crashed: ", err)

}