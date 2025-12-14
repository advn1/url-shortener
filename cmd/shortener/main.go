package main

import (
	"bufio"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"os"

	"github.com/advn1/url-shortener/internal/config"
	"github.com/advn1/url-shortener/internal/handler"
	"github.com/advn1/url-shortener/internal/middleware"
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
	if err := cfg.Validate(); err != nil {
		sugar.Fatalw("Config validation error", "error", err)
	}

	// choose where to store data
	var urlsMap map[string]string = map[string]string{}
	var db *sql.DB

	if cfg.DatabaseDSN != "" {
		sugar.Infow("Storage mode: Database")
		db = initDB(cfg.DatabaseDSN, sugar)
		defer db.Close()
	} else if cfg.FileStoragePath != "" {
		sugar.Infow("Storage mode: File")
		urlsMap = initUrlsMap(cfg.FileStoragePath, sugar)
	} else {
		sugar.Infow("Storage mode: In-memory")
		// nothing happens. urlsMap is already initialized
	}

	// init handler and mux
	h := handler.New(cfg.BaseURL, urlsMap, cfg.FileStoragePath, db, sugar)
	mux := http.NewServeMux()

	// register endpoints
	mux.HandleFunc("/", h.HandlePost)
	mux.HandleFunc("/{id}", h.HandleGetById)
	mux.HandleFunc("/api/shorten", h.HandlePostRESTApi)
	mux.HandleFunc("/api/shorten/batch", h.HandlePostBatchRESTApi)
	mux.HandleFunc("/ping", h.PingBD)

	// create a middlewared-handler
	handler := middleware.GzipMiddleware(middleware.LoggingMiddleware(mux, sugar))

	// start listening
	sugar.Infow("Starting server", "address", cfg.ServerAddr, "base URL", cfg.BaseURL)
	err = http.ListenAndServe(cfg.ServerAddr, handler)
	if err != nil {
		sugar.Fatalw("Starting server", "error", err)
	}
}

func loadFromFile(filename string) (map[string]string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return map[string]string{}, err
	}

	defer file.Close()

	scanner := bufio.NewScanner(file)
	urlsMap := make(map[string]string, 10)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var jsonLine handler.ShortURL

		err := json.Unmarshal(line, &jsonLine)
		if err != nil {
			return map[string]string{}, err
		}

		urlsMap[jsonLine.ShortURL] = jsonLine.OriginalURL
	}

	return urlsMap, nil
}

func initUrlsMap(fileStoragePath string, sugar *zap.SugaredLogger) map[string]string {
	urlsMap, err := loadFromFile(fileStoragePath)
	if err != nil {
		sugar.Fatalw("Loading file error", "error", err)
	}
	return urlsMap
}

func initDB(dsn string, sugar *zap.SugaredLogger) *sql.DB {
	// init db
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		sugar.Fatalw("cannot open db connection", "error", err)
	}

	// check connection
	if err = db.Ping(); err != nil {
		sugar.Fatalw("cannot ping db", "error", err)
	}

	// create table
	_, err = db.ExecContext(context.Background(), `CREATE TABLE IF NOT EXISTS urls (
	id serial PRIMARY KEY,
	uuid CHAR(36) UNIQUE,
	original_url VARCHAR(100) NOT NULL UNIQUE,
	short_url VARCHAR(100) NOT NULL UNIQUE
	)`)
	if err != nil {
		sugar.Fatalw("cannot init db table", "error", err)
	}
	return db
}
