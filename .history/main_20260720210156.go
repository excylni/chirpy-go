package main 

import (
	"fmt"
	"net/http"
)

type apiHandler struct{}

func (apiHandler) ServeHTTP(http.Response)
func main() {
	mux := http.NewServeMux()
}