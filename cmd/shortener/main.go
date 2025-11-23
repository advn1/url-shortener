package main

import (
	"fmt"
	"net/http"

	"github.com/advn1/url-shortener/internal/handler"
)

func main() {
	mux := http.NewServeMux()

	mux.HandleFunc("/", handler.HandlePost)
	mux.HandleFunc("/{id}", handler.HandleGetById)
	err := http.ListenAndServe("localhost:8080", mux)
	if err != nil {
		fmt.Printf("error: %v\n", err)
	}
}