package main

import (
	"context"
	"database/sql"
	"net/http"
	"time"

	"github.com/advn1/url-shortener/internal/config"
	"github.com/advn1/url-shortener/internal/handler"
	"github.com/advn1/url-shortener/internal/middleware"
	"github.com/advn1/url-shortener/internal/repository"
	_ "github.com/jackc/pgx/v5/stdlib"
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
	if err = cfg.Validate(); err != nil {
		sugar.Fatalw("Config validation error", "error", err)
	}

	// choose where to store data
	var repo repository.Storage

	if cfg.DatabaseDSN != "" {
		sugar.Infow("Storage mode: Database")
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
		db := initDB(ctx, cfg.DatabaseDSN, sugar)
		defer cancel()

		repo = repository.DatabaseStorage{DB: db}
	} else if cfg.FileStoragePath != "" {
		sugar.Infow("Storage mode: File")

		fileStorage, err := repository.InitFileStorage(cfg.FileStoragePath)
		if err != nil {
			sugar.Fatalw("Failed initializing file storage", "error", err)
		}

		defer fileStorage.Close()

		repo = fileStorage
	} else {
		sugar.Infow("Storage mode: In-memory")
		repo = repository.InitInMemoryStorage()
	}

	// init handler and mux
	h := handler.New(cfg.BaseURL, repo, sugar)
	mux := http.NewServeMux()

	// register endpoints
	mux.HandleFunc("/{id}", h.HandleGetById)
	mux.HandleFunc("/api/shorten", h.HandlePostRESTApi)
	mux.HandleFunc("/api/shorten/batch", h.HandlePostBatchRESTApi)
	mux.HandleFunc("/ping", h.PingDB)

	// create a middlewared-handler
	middlewaredHandler := middleware.GzipMiddleware(mux)
	middlewaredHandler = middleware.AuthMiddleware("secretkey")(middlewaredHandler)
	middlewaredHandler = middleware.LoggingMiddleware(middlewaredHandler, sugar)

	// create http.Server
	server := &http.Server{
		Handler:           middlewaredHandler,
		Addr:              cfg.ServerAddr,
		ReadTimeout:       5 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       15 * time.Second,
		ReadHeaderTimeout: 3 * time.Second,
	}

	// start listening
	sugar.Infow("Starting server", "address", cfg.ServerAddr, "base URL", cfg.BaseURL)
	err = server.ListenAndServe()
	if err != nil {
		sugar.Fatalw("Starting server", "error", err)
	}
}

func initDB(ctx context.Context, dsn string, sugar *zap.SugaredLogger) *sql.DB {
	// init db
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		sugar.Fatalw("cannot open db connection", "error", err)
	}

	// check connection
	if err = db.PingContext(ctx); err != nil {
		sugar.Fatalw("cannot ping db", "error", err)
	}

	// create table
	_, err = db.ExecContext(context.Background(), `CREATE TABLE IF NOT EXISTS urls (
	id serial PRIMARY KEY,
	uuid CHAR(36) UNIQUE,
	original_url VARCHAR(100) NOT NULL UNIQUE,
	short_url VARCHAR(100) NOT NULL UNIQUE,
	user_id CHAR(36)
	)`)
	if err != nil {
		sugar.Fatalw("cannot init db table", "error", err)
	}
	return db
}
