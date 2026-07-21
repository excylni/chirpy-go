package main 

import (
	"fmt"
	"net/http"
)

type apiHandler struct{}

func (apiHandler) ServeHTTP(http)
func main() {
	mux := http.NewServeMux()
}