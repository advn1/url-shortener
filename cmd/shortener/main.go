package main

import (
	"net/http"

	"github.com/advn1/url-shortener/internal/config"
	"github.com/advn1/url-shortener/internal/handler"
	"github.com/advn1/url-shortener/internal/middleware"
	"go.uber.org/zap"
)

func main() {
	// init logger
	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}
	defer logger.Sync()

	// logger wrapper. provides more ergonomic API
	sugar := logger.Sugar()

	// parse application config and validate it
	cfg := config.Parse()
	if err := cfg.Validate(); err != nil {
		sugar.Fatalw("Config error", "error", err)
	}

	// init handler and mux
	h := handler.New(cfg.BaseURL, sugar)
	mux := http.NewServeMux()

	// added logging middleware
	mux.HandleFunc("/", middleware.LoggingMiddleware(h.HandlePost, sugar))
	mux.HandleFunc("/{id}", middleware.LoggingMiddleware(h.HandleGetById, sugar))
	mux.Handle("/api/shorten", middleware.LoggingMiddleware(h.HandlePostRESTApi, sugar))
	// start listening
	sugar.Infow("Starting server", "address", cfg.Host, "base URL", cfg.BaseURL)
	err = http.ListenAndServe(cfg.Host, mux)
	if err != nil {
		sugar.Fatalw("Starting server", "error", err)
	}
}
