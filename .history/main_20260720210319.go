package main 

import (
	"fmt"
	"net/http"
)

type apiHandler struct{}

func (apiHandler) ServeHTTP(http.ResponseWriter)
func main() {
	mux := http.NewServeMux()
}