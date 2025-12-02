package main

import (
	"flag"
	"fmt"
	"net/http"
	"strings"

	"github.com/advn1/url-shortener/internal/config"
	"github.com/advn1/url-shortener/internal/handler"
)

func main() {
	mux := http.NewServeMux()
	cfg := config.Config {}

	var host string
	flag.StringVar(&host, "a", "localhost:8080", "address of HTTP-server (shorthand)")
	flag.StringVar(&host, "address", "localhost:8080", "address of HTTP-server")

	var baseURL string
	flag.StringVar(&baseURL, "b", "http://localhost:8080", "base address of shortened URL (shorthand)")
	flag.StringVar(&baseURL, "base-url", "http://localhost:8080", "base address of shortened URL")
	
	flag.Parse()

	if !strings.HasPrefix(baseURL, "https://") && !strings.HasPrefix(baseURL, "http://") {
		fmt.Println("base address of shortened URL must start with http:// or https://")
		return
	}
	baseURL = strings.TrimSuffix(baseURL, "/")

	cfg.Host = host
	cfg.BaseURL = baseURL

	h := handler.New(cfg.BaseURL)

	mux.HandleFunc("/", h.HandlePost)
	mux.HandleFunc("/{id}", h.HandleGetById)

	fmt.Println("listening to", cfg.Host)
	err := http.ListenAndServe(cfg.Host, mux)
	if err != nil {
		fmt.Printf("error: %v\n", err)
	}
}