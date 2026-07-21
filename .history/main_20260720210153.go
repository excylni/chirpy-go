package main 

import (
	"fmt"
	"net/http"
)

type apiHandler struct{}

func (apiHandler) ServeHTTP(http.Rens)
func main() {
	mux := http.NewServeMux()
}