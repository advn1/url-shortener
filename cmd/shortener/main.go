package main

import (
	"fmt"
	"net/http"

	"github.com/advn1/url-shortener/internal/config"
	"github.com/advn1/url-shortener/internal/handler"
)

func main() {
	cfg := config.Parse()

	if err := cfg.Validate(); err != nil {
        fmt.Printf("Config error: %v", err)
    }

	fmt.Printf("Server configuration:\n")
    fmt.Printf("  Listening on: %s\n", cfg.Host)
    fmt.Printf("  Base URL: %s\n", cfg.BaseURL)
    fmt.Println()

	h := handler.New(cfg.BaseURL)
	mux := http.NewServeMux()

	mux.HandleFunc("/", h.HandlePost)
	mux.HandleFunc("/{id}", h.HandleGetById)

	err := http.ListenAndServe(cfg.Host, mux)
	if err != nil {
		fmt.Printf("error: %v\n", err)
	}
}