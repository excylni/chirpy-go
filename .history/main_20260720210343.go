package main 

import (
	"fmt"
	"net/http"
)

type apiHandler struct{}

func (apiHandler) ServeHTTP(http.ResponseWriter, *http.Request) {}
func main() {
	mux := http.NewServeMux()
	mux.Handle
}